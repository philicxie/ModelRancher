package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ml-platform/internal/executor/provider"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"

	"github.com/google/uuid"
)

// metricsSnapshot 单个时间点的指标快照
type metricsSnapshot struct {
	Timestamp int64
	CPU       float64
	Mem       float64
	Disk      float64
	GPUAvg    float64
	GPUMemAvg float64
	GPUs      map[string]gpuSnapshot
}

type gpuSnapshot struct {
	Util    float64
	MemUtil float64
}

// metricsBuffer 单个实例的指标环形缓存（最多保留 120 个点 ≈ 30 分钟 @15s）
const metricsMaxPoints = 120

type metricsBuffer struct {
	mu        sync.RWMutex
	points    []metricsSnapshot
	instID    string // provider instance id
	provider  string
}

func (b *metricsBuffer) append(s metricsSnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.points = append(b.points, s)
	if len(b.points) > metricsMaxPoints {
		b.points = b.points[len(b.points)-metricsMaxPoints:]
	}
}

func (b *metricsBuffer) getRange(startTime, endTime int64) []metricsSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.points) == 0 {
		return nil
	}
	// 默认返回全部
	if startTime == 0 && endTime == 0 {
		return append([]metricsSnapshot(nil), b.points...)
	}
	if endTime == 0 {
		endTime = time.Now().Unix()
	}
	var res []metricsSnapshot
	for _, p := range b.points {
		if p.Timestamp >= startTime && p.Timestamp <= endTime {
			res = append(res, p)
		}
	}
	return res
}

// InstanceService 实例租赁服务
type InstanceService struct {
	repo           *repository.InstanceRepository
	providerManager *provider.ProviderManager
	metricsCache   map[string]*metricsBuffer // key: instance.ID (our system id)
	metricsMu      sync.RWMutex
}

// NewInstanceService 创建实例服务
func NewInstanceService(repo *repository.InstanceRepository, pm *provider.ProviderManager) *InstanceService {
	svc := &InstanceService{
		repo:            repo,
		providerManager: pm,
		metricsCache:    make(map[string]*metricsBuffer),
	}
	// 启动自动释放定时器
	go svc.startAutoReleaseLoop()
	// 启动 metrics 轮询
	go svc.startMetricsPollingLoop()
	return svc
}

// CreateOffer 创建租赁（从可用offer创建实例）
func (s *InstanceService) CreateOffer(ctx context.Context, req *model.CreateOfferRequest) (*model.Instance, error) {
	if s.providerManager == nil {
		return nil, fmt.Errorf("provider manager not initialized")
	}

	p, ok := s.providerManager.GetProvider(req.Provider)
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", req.Provider)
	}

	// 调用Provider创建实例
	instInfo, err := p.CreateInstance(ctx, &provider.CreateInstanceRequest{
		OfferID:  req.OfferID,
		Image:    req.Image,
		DiskSize: req.DiskSize,
		NumGPUs:  req.NumGPUs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance on provider: %w", err)
	}

	// 创建数据库记录
	instance := &model.Instance{
		ID:             uuid.New().String(),
		Provider:       req.Provider,
		ProviderInstID: instInfo.InstanceID,
		OfferID:        req.OfferID,
		UserID:         req.UserID,
		Name:           req.Name,
		Image:          req.Image,
		Status:         model.InstanceStatusCreating,
		SSHHost:        instInfo.SSHHost,
		SSHPort:        instInfo.SSHPort,
		SSHUser:        instInfo.SSHUser,
		Password:       instInfo.Password,
		SSHCommand:     instInfo.SSHCommand,
		DiskSize:       req.DiskSize,
		PricePerHour:   req.PricePerHour,
		GPUName:        req.GPUName,
		NumGPUs:        req.NumGPUs,
		Location:       req.Location,
		CreatedAt:      time.Now(),
	}

	if req.DurationHours > 0 {
		t := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		instance.ExpiresAt = &t
	}

	if err := s.repo.Create(ctx, instance); err != nil {
		// 创建记录失败，尝试销毁远程实例
		if destroyErr := p.DestroyInstance(ctx, instInfo.InstanceID); destroyErr != nil {
			log.Printf("[instance] Failed to destroy remote instance after DB error: %v", destroyErr)
		}
		return nil, err
	}

	log.Printf("[instance] Created offer %s on provider %s (remote: %s)", instance.ID, req.Provider, instInfo.InstanceID)

	// 异步轮询等待实例就绪并更新SSH信息
	go s.pollInstanceReady(instance.ID, req.Provider, instInfo.InstanceID)

	return instance, nil
}

// pollInstanceReady 轮询等待实例就绪
func (s *InstanceService) pollInstanceReady(instanceID, providerName, providerInstID string) {
	ctx := context.Background()
	p, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		log.Printf("[instance] Provider %s not found for polling", providerName)
		return
	}

	for i := 0; i < 40; i++ {
		time.Sleep(15 * time.Second)

		info, err := p.GetInstance(ctx, providerInstID)
		if err != nil {
			log.Printf("[instance] Poll %s: get instance error: %v", instanceID, err)
			continue
		}

		// 更新数据库（provider 返回空值时不覆盖已有记录）
		if info.Status == "running" {
			now := time.Now()
			log.Printf("[instance] Instance %s is running, SSH: %s@%s:%d", instanceID, info.SSHUser, info.SSHHost, info.SSHPort)
			if err := s.repo.UpdateStartedAt(ctx, instanceID, &now); err != nil {
				log.Printf("[instance] Failed to update started_at for %s: %v", instanceID, err)
			}
		}

		result := s.repo.UpdateStatus(ctx, instanceID, model.InstanceStatus(info.Status))
		if result != nil {
			log.Printf("[instance] Failed to update status for %s: %v", instanceID, result)
		}
		// 只更新非空的 SSH 字段
		if info.SSHHost != "" || info.Password != "" || info.SSHCommand != "" {
			if err := s.repo.UpdateSSH(ctx, instanceID, info.SSHHost, info.SSHPort, info.SSHUser, info.Password, info.SSHCommand); err != nil {
				log.Printf("[instance] Failed to update SSH for %s: %v", instanceID, err)
			}
		}

		if info.Status == "running" {
			break
		}
	}
}

// GetInstance 获取实例详情（同步 provider 真实状态）
func (s *InstanceService) GetInstance(ctx context.Context, id string) (*model.Instance, error) {
	inst, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.syncInstanceStatus(ctx, inst)
	return inst, nil
}

// GetInstanceMetrics 获取实例监控指标（从内存缓存读取）
func (s *InstanceService) GetInstanceMetrics(ctx context.Context, id string, startTime, endTime int64) (*provider.InstanceMetrics, error) {
	s.metricsMu.RLock()
	buf, ok := s.metricsCache[id]
	s.metricsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no metrics data available for instance %s", id)
	}

	points := buf.getRange(startTime, endTime)
	if len(points) == 0 {
		return nil, fmt.Errorf("no metrics data in specified time range")
	}

	result := &provider.InstanceMetrics{}
	for _, p := range points {
		result.CPUUtilization = append(result.CPUUtilization, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.CPU,
		})
		result.MemUtilization = append(result.MemUtilization, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.Mem,
		})
		result.RootDiskUtilization = append(result.RootDiskUtilization, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.Disk,
		})
		result.GPUUtilizationAvg = append(result.GPUUtilizationAvg, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.GPUAvg,
		})
		result.GPUMemUtilizationAvg = append(result.GPUMemUtilizationAvg, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.GPUMemAvg,
		})
		// per-GPU 数据：这里只保存 avg，如果需要 per-GPU 时序后续可扩展
		if len(result.GPUUtilization) == 0 {
			result.GPUUtilization = append(result.GPUUtilization, provider.GPUInstanceMetrics{
				GPUID: "0", Items: []provider.MetricPoint{},
			})
			result.GPUMemUtilization = append(result.GPUMemUtilization, provider.GPUInstanceMetrics{
				GPUID: "0", Items: []provider.MetricPoint{},
			})
		}
		result.GPUUtilization[0].Items = append(result.GPUUtilization[0].Items, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.GPUAvg,
		})
		result.GPUMemUtilization[0].Items = append(result.GPUMemUtilization[0].Items, provider.MetricPoint{
			Timestamp: p.Timestamp, Value: p.GPUMemAvg,
		})
	}
	return result, nil
}

// startMetricsPollingLoop 定期轮询所有运行中实例的 metrics 并缓存
func (s *InstanceService) startMetricsPollingLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		s.pollAllRunningMetrics(ctx)
		cancel()
	}
}

func (s *InstanceService) pollAllRunningMetrics(ctx context.Context) {
	if s.providerManager == nil {
		log.Printf("[metrics] Skip polling: providerManager is nil")
		return
	}

	// 获取所有非终态实例（包括 running 和 stopped，provider 可能有数据）
	instances, err := s.repo.ListActive(ctx)
	if err != nil {
		log.Printf("[metrics] Failed to list instances for polling: %v", err)
		return
	}

	pollCount := 0
	for _, inst := range instances {
		if inst.Status != model.InstanceStatusRunning && inst.Status != model.InstanceStatusCreating {
			continue
		}
		if inst.ProviderInstID == "" {
			continue
		}
		p, ok := s.providerManager.GetProvider(inst.Provider)
		if !ok {
			log.Printf("[metrics] Provider %q not found for instance %s", inst.Provider, inst.ID)
			continue
		}
		pollCount++

		// 异步获取每个实例的 metrics，避免一个 provider 慢阻塞整体
		go func(instID, providerInstID string, prov provider.ExecutorProvider) {
			pCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// 请求最近 60 秒的数据，默认 15 秒间隔
			endTime := time.Now().Unix()
			startTime := endTime - 60
			log.Printf("[metrics] Polling %s (provider=%s, providerInstID=%s) range=[%d, %d]", instID, inst.Provider, providerInstID, startTime, endTime)
			m, err := prov.GetInstanceMetrics(pCtx, providerInstID, startTime, endTime)
			if err != nil {
				log.Printf("[metrics] Range query failed for %s: %v, trying fallback", instID, err)
				// Vastai 等可能不支持时间范围，fallback 获取当前快照
				m, err = prov.GetInstanceMetrics(pCtx, providerInstID, 0, 0)
				if err != nil {
					log.Printf("[metrics] Fallback query also failed for %s: %v", instID, err)
					return
				}
			}
			log.Printf("[metrics] Got metrics for %s: cpu=%d, mem=%d, disk=%d, gpuAvg=%d points",
				instID, len(m.CPUUtilization), len(m.MemUtilization), len(m.RootDiskUtilization), len(m.GPUUtilizationAvg))
			s.mergeMetrics(instID, m)
		}(inst.ID, inst.ProviderInstID, p)
	}
	log.Printf("[metrics] Polling loop started for %d instances", pollCount)
}

func (s *InstanceService) mergeMetrics(instID string, m *provider.InstanceMetrics) {
	now := time.Now().Unix()

	// 从 provider 返回的时序数据中提取各指标
	// 优先使用最后一条数据作为当前快照
	var cpuVal, memVal, diskVal, gpuVal, gpuMemVal float64
	if len(m.CPUUtilization) > 0 {
		cpuVal = m.CPUUtilization[len(m.CPUUtilization)-1].Value
	}
	if len(m.MemUtilization) > 0 {
		memVal = m.MemUtilization[len(m.MemUtilization)-1].Value
	}
	if len(m.RootDiskUtilization) > 0 {
		diskVal = m.RootDiskUtilization[len(m.RootDiskUtilization)-1].Value
	}
	if len(m.GPUUtilizationAvg) > 0 {
		gpuVal = m.GPUUtilizationAvg[len(m.GPUUtilizationAvg)-1].Value
	}
	if len(m.GPUMemUtilizationAvg) > 0 {
		gpuMemVal = m.GPUMemUtilizationAvg[len(m.GPUMemUtilizationAvg)-1].Value
	}

	log.Printf("[metrics] mergeMetrics for %s: cpuVal=%.1f, memVal=%.1f, diskVal=%.1f, gpuVal=%.1f, gpuMemVal=%.1f, cpuPoints=%d",
		instID, cpuVal, memVal, diskVal, gpuVal, gpuMemVal, len(m.CPUUtilization))

	// 对于 PPIO 返回的多条时序数据，逐条追加
	if len(m.CPUUtilization) > 1 {
		for i := range m.CPUUtilization {
			ts := m.CPUUtilization[i].Timestamp
			if ts == 0 {
				ts = now - int64(len(m.CPUUtilization)-1-i)*15
			}
			cpu := m.CPUUtilization[i].Value
			mem := 0.0
			if i < len(m.MemUtilization) {
				mem = m.MemUtilization[i].Value
			}
			disk := 0.0
			if i < len(m.RootDiskUtilization) {
				disk = m.RootDiskUtilization[i].Value
			}
			gpu := 0.0
			if i < len(m.GPUUtilizationAvg) {
				gpu = m.GPUUtilizationAvg[i].Value
			}
			gpuMem := 0.0
			if i < len(m.GPUMemUtilizationAvg) {
				gpuMem = m.GPUMemUtilizationAvg[i].Value
			}
			s.appendMetricsSnapshot(instID, ts, cpu, mem, disk, gpu, gpuMem)
		}
	} else {
		// 单条快照直接追加
		s.appendMetricsSnapshot(instID, now, cpuVal, memVal, diskVal, gpuVal, gpuMemVal)
	}

	// 打印缓存大小
	s.metricsMu.RLock()
	buf, ok := s.metricsCache[instID]
	s.metricsMu.RUnlock()
	if ok {
		buf.mu.RLock()
		count := len(buf.points)
		buf.mu.RUnlock()
		log.Printf("[metrics] Cache for %s now has %d points", instID, count)
	}
}

func (s *InstanceService) appendMetricsSnapshot(instID string, ts int64, cpu, mem, disk, gpu, gpuMem float64) {
	s.metricsMu.Lock()
	buf, ok := s.metricsCache[instID]
	if !ok {
		buf = &metricsBuffer{}
		s.metricsCache[instID] = buf
	}
	s.metricsMu.Unlock()

	buf.append(metricsSnapshot{
		Timestamp: ts,
		CPU:       cpu,
		Mem:       mem,
		Disk:      disk,
		GPUAvg:    gpu,
		GPUMemAvg: gpuMem,
	})
}

// ListUserInstances 列出用户实例（同步 provider 真实状态）
func (s *InstanceService) ListUserInstances(ctx context.Context, userID string) ([]*model.Instance, error) {
	instances, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, inst := range instances {
		s.syncInstanceStatus(ctx, inst)
	}
	return instances, nil
}

// ListWorkOrders 列出历史工单（含总开销计算）
func (s *InstanceService) ListWorkOrders(ctx context.Context, userID string) ([]*model.WorkOrder, error) {
	instances, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	orders := make([]*model.WorkOrder, 0, len(instances))
	now := time.Now()

	for _, inst := range instances {
		order := &model.WorkOrder{
			Instance: *inst,
		}

		// 计算运行时长和总开销
		if inst.StartedAt != nil {
			var endTime time.Time
			switch inst.Status {
			case model.InstanceStatusDestroyed:
				if inst.DestroyedAt != nil {
					endTime = *inst.DestroyedAt
				} else if inst.StoppedAt != nil {
					endTime = *inst.StoppedAt
				} else {
					endTime = now
				}
			case model.InstanceStatusStopped, model.InstanceStatusFailed:
				if inst.StoppedAt != nil {
					endTime = *inst.StoppedAt
				} else {
					endTime = now
				}
			default:
				// running, starting, creating 等活跃状态
				endTime = now
			}

			duration := endTime.Sub(*inst.StartedAt)
			if duration < 0 {
				duration = 0
			}
			order.DurationHours = duration.Hours()
			order.TotalCost = order.DurationHours * inst.PricePerHour
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// syncInstanceStatus 同步单个实例的 provider 端真实状态
func (s *InstanceService) syncInstanceStatus(ctx context.Context, inst *model.Instance) {
	log.Printf("[instance][SYNC] Start sync for %s (provider=%s, provider_inst_id=%s, status=%s)",
		inst.ID, inst.Provider, inst.ProviderInstID, inst.Status)

	// 已销毁实例无需同步
	if inst.Status == model.InstanceStatusDestroyed {
		log.Printf("[instance][SYNC] Skip %s: already destroyed", inst.ID)
		return
	}
	if s.providerManager == nil {
		log.Printf("[instance][SYNC] Skip %s: providerManager is nil", inst.ID)
		return
	}
	p, ok := s.providerManager.GetProvider(inst.Provider)
	if !ok {
		log.Printf("[instance][SYNC] Skip %s: provider %q not found in manager", inst.ID, inst.Provider)
		return
	}

	// 兜底：provider_inst_id 为空，说明之前字段解析失败，直接标记为 failed
	if inst.ProviderInstID == "" {
		log.Printf("[instance][SYNC] %s has empty ProviderInstID, marking as failed", inst.ID)
		inst.Status = model.InstanceStatusFailed
		inst.ErrorMsg = "provider instance ID is empty, possibly due to previous field parsing bug"
		if dbErr := s.repo.Update(ctx, inst); dbErr != nil {
			log.Printf("[instance][SYNC] Failed to update failed status for %s: %v", inst.ID, dbErr)
		}
		return
	}

	log.Printf("[instance][SYNC] Querying provider %s for instance %s", inst.Provider, inst.ProviderInstID)
	info, err := p.GetInstance(ctx, inst.ProviderInstID)
	if err != nil {
		log.Printf("[instance][SYNC] Provider %s query error for %s: %v", inst.Provider, inst.ProviderInstID, err)
		// provider 查不到实例（或返回 4XX），可能已被回收：标记为 destroyed
		errMsg := err.Error()
		isNotFound := strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "404") ||
			strings.Contains(errMsg, "400") ||
			strings.Contains(errMsg, "401") ||
			strings.Contains(errMsg, "403") ||
			strings.Contains(errMsg, "No such container")
		if isNotFound {
			log.Printf("[instance][SYNC] Instance %s not found on provider %s (err: %s), marking destroyed", inst.ID, inst.Provider, errMsg)
			now := time.Now()
			inst.Status = model.InstanceStatusDestroyed
			inst.DestroyedAt = &now
			if dbErr := s.repo.Update(ctx, inst); dbErr != nil {
				log.Printf("[instance][SYNC] Failed to update destroyed status for %s: %v", inst.ID, dbErr)
			}
		}
		return
	}
	log.Printf("[instance][SYNC] Provider %s returned status %s for %s", inst.Provider, info.Status, inst.ProviderInstID)
	log.Printf("[instance][SYNC] Current DB status for %s: %s, provider status: %s", inst.ID, inst.Status, info.Status)

	// 更新状态
	newStatus := model.InstanceStatus(info.Status)
	if newStatus != inst.Status {
		log.Printf("[instance] Sync status for %s: %s -> %s", inst.ID, inst.Status, newStatus)
		inst.Status = newStatus
		if dbErr := s.repo.UpdateStatus(ctx, inst.ID, newStatus); dbErr != nil {
			log.Printf("[instance] Failed to sync status for %s: %v", inst.ID, dbErr)
		}
	} else {
		log.Printf("[instance][SYNC] Status unchanged for %s: %s", inst.ID, inst.Status)
	}

	// 更新 SSH 信息（provider 返回空值时不覆盖已有记录）
	sshChanged := false
	if info.SSHHost != "" && info.SSHHost != inst.SSHHost {
		inst.SSHHost = info.SSHHost
		sshChanged = true
	}
	if info.SSHPort != 0 && info.SSHPort != inst.SSHPort {
		inst.SSHPort = info.SSHPort
		sshChanged = true
	}
	if info.SSHUser != "" && info.SSHUser != inst.SSHUser {
		inst.SSHUser = info.SSHUser
		sshChanged = true
	}
	if info.Password != "" && info.Password != inst.Password {
		inst.Password = info.Password
		sshChanged = true
	}
	if info.SSHCommand != "" && info.SSHCommand != inst.SSHCommand {
		inst.SSHCommand = info.SSHCommand
		sshChanged = true
	}
	if sshChanged {
		if dbErr := s.repo.UpdateSSH(ctx, inst.ID, inst.SSHHost, inst.SSHPort, inst.SSHUser, inst.Password, inst.SSHCommand); dbErr != nil {
			log.Printf("[instance] Failed to sync SSH for %s: %v", inst.ID, dbErr)
		}
	}
}

// StopInstance 停止实例
func (s *InstanceService) StopInstance(ctx context.Context, id string) error {
	inst, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	p, ok := s.providerManager.GetProvider(inst.Provider)
	if !ok {
		return fmt.Errorf("provider not found: %s", inst.Provider)
	}

	if err := p.StopInstance(ctx, inst.ProviderInstID); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	now := time.Now()
	inst.Status = model.InstanceStatusStopped
	inst.StoppedAt = &now
	return s.repo.Update(ctx, inst)
}

// StartInstance 启动实例
func (s *InstanceService) StartInstance(ctx context.Context, id string) error {
	inst, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	p, ok := s.providerManager.GetProvider(inst.Provider)
	if !ok {
		return fmt.Errorf("provider not found: %s", inst.Provider)
	}

	if err := p.StartInstance(ctx, inst.ProviderInstID); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	inst.Status = model.InstanceStatusRunning
	return s.repo.Update(ctx, inst)
}

// DestroyInstance 销毁实例
func (s *InstanceService) DestroyInstance(ctx context.Context, id string) error {
	inst, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	p, ok := s.providerManager.GetProvider(inst.Provider)
	if !ok {
		return fmt.Errorf("provider not found: %s", inst.Provider)
	}

	// 1. 先更新为 destroying 状态，让前端立即感知
	if err := s.repo.UpdateStatus(ctx, id, model.InstanceStatusDestroying); err != nil {
		return fmt.Errorf("failed to set destroying status: %w", err)
	}
	log.Printf("[instance] Instance %s marked as destroying", id)

	// 2. 下发销毁指令（同步快速返回，失败不阻塞）
	if err := p.DestroyInstance(ctx, inst.ProviderInstID); err != nil {
		log.Printf("[instance] Initial destroy command failed for %s (remote: %s): %v", id, inst.ProviderInstID, err)
	}

	// 3. 异步轮询确认 provider 端已真正销毁
	go s.pollInstanceDestroyed(id, inst.Provider, inst.ProviderInstID)

	return nil
}

// pollInstanceDestroyed 轮询确认实例已销毁
func (s *InstanceService) pollInstanceDestroyed(instanceID, providerName, providerInstID string) {
	ctx := context.Background()
	p, ok := s.providerManager.GetProvider(providerName)
	if !ok {
		log.Printf("[instance] Provider %s not found for destroy polling", providerName)
		return
	}

	for i := 0; i < 24; i++ { // 最多轮询 6 分钟
		time.Sleep(15 * time.Second)

		status, err := p.GetInstanceStatus(ctx, providerInstID)
		if err != nil {
			// 查询出错，可能实例已不存在（已销毁），也可能是网络问题
			// 尝试再次调用 DestroyInstance，若返回 not found / 404 则确认已销毁
			if destroyErr := p.DestroyInstance(ctx, providerInstID); destroyErr != nil {
				errMsg := destroyErr.Error()
				if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404") || strings.Contains(errMsg, "No such container") {
					log.Printf("[instance] Instance %s confirmed destroyed (provider reports not found)", instanceID)
					s.markDestroyed(instanceID)
					return
				}
				log.Printf("[instance] Retry destroy %s failed: %v", instanceID, destroyErr)
			}
			continue
		}

		if status == "destroyed" || status == "exited" || status == "dead" {
			log.Printf("[instance] Instance %s confirmed destroyed (status: %s)", instanceID, status)
			s.markDestroyed(instanceID)
			return
		}

		// 若仍在运行或停止状态，再次下发销毁指令
		if status == "running" || status == "stopped" {
			log.Printf("[instance] Instance %s still %s, retrying destroy...", instanceID, status)
			if err := p.DestroyInstance(ctx, providerInstID); err != nil {
				log.Printf("[instance] Retry destroy %s: %v", instanceID, err)
			}
		}
	}

	// 轮询超时，强制标记为 destroyed
	log.Printf("[instance] Destroy poll timeout for %s, force marking as destroyed", instanceID)
	s.markDestroyed(instanceID)
}

func (s *InstanceService) markDestroyed(instanceID string) {
	ctx := context.Background()
	now := time.Now()
	inst := &model.Instance{
		ID:          instanceID,
		Status:      model.InstanceStatusDestroyed,
		DestroyedAt: &now,
	}
	if err := s.repo.Update(ctx, inst); err != nil {
		log.Printf("[instance] Failed to mark %s as destroyed: %v", instanceID, err)
	}
}



// startAutoReleaseLoop 自动释放过期实例
func (s *InstanceService) startAutoReleaseLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		expired, err := s.repo.ListExpired(ctx)
		if err != nil {
			log.Printf("[instance] Failed to list expired instances: %v", err)
			continue
		}

		for _, inst := range expired {
			log.Printf("[instance] Auto-releasing expired instance %s (expired at %s)", inst.ID, inst.ExpiresAt.Format(time.RFC3339))
			if err := s.DestroyInstance(ctx, inst.ID); err != nil {
				log.Printf("[instance] Failed to auto-release instance %s: %v", inst.ID, err)
			}
		}
	}
}

package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ml-platform/internal/executor/provider"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"

	"github.com/google/uuid"
)

// InstanceService 实例租赁服务
type InstanceService struct {
	repo           *repository.InstanceRepository
	providerManager *provider.ProviderManager
}

// NewInstanceService 创建实例服务
func NewInstanceService(repo *repository.InstanceRepository, pm *provider.ProviderManager) *InstanceService {
	svc := &InstanceService{
		repo:            repo,
		providerManager: pm,
	}
	// 启动自动释放定时器
	go svc.startAutoReleaseLoop()
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

		// 更新数据库
		updates := map[string]interface{}{
			"ssh_host": info.SSHHost,
			"ssh_port": info.SSHPort,
			"ssh_user": info.SSHUser,
			"status":   model.InstanceStatus(info.Status),
		}

		if info.Status == "running" {
			now := time.Now()
			updates["started_at"] = &now
			log.Printf("[instance] Instance %s is running, SSH: %s@%s:%d", instanceID, info.SSHUser, info.SSHHost, info.SSHPort)
		}

		result := s.repo.UpdateStatus(ctx, instanceID, model.InstanceStatus(info.Status))
		if result != nil {
			log.Printf("[instance] Failed to update status for %s: %v", instanceID, result)
		}
		if err := s.repo.UpdateSSH(ctx, instanceID, info.SSHHost, info.SSHPort, info.SSHUser); err != nil {
			log.Printf("[instance] Failed to update SSH for %s: %v", instanceID, err)
		}
		if info.Status == "running" {
			if err := s.repo.UpdateStartedAt(ctx, instanceID, updates["started_at"].(*time.Time)); err != nil {
				log.Printf("[instance] Failed to update started_at for %s: %v", instanceID, err)
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

// syncInstanceStatus 同步单个实例的 provider 端真实状态
func (s *InstanceService) syncInstanceStatus(ctx context.Context, inst *model.Instance) {
	log.Printf("[instance][SYNC] Start sync for %s (provider=%s, provider_inst_id=%s, status=%s)",
		inst.ID, inst.Provider, inst.ProviderInstID, inst.Status)

	// 终态实例无需同步
	if inst.Status == model.InstanceStatusDestroyed || inst.Status == model.InstanceStatusDestroying {
		log.Printf("[instance][SYNC] Skip %s: status is terminal (%s)", inst.ID, inst.Status)
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
		// provider 查不到实例，可能已被回收：标记为 destroyed
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404") || strings.Contains(errMsg, "No such container") {
			log.Printf("[instance][SYNC] Instance %s not found on provider %s, marking destroyed", inst.ID, inst.Provider)
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

	// 更新状态
	newStatus := model.InstanceStatus(info.Status)
	if newStatus != inst.Status {
		log.Printf("[instance] Sync status for %s: %s -> %s", inst.ID, inst.Status, newStatus)
		inst.Status = newStatus
		if dbErr := s.repo.UpdateStatus(ctx, inst.ID, newStatus); dbErr != nil {
			log.Printf("[instance] Failed to sync status for %s: %v", inst.ID, dbErr)
		}
	}

	// 更新 SSH 信息
	if info.SSHHost != "" && (info.SSHHost != inst.SSHHost || info.SSHPort != inst.SSHPort || info.SSHUser != inst.SSHUser) {
		inst.SSHHost = info.SSHHost
		inst.SSHPort = info.SSHPort
		inst.SSHUser = info.SSHUser
		if dbErr := s.repo.UpdateSSH(ctx, inst.ID, info.SSHHost, info.SSHPort, info.SSHUser); dbErr != nil {
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

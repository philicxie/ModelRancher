package vastai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH客户端
type SSHClient struct {
	client *ssh.Client
	config *ssh.ClientConfig
}

// NewSSHClient 创建SSH客户端
func NewSSHClient(host string, port int, user string, privateKey []byte) (*SSHClient, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// 连接SSH服务器
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	client, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	return &SSHClient{
		client: ssh.NewClient(client, chans, reqs),
		config: config,
	}, nil
}

// NewSSHClientWithPassword 使用密码创建SSH客户端
func NewSSHClientWithPassword(host string, port int, user, password string) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	client, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	return &SSHClient{
		client: ssh.NewClient(client, chans, reqs),
		config: config,
	}, nil
}

// Close 关闭SSH连接
func (s *SSHClient) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// Execute 执行命令
func (s *SSHClient) Execute(ctx context.Context, command string) (string, string, int, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// 设置超时
	if ctx != nil {
		session.Setenv("DEADLINE", "true")
	}

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			return "", "", -1, fmt.Errorf("command execution failed: %w", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

// ExecuteWithOutput 实时输出执行结果
func (s *SSHClient) ExecuteWithOutput(ctx context.Context, command string, outputChan chan<- string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = &writerChan{channel: outputChan}
	session.Stderr = &writerChan{channel: outputChan}

	return session.Run(command)
}

// UploadFile 上传文件
func (s *SSHClient) UploadFile(ctx context.Context, localPath, remotePath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	stat, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	go func() {
		defer localFile.Close()
		writer, _ := session.StdinPipe()
		defer writer.Close()

		fmt.Fprintf(writer, "C%04d %d %s\n", 0644, stat.Size(), remotePath)
		io.Copy(writer, localFile)
		fmt.Fprint(writer, "\x00")
	}()

	return session.Run("scp -t " + remotePath)
}

// DownloadFile 下载文件
func (s *SSHClient) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	session.Stdout = localFile

	return session.Run("scp -f " + remotePath)
}

// UploadDirectory 上传目录
func (s *SSHClient) UploadDirectory(ctx context.Context, localPath, remotePath string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	localDir, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local directory: %w", err)
	}
	defer localDir.Close()

	go func() {
		defer localDir.Close()
		writer, _ := session.StdinPipe()
		defer writer.Close()

		fmt.Fprintf(writer, "D%04d 0 %s\n", 0755, remotePath)
		uploadDirRecursive(writer, localDir, localPath, remotePath)
		fmt.Fprint(writer, "E\n")
	}()

	return session.Run("scp -tr " + remotePath)
}

// writerChan 用于实时输出
type writerChan struct {
	channel chan<- string
}

func (w *writerChan) Write(p []byte) (n int, err error) {
	w.channel <- string(p)
	return len(p), nil
}

// uploadDirRecursive 递归上传目录
func uploadDirRecursive(writer io.Writer, dir *os.File, basePath, remotePath string) {
	entries, err := dir.ReadDir(0)
	if err != nil {
		return
	}

	for _, entry := range entries {
		localPath := basePath + "/" + entry.Name()
		remoteFilePath := remotePath + "/" + entry.Name()

		if entry.IsDir() {
			fmt.Fprintf(writer, "D%04d 0 %s\n", 0755, remoteFilePath)
			subDir, err := os.Open(localPath)
			if err == nil {
				uploadDirRecursive(writer, subDir, localPath, remoteFilePath)
				subDir.Close()
			}
			fmt.Fprint(writer, "E\n")
		} else {
			stat, _ := entry.Info()
			if stat != nil {
				file, err := os.Open(localPath)
				if err == nil {
					fmt.Fprintf(writer, "C%04d %d %s\n", 0644, stat.Size(), remoteFilePath)
					io.Copy(writer, file)
					file.Close()
					fmt.Fprint(writer, "\x00")
				}
			}
		}
	}
}

// WaitForReady 等待SSH服务就绪
func (s *SSHClient) WaitForReady(ctx context.Context, timeout time.Duration) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(timeout)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, _, exitCode, err := s.Execute(ctx, "echo 'ready'")
			if err == nil && exitCode == 0 {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for SSH ready")
			}
		}
	}
}

// GetHostKey 获取主机密钥
func GetHostKey(host string, port int) (string, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	config := &ssh.ClientConfig{
		User:            "test",
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	_, _, _, err = ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return "", err
	}

	return "", nil
}

package executor

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient 远程实例 SSH 客户端
type SSHClient struct {
	Host     string
	Port     int
	User     string
	Password string
	client   *ssh.Client
}

// NewSSHClient 创建 SSH 客户端（尚未连接）
func NewSSHClient(host string, port int, user, password string) *SSHClient {
	return &SSHClient{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
}

// Connect 建立 SSH 连接
func (c *SSHClient) Connect() error {
	// 如果 Host 已包含端口（如 host:port），不再拼接 Port
	addr := c.Host
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}

	config := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	c.client = client
	return nil
}

// Close 关闭 SSH 连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Exec 执行远程命令，返回 stdout/stderr
func (c *SSHClient) Exec(command string) (string, string, error) {
	if c.client == nil {
		return "", "", fmt.Errorf("ssh client not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("ssh new session failed: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	log.Printf("[SSH %s@%s:%d] Exec: %s", c.User, c.Host, c.Port, command)
	err = session.Run(command)
	return stdoutBuf.String(), stderrBuf.String(), err
}

// WaitForSSH 轮询等待 SSH 就绪（最多等 5 分钟）
func WaitForSSH(host string, port int, user, password string) (*SSHClient, error) {
	client := NewSSHClient(host, port, user, password)
	maxRetries := 60
	for i := 0; i < maxRetries; i++ {
		if err := client.Connect(); err == nil {
			return client, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("ssh connection timed out after %d retries", maxRetries)
}

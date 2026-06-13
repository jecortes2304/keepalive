package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type StatusInfo struct {
	PID       int           `json:"pid"`
	StartedAt time.Time     `json:"started_at"`
	Strategy  string        `json:"strategy"`
	Movements int64         `json:"movements"`
	Duration  time.Duration `json:"duration"`
	Remaining time.Duration `json:"remaining"`
	Profile   string        `json:"profile"`
	Running   bool          `json:"running"`
}

type Server struct {
	socketPath string
	listener   net.Listener
	mu         sync.RWMutex
	status     StatusInfo
	stopCh     chan struct{}
}

func NewServer(socketPath string) *Server {
	return &Server{
		socketPath: socketPath,
		stopCh:     make(chan struct{}),
	}
}

func (s *Server) Start() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return err
	}
	os.Remove(s.socketPath)

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listening on socket: %w", err)
	}
	s.listener = l

	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() {
	close(s.stopCh)
	if s.listener != nil {
		err := s.listener.Close()
		if err != nil {
			fmt.Printf("failed to close listener: %v\n", err)
		}
	}
	os.Remove(s.socketPath)
}

func (s *Server) UpdateStatus(info StatusInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = info
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("failed to close connection: %v\n", err)
		}
	}(conn)

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	cmd := string(buf[:n])
	switch cmd {
	case "status":
		s.mu.RLock()
		data, _ := json.Marshal(s.status)
		s.mu.RUnlock()
		_, err := conn.Write(data)
		if err != nil {
			fmt.Printf("failed to write: %v\n", err)
		}
	case "stop":
		_, err := conn.Write([]byte("ok"))
		if err != nil {
			fmt.Printf("failed to write: %v\n", err)
		}
	}
}

type Client struct {
	socketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) GetStatus() (*StatusInfo, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon: %w", err)
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("failed to close connection: %v\n", err)
		}
	}(conn)

	_, err = conn.Write([]byte("status"))
	if err != nil {
		return nil, err
	}
	err = conn.(*net.UnixConn).CloseWrite()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var info StatusInfo
	if err := json.Unmarshal(buf[:n], &info); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &info, nil
}

func (c *Client) SendStop() error {
	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("connecting to daemon: %w", err)
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("failed to close connection: %v\n", err)
		}
	}(conn)

	_, err = conn.Write([]byte("stop"))
	if err != nil {
		return err
	}
	return nil
}

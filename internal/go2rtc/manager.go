package go2rtc

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"surveillance-client/internal/camera"
	"surveillance-client/internal/config"
)

type CameraLister interface {
	ListEnabled() ([]camera.Camera, error)
}

type Manager struct {
	cfg        *config.Config
	cameras    CameraLister
	client     *Client
	cmd        *exec.Cmd
	configPath string
}

func NewManager(cfg *config.Config, cameras CameraLister, client *Client) *Manager {
	return &Manager{
		cfg:        cfg,
		cameras:    cameras,
		client:     client,
		configPath: filepath.Join(filepath.Dir(cfg.DBPath), "go2rtc.yaml"),
	}
}

func (m *Manager) Start() error {
	// Check if go2rtc binary exists
	if _, err := os.Stat(m.cfg.Go2RTCPath); os.IsNotExist(err) {
		return fmt.Errorf("go2rtc binary not found at %s (run 'make download-go2rtc')", m.cfg.Go2RTCPath)
	}

	// Generate initial config
	cameras, err := m.cameras.ListEnabled()
	if err != nil {
		return fmt.Errorf("list cameras: %w", err)
	}

	if err := GenerateConfig(cameras, m.configPath); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}

	m.cmd = exec.Command(m.cfg.Go2RTCPath, "-config", m.configPath)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start go2rtc: %w", err)
	}

	log.Printf("go2rtc started (pid %d)", m.cmd.Process.Pid)

	// Wait for go2rtc to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if m.client.Healthy() {
			log.Printf("go2rtc is ready")
			return nil
		}
	}

	return fmt.Errorf("go2rtc did not become healthy within 3s")
}

func (m *Manager) Stop() {
	if m.cmd != nil && m.cmd.Process != nil {
		log.Printf("stopping go2rtc (pid %d)", m.cmd.Process.Pid)
		m.cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- m.cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			m.cmd.Process.Kill()
		}
	}
}

// SyncStreams registers all enabled cameras as go2rtc streams via the API
func (m *Manager) SyncStreams() error {
	cameras, err := m.cameras.ListEnabled()
	if err != nil {
		return fmt.Errorf("list cameras: %w", err)
	}

	if !m.client.Healthy() {
		// go2rtc not running, just regenerate config for next start
		return GenerateConfig(cameras, m.configPath)
	}

	for _, cam := range cameras {
		rtspURL := cam.RTSPMain
		if cam.Username != nil && cam.Password != nil && *cam.Username != "" {
			rtspURL = injectCredentials(rtspURL, *cam.Username, *cam.Password)
		}

		execSrc := fmt.Sprintf("exec:ffmpeg -rtsp_transport tcp -i %s -c:v copy -an -f mpegts -", rtspURL)
		if err := m.client.AddStream("cam_"+cam.ID, execSrc); err != nil {
			log.Printf("warning: failed to add main stream for %s: %v", cam.Name, err)
		}

		if cam.RTSPSub != nil && *cam.RTSPSub != "" {
			subURL := *cam.RTSPSub
			if cam.Username != nil && cam.Password != nil && *cam.Username != "" {
				subURL = injectCredentials(subURL, *cam.Username, *cam.Password)
			}
			execSubSrc := fmt.Sprintf("exec:ffmpeg -rtsp_transport tcp -i %s -c:v copy -an -f mpegts -", subURL)
			if err := m.client.AddStream("cam_"+cam.ID+"_sub", execSubSrc); err != nil {
				log.Printf("warning: failed to add sub stream for %s: %v", cam.Name, err)
			}
		}
	}

	// Also regenerate config file for persistence
	return GenerateConfig(cameras, m.configPath)
}

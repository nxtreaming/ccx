package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BenedictKing/ccx/desktop/internal/backend"
	"github.com/BenedictKing/ccx/desktop/internal/configservice"
	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type DesktopService struct {
	manager       *backend.Manager
	configService *configservice.Service
	app           *application.App
	mainWindow    application.Window
}

type EnvFileState struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Exists  bool   `json:"exists"`
}

func NewDesktopService(manager *backend.Manager) *DesktopService {
	configService, _ := configservice.New(manager.DataDir())
	return &DesktopService{manager: manager, configService: configService}
}

func (s *DesktopService) setApp(app *application.App) {
	s.app = app
}

func (s *DesktopService) setMainWindow(window application.Window) {
	s.mainWindow = window
}

func (s *DesktopService) GetStatus() backend.Status {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	return s.manager.Status(ctx)
}

func (s *DesktopService) GetProxyAccessKey() (string, error) {
	return s.manager.EnsureProxyAccessKey()
}

func (s *DesktopService) GetEnvFile() (EnvFileState, error) {
	path := filepath.Join(s.manager.DataDir(), ".env")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EnvFileState{Path: path, Exists: false}, nil
		}
		return EnvFileState{}, err
	}
	return EnvFileState{Path: path, Content: string(content), Exists: true}, nil
}

func (s *DesktopService) SaveEnvFile(content string) error {
	path := filepath.Join(s.manager.DataDir(), ".env")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func (s *DesktopService) StartService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return s.manager.Start(ctx)
}

func (s *DesktopService) StopService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return s.manager.Stop(ctx)
}

func (s *DesktopService) RestartService() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.manager.Restart(ctx)
}

func (s *DesktopService) GetLogs() []string {
	return s.manager.Logs()
}

func (s *DesktopService) GetAgentConfigStatus(platform string) (configservice.AgentConfigStatus, error) {
	if s.configService == nil {
		return configservice.AgentConfigStatus{}, fmt.Errorf("配置服务未初始化")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	status := s.manager.Status(ctx)
	return s.configService.GetStatus(platform, status.Port)
}

func (s *DesktopService) ApplyAgentConfig(req configservice.ApplyAgentConfigRequest) error {
	if s.configService == nil {
		return fmt.Errorf("配置服务未初始化")
	}
	platform := req.Platform
	if platform == "" {
		return fmt.Errorf("agent 平台不能为空")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	status := s.manager.Status(ctx)
	var key string
	if platform == configservice.PlatformCodex || (platform == configservice.PlatformClaude && (req.Provider == "" || req.Provider == configservice.ProviderCCX)) {
		if !status.Running {
			return fmt.Errorf("请先启动 CCX 服务")
		}
		var err error
		key, err = s.manager.EnsureProxyAccessKey()
		if err != nil {
			return err
		}
	}
	return s.configService.Apply(req, status.Port, key)
}

func (s *DesktopService) RestoreAgentConfig(platform string) error {
	if s.configService == nil {
		return fmt.Errorf("配置服务未初始化")
	}
	return s.configService.Restore(platform)
}

func (s *DesktopService) ShowStatusTab() error {
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "status")
	}
	return nil
}

func (s *DesktopService) ShowAgentTab() error {
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "agent")
	}
	return nil
}

func (s *DesktopService) ShowWebUITab() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.manager.Start(ctx); err != nil {
		return err
	}
	if err := s.manager.WaitHealthy(ctx, 15*time.Second); err != nil {
		return err
	}
	s.showWindow()
	if s.app != nil {
		s.app.Event.Emit("desktop:show-tab", "web")
	}
	return nil
}

func (s *DesktopService) OpenWebUIInBrowser() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.manager.Start(ctx); err != nil {
		return err
	}
	if err := s.manager.WaitHealthy(ctx, 15*time.Second); err != nil {
		return err
	}
	return browser.OpenURL(s.manager.WebURL())
}

func (s *DesktopService) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = s.manager.Stop(ctx)
}

func (s *DesktopService) showWindow() {
	if s.mainWindow == nil {
		return
	}
	if s.mainWindow.IsMinimised() {
		s.mainWindow.UnMinimise()
	}
	s.mainWindow.Show()
	s.mainWindow.Focus()
}

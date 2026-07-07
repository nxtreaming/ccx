package main

import (
	"embed"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/BenedictKing/ccx/desktop/internal/appdirs"
	"github.com/BenedictKing/ccx/desktop/internal/backend"
	"github.com/BenedictKing/ccx/desktop/internal/singleinstance"
	"github.com/BenedictKing/ccx/desktop/internal/windowstate"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

// 构建时通过 -ldflags 注入；保留默认值仅用于 dev 模式
var (
	Version      = "dev"
	BuildTime    = "unknown"
	GitCommit    = "unknown"
	Distribution = "github"
)

func init() {
	application.RegisterEvent[string]("desktop:show-tab")
	application.RegisterEvent[string]("desktop:tray-error")
	application.RegisterEvent[bool]("desktop:window-visibility")
}

func main() {
	defer recoverWithMessageBox()
	if err := run(); err != nil {
		showErrorDialog("CCX Desktop - 启动失败", err.Error())
		os.Exit(1)
	}
}

func run() error {
	bootstrapCloser := setupBootstrapLog()
	defer bootstrapCloser()
	log.Printf("[Desktop-Boot] process starting cwd=%s exe=%s", mustGetwd(), mustExecutable())

	log.Printf("[Desktop-Boot] creating backend manager")
	manager := backend.NewManager(backend.Options{})
	log.Printf("[Desktop-Boot] backend manager created dataDir=%s", manager.DataDir())

	// 文件日志：同时写入 dataDir/desktop.log 和 stderr
	logCloser := setupFileLog(manager.DataDir())
	defer logCloser()
	log.Printf("[Desktop-Boot] cwd=%s exe=%s dataDir=%s", mustGetwd(), mustExecutable(), manager.DataDir())

	// 单实例互斥锁：检测已有实例时弹窗退出
	log.Printf("[Desktop-Boot] acquiring single instance lock")
	lock, err := singleinstance.Acquire(singleInstanceArg(manager.DataDir()))
	if err != nil {
		if err == singleinstance.ErrAlreadyRunning {
			showErrorDialog("CCX Desktop", "CCX Desktop 已经在运行中。\n\n请检查系统托盘或任务栏。")
			os.Exit(0)
		}
		return fmt.Errorf("获取单实例锁失败: %w", err)
	}
	defer lock.Release()
	log.Printf("[Desktop-Boot] single instance lock acquired")
	desktopService := NewDesktopService(manager)
	desktopService.setVersion(VersionInfo{
		Version:      Version,
		BuildTime:    BuildTime,
		GitCommit:    GitCommit,
		Distribution: Distribution,
	})
	log.Printf("[Desktop-Boot] desktop service initialized")
	dockService := dock.New()

	log.Printf("[Desktop-Boot] creating Wails application")
	app := application.New(application.Options{
		Name:        "CCX Desktop",
		Description: "CCX desktop shell and core service supervisor",
		Services: []application.Service{
			application.NewService(desktopService),
			application.NewService(dockService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	log.Printf("[Desktop-Boot] Wails application created")
	desktopService.setApp(app)

	// 应用持久化窗口状态（如存在），否则回退到默认 Center。
	// X/Y 仅在 InitialPosition=WindowXY 时生效（go doc 确认）。
	windowOpts := application.WebviewWindowOptions{
		Title:     "CCX Desktop",
		Width:     1180,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropNormal,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 24, 38),
		URL:              "/",
	}
	var savedMaximised bool
	persistedState, hasPersistedState, _ := windowstate.Load(manager.DataDir())
	if hasPersistedState {
		windowOpts.Width = persistedState.Width
		windowOpts.Height = persistedState.Height
		windowOpts.X = persistedState.X
		windowOpts.Y = persistedState.Y
		windowOpts.InitialPosition = application.WindowXY
		savedMaximised = persistedState.Maximised
	}

	mainWindow := app.Window.NewWithOptions(windowOpts)
	if savedMaximised {
		mainWindow.Maximise()
	}
	desktopService.setMainWindow(mainWindow)
	log.Printf("[Desktop-Boot] main window scheduled")

	saveWindowState := func() {
		x, y := mainWindow.Position()
		w, h := mainWindow.Size()
		if w == 0 && h == 0 {
			return // 窗口未初始化时跳过，避免覆盖有效数据
		}
		state := windowstate.State{
			X:         x,
			Y:         y,
			Width:     w,
			Height:    h,
			Maximised: mainWindow.IsMaximised(),
		}
		if !windowstate.IsValid(state) {
			return
		}
		if err := windowstate.Save(manager.DataDir(), state); err != nil {
			log.Printf("[Desktop-Window] 保存窗口状态失败: %v", err)
		}
	}

	dockIconHidden := false
	setDockIconVisible := func(visible bool) {
		if runtime.GOOS != "darwin" {
			return
		}
		if visible {
			if !dockIconHidden {
				return
			}
			dockIconHidden = false
			go dockService.ShowAppIcon()
			return
		}
		if dockIconHidden {
			return
		}
		dockIconHidden = true
		go dockService.HideAppIcon()
	}

	hideMainWindow := func() {
		mainWindow.Hide()
		setDockIconVisible(false)
		app.Event.Emit("desktop:window-visibility", false)
	}

	var mainWindowCentered = hasPersistedState
	showMainWindow := func(withFocus bool) {
		setDockIconVisible(true)
		if !mainWindowCentered {
			mainWindow.Center()
			mainWindowCentered = true
		}
		if mainWindow.IsMinimised() {
			mainWindow.UnMinimise()
		}
		mainWindow.Show()
		app.Event.Emit("desktop:window-visibility", true)
		if withFocus {
			if runtime.GOOS == "windows" {
				mainWindow.SetAlwaysOnTop(true)
				mainWindow.Focus()
				go func() {
					time.Sleep(150 * time.Millisecond)
					mainWindow.SetAlwaysOnTop(false)
				}()
			} else {
				mainWindow.Focus()
			}
		}
	}

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		saveWindowState()
		hideMainWindow()
		e.Cancel()
	})
	mainWindow.RegisterHook(events.Common.WindowMinimise, func(e *application.WindowEvent) {
		setDockIconVisible(false)
		app.Event.Emit("desktop:window-visibility", false)
	})
	mainWindow.RegisterHook(events.Common.WindowUnMinimise, func(e *application.WindowEvent) {
		setDockIconVisible(true)
		app.Event.Emit("desktop:window-visibility", true)
	})

	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(event *application.ApplicationEvent) {
		showMainWindow(true)
	})

	app.OnShutdown(func() {
		saveWindowState()
		desktopService.Shutdown()
	})

	tray := app.SystemTray.New()
	log.Printf("[Desktop-Boot] system tray created")
	tray.SetTooltip("CCX Desktop")
	if icon, err := assets.ReadFile("frontend/dist/tray-template.png"); err == nil && len(icon) > 0 {
		tray.SetTemplateIcon(icon)
	} else if icon, err := assets.ReadFile("frontend/dist/wails.png"); err == nil && len(icon) > 0 {
		tray.SetTemplateIcon(icon)
	}

	trayAction := func(label string, fn func() error) {
		go func() {
			if err := fn(); err != nil {
				log.Printf("[Desktop-Tray] %s 失败: %v", label, err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("%s 失败: %v", label, err))
			}
		}()
	}

	updaterInitialized := false
	buildTrayMenu := func(running bool, port int, pid int, autostartEnabled bool) *application.Menu {
		menu := application.NewMenu()

		// 顶部状态摘要（不可点击）
		var statusLabel string
		switch {
		case running && port > 0 && pid > 0:
			statusLabel = fmt.Sprintf("● 运行中 · :%d · PID %d", port, pid)
		case running && port > 0:
			statusLabel = fmt.Sprintf("● 运行中 · :%d", port)
		case running:
			statusLabel = "● 运行中"
		default:
			statusLabel = "○ 已停止"
		}
		header := menu.Add(statusLabel)
		header.SetEnabled(false)
		menu.AddSeparator()

		menu.Add("打开 CCX Web UI").OnClick(func(ctx *application.Context) {
			trayAction("打开 CCX Web UI", desktopService.ShowWebUITab)
		})
		menu.Add("显示状态页").OnClick(func(ctx *application.Context) {
			showMainWindow(true)
			app.Event.Emit("desktop:show-tab", "status")
		})
		menu.Add("显示 Agent 配置").OnClick(func(ctx *application.Context) {
			showMainWindow(true)
			app.Event.Emit("desktop:show-tab", "agent")
		})

		menu.AddSeparator()

		startItem := menu.Add("启动服务")
		startItem.OnClick(func(ctx *application.Context) {
			trayAction("启动服务", desktopService.StartService)
		})
		startItem.SetHidden(running)

		stopItem := menu.Add("停止服务")
		stopItem.OnClick(func(ctx *application.Context) {
			trayAction("停止服务", desktopService.StopService)
		})
		stopItem.SetHidden(!running)

		restartItem := menu.Add("重启服务")
		restartItem.OnClick(func(ctx *application.Context) {
			trayAction("重启服务", desktopService.RestartService)
		})
		restartItem.SetHidden(!running)

		menu.Add("在浏览器中打开").OnClick(func(ctx *application.Context) {
			trayAction("在浏览器中打开", desktopService.OpenWebUIInBrowser)
		})

		menu.AddSeparator()

		menu.Add("复制 Web UI 地址").OnClick(func(ctx *application.Context) {
			url := desktopService.WebURL()
			if err := desktopService.CopyText(url); err != nil {
				log.Printf("[Desktop-Tray] 复制 Web UI 地址失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("复制失败: %v", err))
				return
			}
		})

		menu.Add("复制 PROXY_ACCESS_KEY").OnClick(func(ctx *application.Context) {
			key, err := desktopService.GetProxyAccessKey()
			if err != nil {
				log.Printf("[Desktop-Tray] 获取 PROXY_ACCESS_KEY 失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("获取密钥失败: %v", err))
				return
			}
			if err := desktopService.CopyText(key); err != nil {
				log.Printf("[Desktop-Tray] 复制 PROXY_ACCESS_KEY 失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("复制失败: %v", err))
				return
			}
		})

		menu.AddSeparator()

		autostartItem := menu.AddCheckbox("开机自启", autostartEnabled)
		autostartItem.OnClick(func(ctx *application.Context) {
			newState := ctx.IsChecked()
			if err := desktopService.SetAutostart(newState); err != nil {
				log.Printf("[Desktop-Tray] 切换开机自启失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("切换开机自启失败: %v", err))
			}
		})

		if desktopService.isStoreDistribution() {
			updateItem := menu.Add("由 Microsoft Store 更新")
			updateItem.SetEnabled(false)
		} else if !desktopService.versionInfo.SupportsInAppUpdate {
			updateItem := menu.Add("请从发布页更新")
			updateItem.SetEnabled(false)
		} else if !updaterInitialized {
			updateItem := menu.Add("更新不可用")
			updateItem.SetEnabled(false)
		} else {
			menu.Add("检查更新…").OnClick(func(ctx *application.Context) {
				if err := desktopService.StartInAppUpdate(); err != nil {
					log.Printf("[Desktop-Updater] 检查更新失败: %v", err)
					app.Event.Emit("desktop:tray-error", fmt.Sprintf("检查更新失败: %v", err))
				}
			})
		}

		menu.AddSeparator()

		versionItem := menu.Add(fmt.Sprintf("CCX Desktop v%s", Version))
		versionItem.SetEnabled(false)

		menu.Add("退出").OnClick(func(ctx *application.Context) {
			app.Quit()
		})

		return menu
	}

	// 计算托盘 tooltip 文本
	tooltipFor := func(st backend.Status) string {
		switch {
		case st.Running && st.Port > 0:
			return fmt.Sprintf("CCX Desktop · 运行中 · :%d", st.Port)
		case st.Starting:
			return "CCX Desktop · 启动中"
		default:
			return "CCX Desktop · 已停止"
		}
	}

	// 初始化托盘菜单
	initialStatus := desktopService.GetStatus()
	log.Printf("[Desktop-Boot] initial status read: running=%v port=%d pid=%d", initialStatus.Running, initialStatus.Port, initialStatus.PID)
	log.Printf("[Desktop-Boot] reading autostart status")
	initialAutostart, _ := app.Autostart.IsEnabled()
	log.Printf("[Desktop-Boot] autostart status read: enabled=%v", initialAutostart)
	tray.SetMenu(buildTrayMenu(initialStatus.Running, initialStatus.Port, initialStatus.PID, initialAutostart))
	tray.SetTooltip(tooltipFor(initialStatus))
	log.Printf("[Desktop-Boot] tray menu initialized")

	// 状态变化时动态刷新菜单与 tooltip
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		lastRunning := initialStatus.Running
		lastStarting := initialStatus.Starting
		lastPort := initialStatus.Port
		lastPid := initialStatus.PID
		lastAutostart := initialAutostart
		for range ticker.C {
			st := desktopService.GetStatus()
			asEnabled, _ := app.Autostart.IsEnabled()
			menuChanged := st.Running != lastRunning || st.Port != lastPort || st.PID != lastPid || asEnabled != lastAutostart
			tooltipChanged := st.Running != lastRunning || st.Starting != lastStarting || st.Port != lastPort
			if menuChanged {
				tray.SetMenu(buildTrayMenu(st.Running, st.Port, st.PID, asEnabled))
			}
			if tooltipChanged {
				tray.SetTooltip(tooltipFor(st))
			}
			lastRunning = st.Running
			lastStarting = st.Starting
			lastPort = st.Port
			lastPid = st.PID
			lastAutostart = asEnabled
		}
	}()

	// 自定义托盘左键行为：手动 toggle 窗口可见性。
	// 不使用 AttachWindow 的默认 ToggleWindow，因为它会通过 PositionWindow
	// 将窗口定位到托盘图标旁（macOS 右上角），覆盖用户保存的窗口位置。
	tray.OnClick(func() {
		if mainWindow.IsVisible() {
			saveWindowState()
			hideMainWindow()
		} else {
			showMainWindow(true)
		}
	})

	// 初始化 Wails v3 内置 Updater（当前仅非 Store macOS 分发启用）。
	// 不开启 CheckInterval：自动轮询会在每个间隔自动 CheckAndInstall 弹出更新窗口，
	// 即便「up-to-date」或失败也会保留窗口，干扰严重。改为静默调用 GitHub Releases
	// API（见 desktopservice.CheckLatestRelease）+ 侧栏胶囊提示，用户主动点击托盘
	// 菜单或胶囊后再走 wails updater 的下载安装流程。
	if desktopService.versionInfo.SupportsInAppUpdate {
		gh, err := newGitHubUpdaterProvider()
		if err != nil {
			log.Printf("[Desktop-Updater] 创建 GitHub provider 失败: %v", err)
		} else {
			if err := app.Updater.Init(updater.Config{
				CurrentVersion: Version,
				Providers:      []updater.Provider{gh},
			}); err != nil {
				log.Printf("[Desktop-Updater] Updater 初始化失败: %v", err)
			} else {
				updaterInitialized = true
				// Updater 初始化成功后刷新托盘菜单，使"检查更新"入口可见
				st := desktopService.GetStatus()
				asEnabled, _ := app.Autostart.IsEnabled()
				tray.SetMenu(buildTrayMenu(st.Running, st.Port, st.PID, asEnabled))
			}
		}
	}

	showMainWindow(false)
	log.Printf("[Desktop-Boot] main window show requested")

	log.Printf("[Desktop-Boot] entering app.Run")
	if err := app.Run(); err != nil {
		return err
	}
	log.Printf("[Desktop-Boot] app.Run returned")
	return nil
}

// setupFileLog 在 dataDir 下打开 desktop.log 并将 log 输出同时写入文件和 stderr。
// 返回的关闭函数应通过 defer 调用。
func setupFileLog(dataDir string) func() {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Printf("[Desktop-Log] 无法创建日志目录 %s: %v", dataDir, err)
		return func() {}
	}
	logPath := filepath.Join(dataDir, "desktop.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("[Desktop-Log] 无法打开日志文件 %s: %v", logPath, err)
		return func() {}
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.Printf("[Desktop-Log] 日志文件: %s", logPath)
	return func() { f.Close() }
}

// setupBootstrapLog 在 backend manager 初始化前写入固定位置日志，覆盖 dataDir 计算阶段的启动问题。
func setupBootstrapLog() func() {
	logDir := defaultBootstrapLogDir()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return func() {}
	}
	logPath := filepath.Join(logDir, "bootstrap.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return func() {}
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.Printf("[Desktop-Log] 启动日志文件: %s", logPath)
	return func() { f.Close() }
}

func defaultBootstrapLogDir() string {
	return appdirs.DataDir()
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "<error: " + err.Error() + ">"
	}
	return wd
}

func mustExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return "<error: " + err.Error() + ">"
	}
	return exe
}

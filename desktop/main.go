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

	"github.com/BenedictKing/ccx/desktop/internal/backend"
	"github.com/BenedictKing/ccx/desktop/internal/singleinstance"
	"github.com/BenedictKing/ccx/desktop/internal/windowstate"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed all:frontend/dist
var assets embed.FS

// 构建时通过 -ldflags 注入；保留默认值仅用于 dev 模式
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func init() {
	application.RegisterEvent[string]("desktop:show-tab")
	application.RegisterEvent[string]("desktop:tray-error")
	application.RegisterEvent[UpdateInfo]("update:available")
}

func main() {
	defer recoverWithMessageBox()
	if err := run(); err != nil {
		showErrorDialog("CCX Desktop - 启动失败", err.Error())
		os.Exit(1)
	}
}

func run() error {
	manager := backend.NewManager(backend.Options{})

	// 文件日志：同时写入 dataDir/desktop.log 和 stderr
	logCloser := setupFileLog(manager.DataDir())
	defer logCloser()

	// 单实例互斥锁：检测已有实例时弹窗退出
	lock, err := singleinstance.Acquire(singleInstanceArg(manager.DataDir()))
	if err != nil {
		if err == singleinstance.ErrAlreadyRunning {
			showErrorDialog("CCX Desktop", "CCX Desktop 已经在运行中。\n\n请检查系统托盘或任务栏。")
			os.Exit(0)
		}
		return fmt.Errorf("获取单实例锁失败: %w", err)
	}
	defer lock.Release()
	desktopService := NewDesktopService(manager)
	desktopService.setVersion(VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	})
	dockService := dock.New()
	notificationService := notifications.New()
	desktopService.setNotifications(notificationService)

	app := application.New(application.Options{
		Name:        "CCX Desktop",
		Description: "CCX desktop shell and core service supervisor",
		Services: []application.Service{
			application.NewService(desktopService),
			application.NewService(dockService),
			application.NewService(notificationService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})
	desktopService.setApp(app)

	// macOS 首启请求通知权限（系统自身记忆已授权状态，不会反复弹窗）
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if runtime.GOOS != "darwin" {
			return
		}
		go func() {
			granted, err := notificationService.CheckNotificationAuthorization()
			if err != nil {
				log.Printf("[Desktop-Notify] 检查通知授权失败: %v", err)
				return
			}
			if granted {
				return
			}
			if _, err := notificationService.RequestNotificationAuthorization(); err != nil {
				log.Printf("[Desktop-Notify] 请求通知授权失败: %v", err)
			}
		}()
	})

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
			Backdrop:                application.MacBackdropTranslucent,
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

	var mainWindowCentered = hasPersistedState
	showMainWindow := func(withFocus bool) {
		if !mainWindowCentered {
			mainWindow.Center()
			mainWindowCentered = true
		}
		if mainWindow.IsMinimised() {
			mainWindow.UnMinimise()
		}
		mainWindow.Show()
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
		mainWindow.Hide()
		e.Cancel()
	})

	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(event *application.ApplicationEvent) {
		showMainWindow(true)
	})

	app.OnShutdown(func() {
		saveWindowState()
		desktopService.Shutdown()
	})

	tray := app.SystemTray.New()
	tray.SetTooltip("CCX Desktop")
	if icon, err := assets.ReadFile("frontend/dist/wails.png"); err == nil && len(icon) > 0 {
		tray.SetTemplateIcon(icon)
	}

	trayAction := func(label string, fn func() error) {
		go func() {
			if err := fn(); err != nil {
				log.Printf("[Desktop-Tray] %s 失败: %v", label, err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("%s 失败: %v", label, err))
				desktopService.Notify(fmt.Sprintf("CCX %s 失败", label), err.Error())
			}
		}()
	}

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
				desktopService.Notify("CCX 复制失败", err.Error())
				return
			}
			desktopService.Notify("已复制 Web UI 地址", url)
		})

		menu.Add("复制 PROXY_ACCESS_KEY").OnClick(func(ctx *application.Context) {
			key, err := desktopService.GetProxyAccessKey()
			if err != nil {
				log.Printf("[Desktop-Tray] 获取 PROXY_ACCESS_KEY 失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("获取密钥失败: %v", err))
				desktopService.Notify("CCX 获取密钥失败", err.Error())
				return
			}
			if err := desktopService.CopyText(key); err != nil {
				log.Printf("[Desktop-Tray] 复制 PROXY_ACCESS_KEY 失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("复制失败: %v", err))
				desktopService.Notify("CCX 复制失败", err.Error())
				return
			}
			desktopService.Notify("已复制 PROXY_ACCESS_KEY", "可粘贴到 Agent 配置")
		})

		menu.AddSeparator()

		autostartItem := menu.AddCheckbox("开机自启", autostartEnabled)
		autostartItem.OnClick(func(ctx *application.Context) {
			newState := !autostartItem.Checked()
			if err := desktopService.SetAutostart(newState); err != nil {
				log.Printf("[Desktop-Tray] 切换开机自启失败: %v", err)
				app.Event.Emit("desktop:tray-error", fmt.Sprintf("切换开机自启失败: %v", err))
			}
		})

		menu.Add("检查更新…").OnClick(func(ctx *application.Context) {
			go func() {
				info, err := desktopService.CheckUpdate()
				if err != nil {
					log.Printf("[Desktop-Updater] 检查更新失败: %v", err)
					app.Event.Emit("desktop:tray-error", fmt.Sprintf("检查更新失败: %v", err))
					return
				}
				if !info.Available {
					app.Event.Emit("desktop:tray-error", "已经是最新版本")
				}
			}()
		})

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
	initialAutostart, _ := app.Autostart.IsEnabled()
	tray.SetMenu(buildTrayMenu(initialStatus.Running, initialStatus.Port, initialStatus.PID, initialAutostart))
	tray.SetTooltip(tooltipFor(initialStatus))

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
			mainWindow.Hide()
		} else {
			showMainWindow(true)
		}
	})

	// 启动 5s 后首次检查更新，之后每 30 分钟轮询一次
	go func() {
		time.Sleep(5 * time.Second)
		runCheck := func() {
			info, err := desktopService.CheckUpdate()
			if err != nil {
				log.Printf("[Desktop-Updater] 自动检查失败: %v", err)
				return
			}
			if info.Available {
				log.Printf("[Desktop-Updater] 发现新版本 %s", info.LatestVersion)
			}
		}
		runCheck()
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			runCheck()
		}
	}()

	showMainWindow(false)

	if err := app.Run(); err != nil {
		return err
	}
	return nil
}

// setupFileLog 在 dataDir 下打开 desktop.log 并将 log 输出同时写入文件和 stderr。
// 返回的关闭函数应通过 defer 调用。
func setupFileLog(dataDir string) func() {
	logPath := filepath.Join(dataDir, "desktop.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("[Desktop-Log] 无法打开日志文件 %s: %v", logPath, err)
		return func() {}
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.Printf("[Desktop-Log] 日志文件: %s", logPath)
	return func() { f.Close() }
}

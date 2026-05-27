package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/index.html frontend/main.js frontend/index.css frontend/bindings
var assets embed.FS

func main() {
	configManager := NewConfigManager()
	logWriter := NewWailsLogWriter()
	runtime := NewGTProcessRuntime(configManager, logWriter)

	gtApp := NewGTApp(configManager, logWriter, runtime)

	var app *application.App

	app = application.New(application.Options{
		Name:        "GT-Desktop",
		Description: "GT 高性能内网穿透客户端",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(gtApp),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.isrc-cas.gt-desktop",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if mainWindow, found := app.Window.GetByName("main"); found {
					mainWindow.Show()
					mainWindow.Focus()
				}
			},
		},
	})

	tray := app.SystemTray.New()
	tray.SetTooltip("GT 内网穿透客户端")

	menu := application.NewMenu()
	menu.Add("显示主面板").OnClick(func(ctx *application.Context) {
		mainWindow, found := app.Window.GetByName("main")
		if found {
			mainWindow.Show()
			mainWindow.Focus()
		}
	})
	menu.AddSeparator()
	menu.Add("一键开启穿透").OnClick(func(ctx *application.Context) {
		_ = gtApp.StartTunnel()
	})
	menu.Add("一键停止穿透").OnClick(func(ctx *application.Context) {
		_ = gtApp.StopTunnel()
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "GT Desktop Client",
		Width:     960,
		Height:    680,
		MinWidth:  800,
		MinHeight: 600,
		Windows: application.WindowsWindow{
			BackdropType: application.Mica, // Mica 亚克力毛玻璃质感背景，素雅高级
		},
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				Hide: true,
			},
		},
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

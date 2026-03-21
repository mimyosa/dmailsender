package main

import (
	"embed"

	"dMailSender/core"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewAppService()

	// Load saved window state; use min size on first run
	cfg, _ := core.LoadConfig()
	width := cfg.Window.Width
	height := cfg.Window.Height
	if width < 880 || height < 620 {
		width = 880
		height = 620
	}

	err := wails.Run(&options.App{
		Title:     "dMailSender",
		MinWidth:  880,
		MinHeight: 620,
		Width:     width,
		Height:    height,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/michalbartak/dbaccounts/internal/version"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:  "DbAccounts " + version.Get().Version,
		Width:  1100,
		Height: 780,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 26, B: 32, A: 255},
		OnStartup:        app.startup,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

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

// Window sizing: defaults and floor. The persisted size is the OS window size (via Wails
// WindowGetSize), so save/restore round-trips without shrinking each launch.
const (
	defaultWinW = 1100
	defaultWinH = 780
	minWinW     = 900
	minWinH     = 600
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Restore the last window size (floored at the minimum); fall back to the default.
	width, height := defaultWinW, defaultWinH
	cfg := app.store.Get()
	if cfg.WindowWidth >= minWinW {
		width = cfg.WindowWidth
	}
	if cfg.WindowHeight >= minWinH {
		height = cfg.WindowHeight
	}

	err = wails.Run(&options.App{
		Title:     "DbAccounts " + version.Get().Version,
		Width:     width,
		Height:    height,
		MinWidth:  minWinW,
		MinHeight: minWinH,
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

package main

import (
	"embed"
	"log"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/michal-bartak/pgcowboy/internal/version"
)

//go:embed all:frontend
var assets embed.FS

// versionFile is the repo-root VERSION, baked into the binary. It is the single source of the
// app version for every build (go run / wails dev / make), so a VERSION change is reflected
// without ldflags. Build metadata (commit, date) still comes from -ldflags.
//
//go:embed VERSION
var versionFile string

// appIcon is the window icon for Linux. macOS and Windows take theirs from the bundle / exe
// resource, but Wails sets no GTK window icon, so without this the window, task bar and
// alt-tab fall back to the desktop's generic application icon. Same PNG the packaged
// hicolor icon is resized from, so there is one source drawing.
//
//go:embed build/appicon.png
var appIcon []byte

// Window sizing: defaults and floor. The persisted size is the OS window size (via Wails
// WindowGetSize), so save/restore round-trips without shrinking each launch.
const (
	defaultWinW = 1100
	defaultWinH = 780
	minWinW     = 900
	minWinH     = 600
)

func main() {
	// The embedded VERSION file is the source of truth for the app version.
	if v := strings.TrimSpace(versionFile); v != "" {
		version.Version = v
	}

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
		Title:     "pgCowboy " + version.Get().Version,
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
		Linux: &linux.Options{
			// GTK window icon — see appIcon above.
			Icon: appIcon,
			// g_set_prgname(), which is also the Wayland app_id. It must equal the installed
			// desktop file's basename (pgcowboy.desktop) or the compositor cannot match the
			// window to the entry, and shows the generic icon and name. Without this it
			// defaults to the executable name ("pgCowboy"), which does not match.
			ProgramName: "pgcowboy",
			// Wails applies this default itself only when Options.Linux is nil; passing a
			// non-nil struct would otherwise silently flip it to "always" (the zero value)
			// and reintroduce the WebKitGTK rendering glitches of wailsapp/wails#2977.
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

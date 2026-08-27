package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// ── Service mode: run as a real Windows Service ──────────────
	for _, arg := range os.Args {
		if arg == "-svc" || arg == "--svc" {
			RunAsService()
			return
		}
	}

	// ── Silent audit mode ────────────────────────────────────────
	app := NewApp()
	for _, arg := range os.Args {
		if arg == "-silent" || arg == "-s" {
			err := app.PerformSilentAudit()
			if err != nil {
				fmt.Printf("Silent audit failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Silent audit successful.")
			os.Exit(0)
		}
	}

	// ── GUI mode ─────────────────────────────────────────────────
	err := wails.Run(&options.App{
		Title:  "TGS_All_In_One",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}


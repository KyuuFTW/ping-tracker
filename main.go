package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	interval := flag.Duration("interval", 3*time.Second, "scan interval")
	noPing := flag.Bool("no-ping", false, "disable ping measurements (faster, no TCP probes)")
	filter := flag.String("filter", "", "initial app name filter (substring match)")
	flag.Parse()

	app := NewApp(*interval, !*noPing, *filter)

	err := wails.Run(&options.App{
		Title:     "Ping Tracker",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

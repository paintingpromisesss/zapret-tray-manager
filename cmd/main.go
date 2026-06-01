package main

import (
	"log"
	"os"
	"strings"

	"zapret-tray-manager/internal/app"
	"zapret-tray-manager/internal/client"
	"zapret-tray-manager/internal/config"
	"zapret-tray-manager/internal/logging"
	"zapret-tray-manager/internal/manager"
	"zapret-tray-manager/internal/tray"
	"zapret-tray-manager/internal/zapretver"
)

func parseLangArg() string {
	for _, arg := range os.Args[1:] {
		if v, ok := strings.CutPrefix(arg, "--lang="); ok {
			return v
		}
	}
	return ""
}

func main() {
	logs, err := logging.Setup()
	if err != nil {
		log.Fatalf("setup logging: %v", err)
	}
	defer func() {
		if cerr := logs.Close(); cerr != nil {
			log.Printf("close logging: %v", cerr)
		}
	}()

	store, cfg, err := config.Load("")
	if err != nil {
		logs.Logger.Error("load config failed", "error", err)
		cfg = config.Default()
		store = config.NewConfigStore("")
	}

	if lang := parseLangArg(); lang != "" && cfg.Language == "" {
		cfg.Language = lang
		if err := store.Write(cfg); err != nil {
			logs.Logger.Warn("persist language failed", "error", err)
		}
	}

	httpClient := client.NewClient()
	mgr := manager.New(cfg.CurrentRoot, logs.Logger)
	vc := zapretver.NewClient(httpClient)
	application := app.New(store, cfg, mgr, vc, logs.Logger)
	application.SyncAutoRunFromService()
	tray.Run(application, logs.Logger, logs.Path)
}

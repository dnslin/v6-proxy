package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/zbronya/v6-proxy/config"
	"github.com/zbronya/v6-proxy/proxy"
	"github.com/zbronya/v6-proxy/sysutils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.ParseFlags()
	if cfg.CIDR == "" {
		slog.Error("cidr is required")
		os.Exit(1)
	}

	if cfg.AutoForwarding {
		sysutils.SetV6Forwarding()
	}

	if cfg.AutoRoute {
		sysutils.AddV6Route(cfg.CIDR)
	}

	if cfg.AutoIpNoLocalBind {
		sysutils.SetIpNonLocalBind()
	}

	p := proxy.NewProxyServer(cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Port)
	slog.Info("Starting server", "address", addr)
	err := http.ListenAndServe(addr, p)

	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}

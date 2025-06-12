package proxy

import (
	"github.com/elazarl/goproxy"
	"github.com/zbronya/v6-proxy/config"
)

func NewProxyServer(cfg config.Config) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	handler := NewProxyHandler(&cfg)

	// Handle HTTP requests
	proxy.OnRequest().DoFunc(handler.HandleRequest)

	// Handle HTTPS connect requests
	proxy.OnRequest().HijackConnect(handler.HandleConnect)

	return proxy
}

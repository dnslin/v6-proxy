package proxy

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/zbronya/v6-proxy/config"
	"github.com/zbronya/v6-proxy/netutils"

	"github.com/elazarl/goproxy"
)

// ProxyHandler holds the proxying logic.
type ProxyHandler struct {
	Cfg *config.Config
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{Cfg: cfg}
}

// HandleConnect handles CONNECT requests for HTTPS connections.
func (h *ProxyHandler) HandleConnect(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
	username := h.Cfg.AuthConfig.Username
	password := h.Cfg.AuthConfig.Password
	if username != "" && password != "" && !checkAuth(username, password, req) {
		client.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"Proxy\"\r\n\r\n"))
		client.Close()
		return
	}

	host := req.URL.Hostname()
	targetIp, isV6, err := netutils.GetIPAddress(host)
	if err != nil {
		log.Printf("Get IP address error: %v", err)
		return
	}

	if !isV6 {
		log.Printf("Connecting to %s [%s] from local net", req.URL.Host, targetIp)
		handleDirectConnection(req, client)
	} else {
		outgoingIP, err := netutils.RandomV6(h.Cfg.CIDR)
		if err != nil {
			log.Printf("Generate random IPv6 error: %v", err)
			return
		}

		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{IP: outgoingIP, Port: 0},
			Timeout:   30 * time.Second,
		}

		start := time.Now()
		server, err := dialer.Dial("tcp", req.URL.Host)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("Failed to connect to %s/%s from %s after %s: %v", req.URL.Host, req.URL.Scheme, outgoingIP.String(), elapsed, err)
			var opErr *net.OpError
			if errors.As(err, &opErr) {
				if opErr.Timeout() {
					log.Printf("Connection to %s timed out after %s", req.URL.Host, elapsed)
				} else {
					log.Printf("Failed to connect to %s due to: %v", req.URL.Host, opErr.Err)
				}
			}
			errorResponse := fmt.Sprintf("%s 500 Internal Server Error\r\n\r\n", req.Proto)
			client.Write([]byte(errorResponse))
			client.Close()
			return
		}

		log.Printf("Connecting to %s [%s] from %s", req.URL.Host, targetIp, outgoingIP.String())

		okResponse := fmt.Sprintf("%s 200 OK\r\n\r\n", req.Proto)
		client.Write([]byte(okResponse))

		proxyClientServer(client, server)
	}
}

// HandleRequest handles plain HTTP requests.
func (h *ProxyHandler) HandleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	// Authentication
	username := h.Cfg.AuthConfig.Username
	password := h.Cfg.AuthConfig.Password
	if username != "" && password != "" && !checkAuth(username, password, req) {
		return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusProxyAuthRequired, "Proxy Authentication Required")
	}

	// We need to remove the Proxy-Authorization header before forwarding the request
	req.Header.Del("Proxy-Authorization")

	host := req.URL.Hostname()
	targetIp, isV6, err := netutils.GetIPAddress(host)
	if err != nil {
		log.Printf("Get IP address error: %v", err)
		return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusInternalServerError, "Failed to resolve host")
	}

	var transport http.RoundTripper
	if isV6 {
		outgoingIP, err := netutils.RandomV6(h.Cfg.CIDR)
		if err != nil {
			log.Printf("Generate random IPv6 error: %v", err)
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusInternalServerError, "Failed to generate outgoing IP")
		}
		log.Printf("Connecting to %s [%s] from %s", req.URL.Host, targetIp, outgoingIP.String())
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				LocalAddr: &net.TCPAddr{IP: outgoingIP, Port: 0},
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
	} else {
		log.Printf("Connecting to %s [%s] from local net", req.URL.Host, targetIp)
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		log.Printf("HTTP request roundtrip error: %v", err)
		return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusInternalServerError, err.Error())
	}

	return req, resp
}

func proxyClientServer(client, server net.Conn) {
	go func() {
		defer server.Close()
		defer client.Close()
		io.Copy(server, client)
	}()
	go func() {
		defer server.Close()
		defer client.Close()
		io.Copy(client, server)
	}()
}

func handleDirectConnection(req *http.Request, client net.Conn) {
	server, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		errorResponse := fmt.Sprintf("%s 500 Internal Server Error\r\n\r\n", req.Proto)
		client.Write([]byte(errorResponse))
		client.Close()
		return
	}
	okResponse := fmt.Sprintf("%s 200 OK\r\n\r\n", req.Proto)
	client.Write([]byte(okResponse))
	proxyClientServer(client, server)
}

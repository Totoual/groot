package mcp

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultHTTPEndpoint = "/mcp"

type HTTPHandler struct {
	server   *Server
	endpoint string
	mu       sync.Mutex
}

func NewHTTPHandler(server *Server, endpoint string) *HTTPHandler {
	endpoint = normalizeHTTPEndpoint(endpoint)
	return &HTTPHandler{
		server:   server,
		endpoint: endpoint,
	}
}

func (s *Server) ListenHTTP(addr, endpoint string) error {
	handler := NewHTTPHandler(s, endpoint)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	return server.ListenAndServe()
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.endpoint {
		http.NotFound(w, r)
		return
	}
	if !mcpOriginAllowed(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handlePost(w, r)
	case http.MethodGet:
		w.Header().Set("Allow", "POST")
		http.Error(w, "server-sent event streams are not supported", http.StatusMethodNotAllowed)
	default:
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4*1024*1024))
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	response, err := h.server.HandleMessage(body)
	h.mu.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("handle mcp message: %v", err), http.StatusInternalServerError)
		return
	}
	if len(response) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("MCP-Protocol-Version", ProtocolVersion)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func normalizeHTTPEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return defaultHTTPEndpoint
	}
	if !strings.HasPrefix(endpoint, "/") {
		return "/" + endpoint
	}
	return endpoint
}

func mcpOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return true
	}

	return isLocalHTTPHost(u.Host) || sameHTTPHost(u.Host, r.Host)
}

func sameHTTPHost(left, right string) bool {
	leftHost, leftPort := splitHTTPHostPort(left)
	rightHost, rightPort := splitHTTPHostPort(right)
	return strings.EqualFold(leftHost, rightHost) && leftPort == rightPort
}

func isLocalHTTPHost(hostport string) bool {
	host, _ := splitHTTPHostPort(hostport)
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func splitHTTPHostPort(hostport string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err == nil {
		return strings.Trim(host, "[]"), port
	}
	return strings.Trim(hostport, "[]"), ""
}

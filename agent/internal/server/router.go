// Package server 本地 HTTP 服务: 路由 + 安全四件套中间件 + embed 前端托管。
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"

	"fenjue-agent/internal/platform"
	"fenjue-agent/internal/state"
)

// Version 对外汇报的代理版本。发版时须与 git tag 同步 bump (v0.2.2 曾漂移, docker 轮发现)。
const Version = "0.2.2"

// Server 本地伴随程序。
type Server struct {
	cfg     *platform.Config
	token   string
	port    int
	origins map[string]bool
	dist    fs.FS
}

// New 创建服务实例。extraOrigins 追加到默认 Origin 白名单。
func New(cfg *platform.Config, token string, port int, extraOrigins []string, dist fs.FS) *Server {
	origins := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:7799": true,
		"http://localhost:7799": true,
	}
	// 线上控制台来源 (platforms.json site.*): 握手链接印的就是这些域名, 默认放行。
	for _, site := range []string{cfg.Site.Primary, cfg.Site.Mirror} {
		if o, ok := originOf(site); ok {
			origins[o] = true
		}
	}
	for _, o := range extraOrigins {
		if o != "" {
			origins[o] = true
		}
	}
	return &Server{cfg: cfg, token: token, port: port, origins: origins, dist: dist}
}

// originOf 从站点 URL 提取 Origin (scheme://host[:port]); 非法输入返回 false。
func originOf(site string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(site))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	return u.Scheme + "://" + u.Host, true
}

// Handler 返回带安全中间件的根处理器。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/verify", s.handleVerify)
	mux.HandleFunc("/api/skills", s.handleSkills)
	mux.HandleFunc("/api/roots", s.handleRootsSet)
	mux.HandleFunc("/api/presets", s.handlePresets)
	mux.HandleFunc("/api/presets/", s.handlePresetOp)
	mux.HandleFunc("/api/platforms/", s.handlePlatformOp)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "unknown api path "+r.URL.Path)
	})
	mux.HandleFunc("/", s.handleStatic)
	return s.middleware(mux)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"ok": false, "error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"version":        Version,
		"os":             runtime.GOOS,
		"token_required": true,
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use GET")
		return
	}
	writeJSON(w, http.StatusOK, state.BuildSnapshot(s.cfg))
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use POST")
		return
	}
	writeJSON(w, http.StatusOK, state.Verify(s.cfg))
}

type enableBody struct {
	Roots map[string]string `json:"roots"`
}

type disableBody struct {
	Soft *bool `json:"soft"`
}

type restoreBody struct {
	BackupID string `json:"backupId"`
}

func (s *Server) handlePlatformOp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use POST")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/platforms/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeErr(w, http.StatusNotFound, "expected /api/platforms/{id}/{enable|disable|restore}")
		return
	}
		id, action := parts[0], parts[1]
		switch action {
		case "enable":
		var body enableBody
		if err := decodeBody(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cfg := s.cfg
		if len(body.Roots) > 0 {
			cfg = s.cfg.WithRoots(body.Roots)
		}
		res, err := state.Enable(cfg, id)
		if err != nil {
			respondOpErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	case "disable":
		var body disableBody
		if err := decodeBody(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		soft := true // 缺省按软关闭处理; 显式 soft=false 才硬关
		if body.Soft != nil {
			soft = *body.Soft
		}
		res, err := state.Disable(s.cfg, id, soft)
		if err != nil {
			respondOpErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	case "restore":
		var body restoreBody
		if err := decodeBody(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.TrimSpace(body.BackupID) == "" {
			writeErr(w, http.StatusBadRequest, "backupId is required in body")
			return
		}
		res, err := state.Restore(body.BackupID)
		if err != nil {
			respondOpErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	case "sync":
		res, err := state.SyncMirror(s.cfg, id)
		if err != nil {
			respondOpErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, res)
	default:
		writeErr(w, http.StatusNotFound, "unknown action "+action+"; use enable|disable|restore|sync")
	}
}

func respondOpErr(w http.ResponseWriter, err error) {
	var nf *state.NotFoundError
	if errors.As(err, &nf) {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// handleStatic 托管 embed 前端: / 与 /console 返回 index.html, 其余按静态文件, 未命中回落 index.html。
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" || name == "console" {
		name = "index.html"
	}
	data, err := fs.ReadFile(s.dist, name)
	if err != nil {
		data, err = fs.ReadFile(s.dist, "index.html")
		if err != nil {
			http.Error(w, "frontend dist missing; run build.ps1", http.StatusNotFound)
			return
		}
		name = "index.html"
	}
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(name, ".json"):
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}

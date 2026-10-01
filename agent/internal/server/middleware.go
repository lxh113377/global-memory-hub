package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"fenjue-agent/internal/logx"
)

// middleware 安全四件套: Host 头校验 -> Origin 白名单+CORS -> OPTIONS 短路 -> token 校验。
// 绑定只在 127.0.0.1 (见 main)。
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logx.Error("panic on %s: %v", r.URL.Path, rec)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "internal panic; see agent log"})
			}
		}()
		// 3) Host 头校验: 仅允许本机回环主机名+端口。
		host127 := fmt.Sprintf("127.0.0.1:%d", s.port)
		hostLocal := fmt.Sprintf("localhost:%d", s.port)
		if r.Host != host127 && r.Host != hostLocal {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": fmt.Sprintf("forbidden Host header %q; use %s or %s", r.Host, host127, hostLocal),
			})
			return
		}
		// 1) Origin 白名单 + 4) CORS (仅白名单 Origin 回 ACAO)。
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !s.origins[origin] {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"ok":    false,
					"error": "forbidden Origin " + origin + "; add it via --origin",
				})
				return
			}
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Headers", "X-Fenjue-Token, Content-Type")
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		// OPTIONS 预检直接 204 短路 (在 token 之前)。
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// 2) token 校验: 仅 /api/* 要求 (/api/health 豁免);
		// 静态页(/ 与 /console)必须免 token 可达 —— 握手链接靠 URL fragment 带给前端。
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" {
			got := r.Header.Get("X-Fenjue-Token")
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
				w.Header().Set("WWW-Authenticate", "X-Fenjue-Token")
				writeJSON(w, http.StatusUnauthorized, map[string]any{
					"ok":    false,
					"error": "missing or invalid X-Fenjue-Token header; open the console link printed at startup",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

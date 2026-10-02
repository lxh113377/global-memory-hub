package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/safeio"
)

// handleRootsSet POST /api/roots: 持久化自定义库根到 ~/.fenjue/state/roots.json。
// 保存即返回; 运行中的 cfg 已解析, 重启 agent 后生效 (loadRootsOverride 会读取)。
func (s *Server) handleRootsSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use POST")
		return
	}
	var body struct {
		Memory string `json:"memory"`
		Skills string `json:"skills"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	saved := map[string]string{}
	for key, raw := range map[string]string{"memory": body.Memory, "skills": body.Skills} {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		exp, err := safeio.ExpandPath(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("expand %s root %q: %v", key, v, err))
			return
		}
		if filepath.Clean(exp) == "" || strings.ContainsAny(filepath.Clean(exp), "<>") {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("invalid %s root %q", key, v))
			return
		}
		saved[key] = exp
	}
	if len(saved) == 0 {
		writeErr(w, http.StatusBadRequest, "body must include at least one of memory/skills")
		return
	}
	if err := os.MkdirAll(safeio.StateDir(), 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("create state dir: %v", err))
		return
	}
	target := filepath.Join(safeio.StateDir(), "roots.json")
	data, _ := json.MarshalIndent(saved, "", "  ")
	if err := safeio.WriteFile(target, data, 0o600); err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Sprintf("write roots.json: %v", err))
		return
	}
	logx.Info("roots override saved: %v", saved)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"applied": false,
		"note":    "saved; restart fenjue-agent to take effect",
		"roots":   saved,
	})
}

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/safeio"
)

// keysOf 返回 map 的键(已排序), 让审计行的 detail 是确定的 —— 审计若因遍历顺序
// 而每次不同, 就没法用 diff 比对两次审计。
func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

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
		// ADR-001: 路径策略判定取代原先"非空 + 不含 <>"两条弱校验。
		// 原校验允许把库根指向任意路径, 随后 enable 会在那里 seed.Bootstrap 写入 ——
		// 写入口交给 Agent 后, 这等于以本机权限任意写盘。
		if err := s.rootPolicy().Check(exp); err != nil {
			logx.Audit("roots.set", exp, false, key+": "+err.Error())
			writeErr(w, http.StatusBadRequest, err.Error())
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
	logx.Audit("roots.set", filepath.Join(safeio.StateDir(), "roots.json"), true, fmt.Sprintf("keys=%v", keysOf(saved)))
	logx.Info("roots override saved: %v", saved)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"applied": false,
		"note":    "saved; restart fenjue-agent to take effect",
		"roots":   saved,
	})
}

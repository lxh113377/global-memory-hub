package server

import (
	"errors"
	"net/http"
	"strings"

	"fenjue-agent/internal/state"
)

// 命名预设的 HTTP 接口。形状与平台操作保持一致:
//   GET  /api/presets            列出预设 (名字、标签、成员)
//   POST /api/presets/{name}     对预设内每个成员执行 action
//
// 批量动作的响应体是逐端结果 (state.PresetResult), 不是单个 OpResult:
// 用户按下按钮后需要知道**每一端**的结果, 而不是一句总的 ok。

type presetListItem struct {
	Name      string   `json:"name"`
	Label     string   `json:"label"`
	Note      string   `json:"note"`
	Platforms []string `json:"platforms"`
}

type presetOpBody struct {
	Action string `json:"action"`
	DryRun bool   `json:"dry_run"`
}

// handlePresets GET /api/presets: 列出全部预设。没有预设时返回空数组而不是 404,
// 因为"你没配预设"是正常状态, 不是错误。
func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use GET")
		return
	}
	items := make([]presetListItem, 0, len(s.cfg.Presets))
	for _, p := range s.cfg.Presets {
		items = append(items, presetListItem{
			Name:      p.Name,
			Label:     p.Label,
			Note:      p.Note,
			Platforms: p.Platforms,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"presets": items,
		"total":   len(items),
	})
}

// handlePresetOp POST /api/presets/{name}, body {"action":"enable"|"disable","dry_run":bool}
// action 缺省为 enable; dry_run 缺省为 false。
func (s *Server) handlePresetOp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use POST")
		return
	}
	name := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/presets/"), "/")
	if name == "" || strings.Contains(name, "/") {
		writeErr(w, http.StatusNotFound, "expected /api/presets/{name}")
		return
	}
	var body presetOpBody
	if err := decodeBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	action := strings.TrimSpace(body.Action)
	if action == "" {
		action = state.PresetActionEnable
	}
	// action 先在这里校验一次, 让非法值落到 400 (请求错) 而不是 500 (服务端错)。
	if action != state.PresetActionEnable && action != state.PresetActionDisable {
		writeErr(w, http.StatusBadRequest,
			"action must be \"enable\" or \"disable\", got "+quote(action))
		return
	}
	res, err := state.ApplyPreset(s.cfg, name, action, body.DryRun)
	if err != nil {
		respondPresetErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func respondPresetErr(w http.ResponseWriter, err error) {
	var nf *state.PresetNotFoundError
	if errors.As(err, &nf) {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

// quote 只为把 action 原样回显在错误信息里 (避免手写引号转义)。
func quote(s string) string {
	return "\"" + s + "\""
}

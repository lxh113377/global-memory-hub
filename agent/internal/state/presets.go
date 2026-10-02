package state

import (
	"fmt"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/platform"
)

// 命名预设的批量编排层。
//
// 为什么不新写一套挂载逻辑: 直接复用 Enable/Disable, 于是备份、回滚、幂等、注入标记
// 这些语义自动与单端操作**完全一致**。若预设另走一条实现, 迟早会出现"单端能回滚、
// 预设不能"这类分裂, 而分裂一旦发生就很难在测试里看出来。
//
// 成员来自 platforms.json 的 presets 段, 代码不内置任何分组语义: 哪些端该一起开,
// 只有用户自己知道。

const (
	PresetActionEnable  = "enable"
	PresetActionDisable = "disable"
)

// PresetNotFoundError 未知预设名。独立于 NotFoundError, 因为错误文案要说的是 preset。
type PresetNotFoundError struct{ Preset string }

func (e *PresetNotFoundError) Error() string { return fmt.Sprintf("unknown preset %q", e.Preset) }

// PresetMember 单个成员的执行结果。
type PresetMember struct {
	ID       string   `json:"id"`
	Action   string   `json:"action"`
	OK       bool     `json:"ok"`
	BackupID string   `json:"backupId,omitempty"`
	Changes  []string `json:"changes,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// PresetResult 一次批量动作的逐端结果。
//
// 语义约定: 部分成员失败时**继续执行其余成员**, 整体 OK 记 false 并在 members 里逐条给出原因。
// 批量动作中途停下会让用户拿到一个"到底开了哪些"的糊涂账。
type PresetResult struct {
	Preset  string         `json:"preset"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	DryRun  bool           `json:"dryRun"`
	OK      bool           `json:"ok"`
	Members []PresetMember `json:"members"`
}

// ApplyPreset 对预设内的每个成员执行 action。
//
// dryRun=true 时只解析成员并返回"将要做什么", 不碰任何文件: 批量动作影响面大,
// 用户需要先看清要动哪些端再按下按钮。
func ApplyPreset(cfg *platform.Config, name, action string, dryRun bool) (*PresetResult, error) {
	if action != PresetActionEnable && action != PresetActionDisable {
		return nil, fmt.Errorf("preset %s: action must be %s or %s, got %q", name, PresetActionEnable, PresetActionDisable, action)
	}
	p := cfg.FindPreset(name)
	if p == nil {
		return nil, &PresetNotFoundError{Preset: name}
	}
	res := &PresetResult{
		Preset:  p.Name,
		Label:   p.Label,
		Action:  action,
		DryRun:  dryRun,
		OK:      true,
		Members: make([]PresetMember, 0, len(p.Platforms)),
	}
	for _, id := range p.Platforms {
		m := PresetMember{ID: id, Action: action, OK: true}
		if dryRun {
			m.Changes = []string{"dry-run: would " + action + " " + id}
			res.Members = append(res.Members, m)
			continue
		}
		var op *OpResult
		var err error
		if action == PresetActionEnable {
			op, err = Enable(cfg, id)
		} else {
			// 预设的批量停用一律走软关闭 (摘链接 + 注入改停用短壳), 与控制台默认行为一致。
			// 硬关闭会删掉注入块, 那是单端、逐条确认才该做的动作, 不该被一个批量按钮顺手做掉。
			op, err = Disable(cfg, id, true)
		}
		if err != nil {
			m.OK = false
			m.Error = err.Error()
			res.OK = false
		} else {
			m.BackupID = op.BackupID
			m.Changes = op.Changes
		}
		res.Members = append(res.Members, m)
	}
	failed := 0
	for _, m := range res.Members {
		if !m.OK {
			failed++
		}
	}
	logx.Info("preset %s action=%s dryRun=%v members=%d failed=%d", p.Name, action, dryRun, len(res.Members), failed)
	return res, nil
}
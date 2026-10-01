package state

import (
	"fmt"
	"os"
	"path/filepath"

	"fenjue-agent/internal/inject"
	"fenjue-agent/internal/mount"
	"fenjue-agent/internal/platform"
)

// 状态四态 + SKIP (对齐 API 契约)。
const (
	StatusOK       = "OK"
	StatusBroken   = "BROKEN" // 目标不可达
	StatusMismatch = "MISMATCH"
	StatusMissing  = "MISSING" // 挂载点不存在
	StatusSkip     = "SKIP"    // 该端 home 不存在, 不计异常
)

// MountState 单条挂载的探活结果。
type MountState struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

// InjectState 单条注入的探活结果。
type InjectState struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// PlatformState 单个平台的探活结果。
type PlatformState struct {
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Support string        `json:"support"`
	Status  string        `json:"status"`
	Mounts  []MountState  `json:"mounts"`
	Inject  []InjectState `json:"inject"`
}

// Snapshot /api/state 的响应体。
type Snapshot struct {
	Roots     map[string]string `json:"roots"`
	Platforms []PlatformState   `json:"platforms"`
}

// BuildSnapshot 全量探活。
func BuildSnapshot(cfg *platform.Config) *Snapshot {
	snap := &Snapshot{Roots: cfg.Roots}
	for _, p := range cfg.Platforms {
		ps := PlatformState{ID: p.ID, Label: p.Label, Support: p.Support, Status: StatusOK}
		if !p.OSSupported || !p.HomeKnown || !dirExists(p.Home) {
			ps.Status = StatusSkip
			for _, m := range p.Mounts {
				ps.Mounts = append(ps.Mounts, MountState{From: m.From, To: m.To, Kind: m.Kind, Status: StatusSkip})
			}
			for _, inj := range p.Injects {
				ps.Inject = append(ps.Inject, InjectState{Path: inj.Path, Status: StatusSkip})
			}
			snap.Platforms = append(snap.Platforms, ps)
			continue
		}
		worst := StatusOK
		for _, m := range p.Mounts {
			st := statusMount(m)
			ps.Mounts = append(ps.Mounts, MountState{From: m.From, To: m.To, Kind: m.Kind, Status: st})
			worst = worse(worst, st)
		}
		for _, inj := range p.Injects {
			st := StatusSkip
			if inj.Resolved {
				st = inject.Probe(inj.Path)
			}
			ps.Inject = append(ps.Inject, InjectState{Path: inj.Path, Status: st})
			worst = worse(worst, st)
		}
		ps.Status = worst
		snap.Platforms = append(snap.Platforms, ps)
	}
	return snap
}

// Report /api/verify 的响应体。
type Report struct {
	Total    int      `json:"total"`
	OK       int      `json:"ok"`
	Broken   []string `json:"broken"`
	Mismatch []string `json:"mismatch"`
	Missing  []string `json:"missing"`
}

// Verify 汇总四态计数 (SKIP 不计入 total)。
func Verify(cfg *platform.Config) *Report {
	r := &Report{Broken: []string{}, Mismatch: []string{}, Missing: []string{}}
	for _, ps := range BuildSnapshot(cfg).Platforms {
		for _, m := range ps.Mounts {
			if m.Status == StatusSkip {
				continue
			}
			r.Total++
			desc := fmt.Sprintf("%s: %s -> %s (%s)", ps.ID, m.From, m.To, m.Kind)
			switch m.Status {
			case StatusOK:
				r.OK++
			case StatusBroken:
				r.Broken = append(r.Broken, desc)
			case StatusMismatch:
				r.Mismatch = append(r.Mismatch, desc)
			case StatusMissing:
				r.Missing = append(r.Missing, desc)
			}
		}
		for _, inj := range ps.Inject {
			if inj.Status == StatusSkip {
				continue
			}
			r.Total++
			desc := fmt.Sprintf("%s: inject %s", ps.ID, inj.Path)
			switch inj.Status {
			case StatusOK:
				r.OK++
			case StatusMismatch:
				r.Mismatch = append(r.Mismatch, desc)
			case StatusMissing:
				r.Missing = append(r.Missing, desc)
			default:
				r.OK++
			}
		}
	}
	return r
}

func statusMount(m platform.Mount) string {
	if !m.Resolved {
		return StatusSkip
	}
	switch m.Kind {
	case "link":
		return statusLink(m.From, m.To)
	case "mirror":
		return statusMirror(m.From, m.To)
	case "per-skill":
		return statusPerSkill(m.From, m.To)
	default:
		return StatusSkip
	}
}

// statusLink 复刻 junction 校验语义: 缺失=MISSING, 非链接/目标不符=MISMATCH, 目标不可达=BROKEN。
func statusLink(from, to string) string {
	info, err := mount.Probe(from)
	if err != nil {
		return StatusMismatch
	}
	if !info.IsLink {
		if !pathExists(from) {
			return StatusMissing
		}
		return StatusMismatch
	}
	if !mount.SameTarget(info.Target, to) {
		return StatusMismatch
	}
	if !pathExists(to) {
		return StatusBroken
	}
	return StatusOK
}

func statusMirror(from, to string) string {
	if !pathExists(from) {
		return StatusMissing
	}
	if !pathExists(to) {
		return StatusBroken
	}
	return StatusOK
}

func statusPerSkill(from, to string) string {
	entries, err := os.ReadDir(to)
	if err != nil {
		return StatusBroken // hub 根不可达
	}
	if _, err := os.Stat(from); err != nil {
		if os.IsNotExist(err) {
			return StatusMissing
		}
		return StatusMismatch
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			continue // 文件不是技能条目
		}
		st := statusLink(filepath.Join(from, e.Name()), filepath.Join(to, e.Name()))
		if st != StatusOK {
			return st
		}
	}
	return StatusOK
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func severity(s string) int {
	switch s {
	case StatusBroken:
		return 4
	case StatusMismatch:
		return 3
	case StatusMissing:
		return 2
	case StatusOK:
		return 1
	default:
		return 0
	}
}

func worse(a, b string) string {
	if severity(b) > severity(a) {
		return b
	}
	return a
}

package state

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fenjue-agent/internal/inject"
	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/mount"
	"fenjue-agent/internal/platform"
	"fenjue-agent/internal/seed"
	"fenjue-agent/internal/safeio"
)

var opMu sync.Mutex

// NotFoundError 未知平台 id。
type NotFoundError struct{ Platform string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("unknown platform %q", e.Platform) }

// OpResult enable/disable/restore 的统一响应体。
type OpResult struct {
	OK       bool     `json:"ok"`
	BackupID string   `json:"backupId"`
	Changes  []string `json:"changes"`
}

// classifyPath 备份分类器: 链接记目标, 文件/目录拷贝, 不存在记 missing。
func classifyPath(path string) (string, string, error) {
	info, err := mount.Probe(path)
	if err != nil {
		return "", "", err
	}
	if info.IsLink {
		return "link", info.Target, nil
	}
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing", "", nil
		}
		return "", "", err
	}
	if fi.IsDir() {
		return "dir", "", nil
	}
	return "file", "", nil
}

var linkOps = safeio.LinkOps{
	Classify:  classifyPath,
	Create:    mount.Create,
	RemoveAny: mount.RemoveAny,
}

// Enable 启用平台: 备份 -> 建链/同步 -> 注入激活标记块。
// 已 OK 的挂载点 no-op (幂等, 绝不破坏已有链接)。
func Enable(cfg *platform.Config, id string) (*OpResult, error) {
	opMu.Lock()
	defer opMu.Unlock()
	p := cfg.FindPlatform(id)
	if p == nil {
		return nil, &NotFoundError{Platform: id}
	}
	if !p.OSSupported {
		return nil, fmt.Errorf("platform %s does not support %s", id, runtime.GOOS)
	}
	if !p.HomeKnown {
		return nil, fmt.Errorf("platform %s: home unresolved (%s); set the env var or fix platforms.json", id, p.HomeWarn)
	}
	changes := []string{}
	if !dirExists(p.Home) {
		if err := os.MkdirAll(p.Home, 0o755); err != nil {
			return nil, fmt.Errorf("platform %s: create home %q: %w", id, p.Home, err)
		}
		changes = append(changes, "created home dir "+p.Home)
	}
	// 统一库引导: 目录不存在则建, 空库播种; 已有库零覆盖 (幂等)。
	seedChanges, err := seed.Bootstrap(cfg.Roots)
	if err != nil {
		return nil, fmt.Errorf("platform %s: %w", id, err)
	}
	changes = append(changes, seedChanges...)
	backupID, err := backupFor(p)
	if err != nil {
		return nil, err
	}
	for _, m := range p.Mounts {
		if !m.Resolved {
			changes = append(changes, "skip unresolved mount on platform "+id)
			continue
		}
		switch m.Kind {
		case "link":
			if err := enableLink(p, m, &changes); err != nil {
				return nil, err
			}
		case "mirror":
			// 仅当同步过(synced 标记)且当前状态 OK 才 no-op; 从未同步过的即使目录已存在也要真同步。
			if fileExists(mirrorSyncedPath(id)) && statusMirror(m.From, m.To) == StatusOK {
				clearMirrorOff(id)
				changes = append(changes, "mirror already in sync, no-op: "+m.From+" -> "+m.To)
				continue
			}
			n, err := mount.MirrorSync(m.From, m.To)
			if err != nil {
				return nil, fmt.Errorf("platform %s mirror %q -> %q: %w", id, m.From, m.To, err)
			}
			if err := writeMirrorSynced(id, m.From, m.To); err != nil {
				return nil, err
			}
			clearMirrorOff(id)
			changes = append(changes, fmt.Sprintf("mirror synced %d files: %s -> %s", n, m.From, m.To))
		case "per-skill":
			if statusPerSkill(m.From, m.To) == StatusOK {
				changes = append(changes, "per-skill already OK, no-op: "+m.From)
				continue
			}
			n, err := mount.PerSkillEnable(m.From, m.To)
			if err != nil {
				return nil, fmt.Errorf("platform %s per-skill %q: %w", id, m.From, err)
			}
			changes = append(changes, fmt.Sprintf("per-skill links ready: %d items in %s", n, m.From))
		default:
			changes = append(changes, "skip unknown mount kind "+m.Kind)
		}
	}
	for _, inj := range p.Injects {
		if !inj.Resolved {
			changes = append(changes, "skip unresolved inject on platform "+id)
			continue
		}
		if err := inject.Apply(inj.Path, activeContent(p), true); err != nil {
			return nil, fmt.Errorf("platform %s inject %q: %w", id, inj.Path, err)
		}
		changes = append(changes, "inject updated (active): "+inj.Path)
	}
	logx.Info("enable platform %s: backupId=%s changes=%d", id, backupID, len(changes))
	return &OpResult{OK: true, BackupID: backupID, Changes: changes}, nil
}

func enableLink(p *platform.Platform, m platform.Mount, changes *[]string) error {
	info, err := mount.Probe(m.From)
	if err != nil {
		return fmt.Errorf("platform %s probe %q: %w", p.ID, m.From, err)
	}
	if info.IsLink && mount.SameTarget(info.Target, m.To) {
		*changes = append(*changes, "link already OK, no-op: "+m.From+" -> "+m.To)
		return nil
	}
	if info.IsLink {
		if err := mount.Remove(m.From); err != nil {
			return fmt.Errorf("platform %s relink %q: %w", p.ID, m.From, err)
		}
		*changes = append(*changes, "relinked (target changed): "+m.From)
	} else if _, err := os.Lstat(m.From); err == nil {
		// 已有真实文件/目录: 已随备份留底, 此处替换为链接。
		if err := mount.RemoveAny(m.From); err != nil {
			return fmt.Errorf("platform %s replace %q: %w", p.ID, m.From, err)
		}
		*changes = append(*changes, "replaced existing entry (backed up): "+m.From)
	}
	if err := mount.Create(m.From, m.To); err != nil {
		return fmt.Errorf("platform %s create %q -> %q: %w", p.ID, m.From, m.To, err)
	}
	*changes = append(*changes, "link created: "+m.From+" -> "+m.To)
	return nil
}

// Disable 停用平台: soft=true 摘链+注入改停用短壳; soft=false 摘链+删段。
func Disable(cfg *platform.Config, id string, soft bool) (*OpResult, error) {
	opMu.Lock()
	defer opMu.Unlock()
	p := cfg.FindPlatform(id)
	if p == nil {
		return nil, &NotFoundError{Platform: id}
	}
	changes := []string{}
	if !p.OSSupported || !p.HomeKnown || !dirExists(p.Home) {
		changes = append(changes, "platform home missing or unsupported on "+runtime.GOOS+"; nothing to disable")
		return &OpResult{OK: true, BackupID: "", Changes: changes}, nil
	}
	backupID, err := backupFor(p)
	if err != nil {
		return nil, err
	}
	for _, m := range p.Mounts {
		if !m.Resolved {
			continue
		}
		switch m.Kind {
		case "link":
			info, err := mount.Probe(m.From)
			if err != nil {
				return nil, fmt.Errorf("platform %s probe %q: %w", id, m.From, err)
			}
			if info.IsLink {
				if err := mount.Remove(m.From); err != nil {
					return nil, fmt.Errorf("platform %s remove link %q: %w", id, m.From, err)
				}
				changes = append(changes, "link removed: "+m.From)
			} else {
				changes = append(changes, "no link at "+m.From+" (left untouched)")
			}
		case "per-skill":
			n, err := mount.PerSkillDisable(m.From)
			if err != nil {
				return nil, fmt.Errorf("platform %s per-skill disable %q: %w", id, m.From, err)
			}
			changes = append(changes, fmt.Sprintf("per-skill links removed: %d", n))
		case "mirror":
			if err := writeMirrorOff(id, m.From); err != nil {
				return nil, err
			}
			changes = append(changes, "mirror sync stopped (soft-off marker): "+m.From)
		}
	}
	for _, inj := range p.Injects {
		if !inj.Resolved {
			continue
		}
		if soft {
			if err := inject.Apply(inj.Path, "", false); err != nil {
				return nil, fmt.Errorf("platform %s inject soft-off %q: %w", id, inj.Path, err)
			}
			changes = append(changes, "inject soft-disabled: "+inj.Path)
		} else {
			removed, err := inject.RemoveBlock(inj.Path)
			if err != nil {
				return nil, fmt.Errorf("platform %s inject remove %q: %w", id, inj.Path, err)
			}
			changes = append(changes, fmt.Sprintf("inject block removed (%v): %s", removed, inj.Path))
		}
	}
	logx.Info("disable platform %s soft=%v: backupId=%s changes=%d", id, soft, backupID, len(changes))
	return &OpResult{OK: true, BackupID: backupID, Changes: changes}, nil
}

// Restore 从备份还原 (注入语义: 删段+回滚备份, 由整文件回滚天然覆盖)。
func Restore(backupID string) (*OpResult, error) {
	opMu.Lock()
	defer opMu.Unlock()
	if strings.TrimSpace(backupID) == "" {
		return nil, fmt.Errorf("restore: backupId is required")
	}
	n, err := safeio.Restore(backupID, linkOps)
	if err != nil {
		return nil, fmt.Errorf("restore backup %s: %w", backupID, err)
	}
	logx.Info("restore backup %s: %d items", backupID, n)
	return &OpResult{OK: true, BackupID: backupID, Changes: []string{fmt.Sprintf("restored %d items from backup %s", n, backupID)}}, nil
}

func backupFor(p *platform.Platform) (string, error) {
	var paths []string
	for _, m := range p.Mounts {
		if m.Resolved {
			paths = append(paths, m.From)
		}
	}
	for _, inj := range p.Injects {
		if inj.Resolved {
			paths = append(paths, inj.Path)
		}
	}
	id, err := safeio.NewBackup(paths, classifyPath)
	if err != nil {
		return "", fmt.Errorf("platform %s backup: %w", p.ID, err)
	}
	return id, nil
}

func activeContent(p *platform.Platform) string {
	var b strings.Builder
	b.WriteString("# Fenjue Agent Injection (auto-managed)\n\n")
	b.WriteString("- platform: " + p.ID + " (" + p.Label + ")\n")
	b.WriteString("- managed by fenjue-agent; edits inside FENJUE markers are overwritten\n")
	for _, m := range p.Mounts {
		if m.Resolved {
			b.WriteString("- mount " + m.Kind + ": " + m.From + " -> " + m.To + "\n")
		}
	}
	return b.String()
}

func mirrorOffPath(id string) string {
	return filepath.Join(safeio.StateDir(), id+".mirror.off")
}

func mirrorSyncedPath(id string) string {
	return filepath.Join(safeio.StateDir(), id+".mirror.synced")
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func writeMarker(path, content string) error {
	if err := safeio.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write state marker %q: %w", path, err)
	}
	return nil
}

func writeMirrorOff(id, from string) error {
	content := fmt.Sprintf("disabled-at=%s\nfrom=%s\n", time.Now().Format(time.RFC3339), from)
	return writeMarker(mirrorOffPath(id), content)
}

func writeMirrorSynced(id, from, to string) error {
	content := fmt.Sprintf("synced-at=%s\nfrom=%s\nto=%s\n", time.Now().Format(time.RFC3339), from, to)
	return writeMarker(mirrorSyncedPath(id), content)
}

func clearMirrorOff(id string) { os.Remove(mirrorOffPath(id)) }

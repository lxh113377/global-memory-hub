package safeio

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Classifier 判定路径对象类型: "file" / "dir" / "link" / "missing"。
// link 时返回 linkTarget。由调用方注入以识别 junction/symlink(平台相关)。
type Classifier func(path string) (kind string, linkTarget string, err error)

// LinkOps 注入平台相关链接操作, 供 Restore 重建链接。
type LinkOps struct {
	Classify  Classifier
	Create    func(from, to string) error
	RemoveAny func(path string) error
}

// BackupItem 备份清单条目。
type BackupItem struct {
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	LinkTarget string `json:"linkTarget,omitempty"`
	DataDir    string `json:"dataDir,omitempty"`
}

type backupManifest struct {
	ID        string       `json:"id"`
	CreatedAt string       `json:"createdAt"`
	Items     []BackupItem `json:"items"`
}

// TrashRoot 备份根目录 ~/.fenjue/trash。
func TrashRoot() string { return filepath.Join(FenjueHome(), "trash") }

// StateDir 状态目录 ~/.fenjue/state。
func StateDir() string { return filepath.Join(FenjueHome(), "state") }

// TokenPath token 文件路径 ~/.fenjue/token。
func TokenPath() string { return filepath.Join(FenjueHome(), "token") }

var backupIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("safeio: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// NewBackup 快照 paths 中所有存在的对象到 ~/.fenjue/trash/<id>/, 返回 backupId。
// 链接只记录目标不拷贝内容; 文件/目录整体拷贝(树内链接跳过不跟随)。
func NewBackup(paths []string, classify Classifier) (string, error) {
	if classify == nil {
		return "", fmt.Errorf("safeio: backup classifier is required")
	}
	id := time.Now().Format("20060102-150405") + "-" + randHex(3)
	dir := filepath.Join(TrashRoot(), id)
	dataRoot := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return "", fmt.Errorf("safeio: create trash dir %q: %w", dir, err)
	}
	m := backupManifest{ID: id, CreatedAt: time.Now().Format(time.RFC3339)}
	for i, p := range paths {
		kind, target, err := classify(p)
		if err != nil {
			return "", fmt.Errorf("safeio: classify %q: %w", p, err)
		}
		if kind == "missing" {
			continue
		}
		item := BackupItem{Path: p, Kind: kind, LinkTarget: target}
		if kind == "file" || kind == "dir" {
			sub := fmt.Sprintf("item-%03d", i)
			dst := filepath.Join(dataRoot, sub)
			if kind == "file" {
				if err := copyFile(p, dst); err != nil {
					return "", err
				}
			} else {
				if err := copyTree(p, dst, classify); err != nil {
					return "", fmt.Errorf("safeio: backup dir %q: %w", p, err)
				}
			}
			item.DataDir = sub
		}
		m.Items = append(m.Items, item)
	}
	raw, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("safeio: marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
		return "", fmt.Errorf("safeio: write manifest: %w", err)
	}
	return id, nil
}

func copyFile(src, dst string) error {
	if err := CheckReservedName(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("safeio: mkdir for %q: %w", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("safeio: open %q: %w", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("safeio: create %q: %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("safeio: copy %q -> %q: %w", src, dst, err)
	}
	return nil
}

func copyTree(src, dst string, classify Classifier) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			return rerr
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		kind, _, cerr := classify(p)
		if cerr != nil {
			return cerr
		}
		switch kind {
		case "link":
			// 树内链接不跟随、不复制, 防止把链接目标内容拷进来。
			return nil
		case "dir":
			return os.MkdirAll(target, 0o755)
		default:
			return copyFile(p, target)
		}
	})
}

// Restore 按 manifest 还原 backupId 对应的全部对象, 返回还原条数。
func Restore(backupID string, ops LinkOps) (int, error) {
	if !backupIDRe.MatchString(backupID) {
		return 0, fmt.Errorf("safeio: invalid backupId %q", backupID)
	}
	if ops.Classify == nil || ops.Create == nil || ops.RemoveAny == nil {
		return 0, fmt.Errorf("safeio: restore link ops incomplete")
	}
	dir := filepath.Join(TrashRoot(), backupID)
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return 0, fmt.Errorf("safeio: read manifest of backup %q: %w", backupID, err)
	}
	var m backupManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0, fmt.Errorf("safeio: parse manifest of backup %q: %w", backupID, err)
	}
	restored := 0
	for _, item := range m.Items {
		switch item.Kind {
		case "link":
			if err := ops.RemoveAny(item.Path); err != nil {
				return restored, fmt.Errorf("safeio: clear %q before link restore: %w", item.Path, err)
			}
			if err := os.MkdirAll(filepath.Dir(item.Path), 0o755); err != nil {
				return restored, fmt.Errorf("safeio: mkdir for %q: %w", item.Path, err)
			}
			if err := ops.Create(item.Path, item.LinkTarget); err != nil {
				return restored, fmt.Errorf("safeio: recreate link %q -> %q: %w", item.Path, item.LinkTarget, err)
			}
		case "file":
			src := filepath.Join(dir, "data", item.DataDir)
			if err := copyFile(src, item.Path); err != nil {
				return restored, fmt.Errorf("safeio: restore file %q: %w", item.Path, err)
			}
		case "dir":
			// 若当前位置是链接, 先摘链, 避免内容透写进链接目标。
			if kind, _, e := ops.Classify(item.Path); e == nil && kind == "link" {
				ops.RemoveAny(item.Path)
			}
			src := filepath.Join(dir, "data", item.DataDir)
			if err := copyTree(src, item.Path, ops.Classify); err != nil {
				return restored, fmt.Errorf("safeio: restore dir %q: %w", item.Path, err)
			}
		default:
			continue
		}
		restored++
	}
	return restored, nil
}

package mount

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"fenjue-agent/internal/logx"
)

// MirrorSync cx 物理镜像: 单向复制 from -> to (合并式覆盖, 不删除 to 中多余文件)。
// 返回复制的文件数。mirror 是真实复制, 不产生任何链接。
func MirrorSync(from, to string) (int, error) {
	fi, err := os.Stat(from)
	if err != nil {
		return 0, fmt.Errorf("mount: mirror source %q: %w", from, err)
	}
	if !fi.IsDir() {
		return 0, fmt.Errorf("mount: mirror source %q is not a directory", from)
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return 0, fmt.Errorf("mount: mirror target %q: %w", to, err)
	}
	count := 0
	err = filepath.WalkDir(from, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, rerr := filepath.Rel(from, p)
		if rerr != nil {
			return rerr
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(to, rel)
		if d.Type()&os.ModeSymlink != 0 {
			logx.Warn("mount: mirror skips link entry %q", p)
			return nil
		}
		// junction 在 Lstat/DirEntry 下不带 ModeSymlink, 用 Readlink 兜底识别。
		if _, lerr := os.Readlink(p); lerr == nil {
			logx.Warn("mount: mirror skips junction entry %q", p)
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if err := copyFile(p, dst); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return count, fmt.Errorf("mount: mirror sync %q -> %q: %w", from, to, err)
	}
	return count, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mount: mkdir %q: %w", filepath.Dir(dst), err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("mount: open %q: %w", src, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("mount: create %q: %w", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("mount: copy %q -> %q: %w", src, dst, err)
	}
	return nil
}

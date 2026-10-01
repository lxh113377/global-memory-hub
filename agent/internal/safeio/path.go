// Package safeio 路径规范化(~ / %VAR% / $VAR 展开)、Windows 保留名拦截、时间戳备份与还原。
package safeio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envVarRe = regexp.MustCompile(`%([A-Za-z0-9_]+)%`)

// HomeDir 返回用户主目录, 优先 os.UserHomeDir, 回退 USERPROFILE / HOME。
func HomeDir() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	if h := os.Getenv("USERPROFILE"); h != "" {
		return h
	}
	return os.Getenv("HOME")
}

// FenjueHome 返回 ~/.fenjue 绝对路径。
func FenjueHome() string { return filepath.Join(HomeDir(), ".fenjue") }

// ExpandPath 展开 %VAR%、$VAR/${VAR} 与前导 ~, 返回绝对路径。
func ExpandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("safeio: empty path")
	}
	if strings.Contains(p, "%") {
		p = envVarRe.ReplaceAllStringFunc(p, func(m string) string {
			if v := os.Getenv(m[1:len(m)-1]); v != "" {
				return v
			}
			return m
		})
	}
	p = os.ExpandEnv(p)
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home := HomeDir()
		if home == "" {
			return "", errors.New("safeio: cannot resolve home directory")
		}
		rest := ""
		if len(p) > 2 {
			rest = p[2:]
		}
		p = filepath.Join(home, filepath.FromSlash(rest))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("safeio: abs %q: %w", p, err)
	}
	return abs, nil
}

var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// CheckReservedName 拦截 Windows 保留设备名(任一路径段命中即报错)。
func CheckReservedName(path string) error {
	for _, seg := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		base := strings.ToUpper(seg)
		if i := strings.IndexByte(base, '.'); i >= 0 {
			base = base[:i]
		}
		if reservedNames[base] {
			return fmt.Errorf("safeio: %q contains Windows reserved device name %q", path, base)
		}
	}
	return nil
}

// WriteFile 带保留名检查与父目录创建的写文件。
func WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := CheckReservedName(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("safeio: mkdir for %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("safeio: write %q: %w", path, err)
	}
	return nil
}

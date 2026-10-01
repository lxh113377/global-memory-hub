// Package inject 标记块注入引擎: <!-- FENJUE:BEGIN --> ... <!-- FENJUE:END -->。
// 启用=插入或更新段; 软关闭=段内替换为停用短壳(保留标记保证幂等); 还原=删段+回滚备份。
// 绝不整文件改写: 只在标记段内替换或文末追加。
package inject

import (
	"fmt"
	"os"
	"strings"

	"fenjue-agent/internal/safeio"
)

// 标记块边界。
const (
	Begin = "<!-- FENJUE:BEGIN -->"
	End   = "<!-- FENJUE:END -->"
)

const disabledNote = "<!-- fenjue soft-disabled by fenjue-agent; run enable to restore. -->"

// Apply 插入或更新 path 中的标记块。active=false 写入停用短壳。
// 文件不存在时创建为仅含标记块的新文件。
func Apply(path, content string, active bool) error {
	var block string
	if active {
		block = Begin + "\n" + strings.TrimRight(content, "\n") + "\n" + End + "\n"
	} else {
		block = Begin + "\n" + disabledNote + "\n" + End + "\n"
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("inject: read %q: %w", path, err)
		}
		return safeio.WriteFile(path, []byte(block), 0o644)
	}
	text := string(existing)
	updated, found := replaceBlock(text, block)
	if !found {
		trimmed := strings.TrimRight(text, "\n")
		if trimmed == "" {
			updated = block
		} else {
			updated = trimmed + "\n\n" + block
		}
	}
	return safeio.WriteFile(path, []byte(updated), 0o644)
}

// replaceBlock 就地替换标记段(含损坏段: 有头无尾时截到文末); 无段时返回原文与 false。
func replaceBlock(text, block string) (string, bool) {
	i := strings.Index(text, Begin)
	if i < 0 {
		return text, false
	}
	j := strings.Index(text[i:], End)
	if j < 0 {
		return text[:i] + block, true
	}
	return text[:i] + block + text[i+j+len(End):], true
}

// RemoveBlock 删除标记段(含标记本身)。文件不存在或无段时返回 false。
func RemoveBlock(path string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("inject: read %q: %w", path, err)
	}
	text := string(existing)
	i := strings.Index(text, Begin)
	if i < 0 {
		return false, nil
	}
	j := strings.Index(text[i:], End)
	var updated string
	if j < 0 {
		updated = text[:i]
	} else {
		updated = text[:i] + text[i+j+len(End):]
	}
	if err := safeio.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// Probe 返回注入状态: OK(激活段) / MISMATCH(停用壳或文件无段) / MISSING(文件不存在)。
func Probe(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "MISSING"
	}
	text := string(data)
	i := strings.Index(text, Begin)
	if i < 0 {
		return "MISMATCH"
	}
	j := strings.Index(text[i:], End)
	if j < 0 {
		return "MISMATCH"
	}
	if strings.Contains(text[i+len(Begin):i+j], "soft-disabled") {
		return "MISMATCH"
	}
	return "OK"
}

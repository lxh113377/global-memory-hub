package mount

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// perSkillConcurrency 逐技能树挂载的并发上限 (platforms.json notes 约定 8)。
const perSkillConcurrency = 8

// PerSkillEnable 逐技能树挂载: 对 to(hub 根)的每个非文件条目, 在 from 下建一个链接。
// 先在临时目录构建全部链接(并发上限 8), 成功后整体换入 from;
// 任一失败则删除临时目录回滚, 原有 from 目录不动。返回就位的条目数。
func PerSkillEnable(from, to string) (int, error) {
	entries, err := os.ReadDir(to)
	if err != nil {
		return 0, fmt.Errorf("mount: per-skill read hub root %q: %w", to, err)
	}
	var names []string
	for _, e := range entries {
		// 只挂目录型条目(含指向目录的 junction/symlink); 普通文件不是技能。
		if e.Type().IsRegular() {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		if err := os.MkdirAll(from, 0o755); err != nil {
			return 0, fmt.Errorf("mount: per-skill mkdir %q: %w", from, err)
		}
		return 0, nil
	}
	tmp := from + ".fenjue-tmp"
	old := from + ".fenjue-old"
	if err := RemoveAny(tmp); err != nil {
		return 0, fmt.Errorf("mount: per-skill clean tmp %q: %w", tmp, err)
	}
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return 0, fmt.Errorf("mount: per-skill mkdir tmp %q: %w", tmp, err)
	}
	errCh := make(chan error, len(names))
	sem := make(chan struct{}, perSkillConcurrency)
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			linkPath := filepath.Join(tmp, n)
			target := filepath.Join(to, n)
			if err := Create(linkPath, target); err != nil {
				errCh <- fmt.Errorf("per-skill link %q: %w", linkPath, err)
			}
		}(name)
	}
	wg.Wait()
	close(errCh)
	var firstErr error
	for e := range errCh {
		if firstErr == nil {
			firstErr = e
		}
	}
	if firstErr != nil {
		RemoveAny(tmp)
		return 0, fmt.Errorf("mount: per-skill build failed (rolled back, %q untouched): %w", from, firstErr)
	}
	// 换入: 旧目录挪走 -> tmp 换入 -> 失败则挪回。
	hadOld := false
	if _, err := os.Lstat(from); err == nil {
		if err := RemoveAny(old); err != nil {
			RemoveAny(tmp)
			return 0, fmt.Errorf("mount: per-skill clean old %q: %w", old, err)
		}
		if err := os.Rename(from, old); err != nil {
			RemoveAny(tmp)
			return 0, fmt.Errorf("mount: per-skill move aside %q: %w", from, err)
		}
		hadOld = true
	} else if !os.IsNotExist(err) {
		RemoveAny(tmp)
		return 0, fmt.Errorf("mount: per-skill lstat %q: %w", from, err)
	}
	if err := os.Rename(tmp, from); err != nil {
		if hadOld {
			os.Rename(old, from)
		}
		RemoveAny(tmp)
		return 0, fmt.Errorf("mount: per-skill swap in %q: %w", from, err)
	}
	if hadOld {
		RemoveAny(old)
	}
	return len(names), nil
}

// PerSkillDisable 摘除 from 下所有链接条目, 返回摘除数。
// 非链接条目不是本工具创建的, 不碰。from 不存在时返回 0。
func PerSkillDisable(from string) (int, error) {
	entries, err := os.ReadDir(from)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("mount: per-skill read %q: %w", from, err)
	}
	removed := 0
	for _, e := range entries {
		p := filepath.Join(from, e.Name())
		info, err := Probe(p)
		if err != nil {
			return removed, fmt.Errorf("mount: per-skill probe %q: %w", p, err)
		}
		if !info.IsLink {
			continue
		}
		if err := Remove(p); err != nil {
			return removed, fmt.Errorf("mount: per-skill remove %q: %w", p, err)
		}
		removed++
	}
	return removed, nil
}

package safeio

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// isWindows 路径比较的 case 敏感性开关。用 runtime.GOOS 而非 build tag,
// 是为了不为这一个判断再拆一对文件; 代价是交叉编译时它会按目标平台求值,
// 那正是我们要的语义。
func isWindows() bool { return runtime.GOOS == "windows" }

// RootPolicy 规定「库根可以放在哪里」。
//
// 背景: POST /api/roots 原本只校验「非空 + 不含 <>」, 于是可以把统一库指向任意路径,
// 随后 enable 会在那个路径上 seed.Bootstrap 写入。写入口一旦交给 Agent, 这等于
// 「以本机权限任意写盘」。本文件把那条弱校验换成前缀包含判定。
//
// 三条设计约束:
//   - 判定必须防前缀混淆: C:\Users\foo 不得放行 C:\Users\foobar。故用「相等 或 以 prefix+分隔符 开头」,
//     不用裸 strings.HasPrefix。
//   - Windows 下比较大小写不敏感。
//   - 默认允许集 = 用户主目录 + ~/.fenjue; 放宽必须显式声明, 不做「猜」。
type RootPolicy struct {
	// Allowed 是允许的根前缀, 已 Clean。空表示「什么都不允许」。
	Allowed []string
}

// DefaultRootPolicy 返回默认策略: 用户主目录与 ~/.fenjue。
// 两者都在主目录内, 故实际效果是「只许写用户家目录之内」——这是最贴近普通用户心智的默认,
// 也是收紧校验时最容易解释的一条。
func DefaultRootPolicy() RootPolicy {
	h := HomeDir()
	allowed := []string{}
	if h != "" {
		allowed = append(allowed, filepath.Clean(h))
	}
	allowed = append(allowed, filepath.Clean(FenjueHome()))
	return RootPolicy{Allowed: allowed}
}

// WithRoots 返回追加了额外允许前缀的副本(用于 --allow-root)。传入项会被展开与 Clean。
func (p RootPolicy) WithRoots(extra ...string) RootPolicy {
	out := RootPolicy{Allowed: append([]string{}, p.Allowed...)}
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if abs, err := ExpandPath(e); err == nil {
			e = abs
		}
		out.Allowed = append(out.Allowed, filepath.Clean(e))
	}
	return out
}

// Contains 判定 p 是否落在任一允许前缀内。
// 注意先 Clean 再比, 否则 "…/foo/../bar" 这类路径能绕过字面比较。
func (p RootPolicy) Contains(target string) bool {
	if target == "" || len(p.Allowed) == 0 {
		return false
	}
	t := filepath.Clean(target)
	for _, a := range p.Allowed {
		a = filepath.Clean(a)
		if a == "" || a == "." {
			continue
		}
		if equalPath(t, a) {
			return true
		}
		if hasPathPrefix(t, a) {
			return true
		}
	}
	return false
}

// Check 返回 nil 表示通过; 否则给出可直接照抄的修复指引(必须点名放宽开关, 否则排查者无从下手)。
func (p RootPolicy) Check(target string) error {
	if p.Contains(target) {
		return nil
	}
	return fmt.Errorf("root %q is outside the allowed set %v; "+
		"keep it under your home directory, or start the agent with "+
		"--allow-root <prefix> (repeatable) to widen the policy explicitly", target, p.Allowed)
}

// equalPath: Windows 下大小写不敏感, 其余平台按原样。
func equalPath(a, b string) bool {
	if isWindows() {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// hasPathPrefix: t 是否在 a 之下。要求分隔符边界, 防 C:\Users\foo 匹配 C:\Users\foobar。
func hasPathPrefix(t, a string) bool {
	if equalPath(t, a) {
		return true
	}
	sep := string(filepath.Separator)
	if !strings.HasSuffix(a, sep) {
		a += sep
	}
	tt, ta := t, a
	if isWindows() {
		tt, ta = strings.ToLower(t), strings.ToLower(a)
	}
	return strings.HasPrefix(tt, ta)
}
package safeio

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 本测试是 rootpolicy 的判据载体。遵循本仓 §5 硬前置: 每条判据都要有阳性对照,
// 且必须带「反例腿」——只测「该放行的放行了」等于没测。

func tempHome(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestRootPolicyContainsUnderHome(t *testing.T) {
	home := tempHome(t)
	p := RootPolicy{Allowed: []string{filepath.Clean(home)}}
	ok := []string{
		home,                              // 相等
		filepath.Join(home, "library"),    // 直接子目录
		filepath.Join(home, "a", "b", "c"), // 深层
	}
	for _, c := range ok {
		if !p.Contains(c) {
			t.Fatalf("positive control failed: %q under %q should be allowed", c, home)
		}
	}
}

func TestRootPolicyRejectsPrefixConfusion(t *testing.T) {
	// 反例腿: 裸 strings.HasPrefix 会在这里放行。home 是 t.TempDir() 的子路径,
	// 所以用同级的 "…-evil" 变体构造最贴近真实的误放行场景。
	parent := tempHome(t)
	home := filepath.Join(parent, "lib")
	evil := filepath.Join(parent, "lib-evil")
	p := RootPolicy{Allowed: []string{filepath.Clean(home)}}
	if p.Contains(evil) {
		t.Fatalf("prefix confusion: %q must NOT be allowed by prefix %q", evil, home)
	}
}

func TestRootPolicyRejectsOutsideAndCleanedEscape(t *testing.T) {
	home := tempHome(t)
	p := RootPolicy{Allowed: []string{filepath.Clean(home)}}
	bad := []string{
		filepath.Join(home, "..", "escaped"), // Clean 后落在允许集外
		filepath.Dir(home),                    // 上一级
		string(filepath.Separator),            // 根
		"",                                    // 空
	}
	for _, b := range bad {
		if p.Contains(b) {
			t.Fatalf("escape control failed: %q (clean=%q) must NOT be allowed", b, filepath.Clean(b))
		}
	}
}

func TestRootPolicyEmptyAllowedDeniesEverything(t *testing.T) {
	p := RootPolicy{}
	if p.Contains(tempHome(t)) {
		t.Fatal("empty Allowed must deny, not default-allow")
	}
}

func TestRootPolicyWithRootsWidensExplicitly(t *testing.T) {
	outside := filepath.Join(tempHome(t), "elsewhere")
	p := DefaultRootPolicy()
	if p.Contains(outside) {
		t.Fatal("precondition failed: default policy should not already contain it")
	}
	p2 := p.WithRoots(outside)
	if !p2.Contains(outside) {
		t.Fatalf("--allow-root should widen explicitly: %q", outside)
	}
	// 放宽不得连带放宽同级兄弟目录。
	sibling := outside + "-sibling"
	if p2.Contains(sibling) {
		t.Fatalf("widening leaked to sibling: %q", sibling)
	}
}

func TestRootPolicyCheckErrorNamesTheFlag(t *testing.T) {
	p := DefaultRootPolicy()
	err := p.Check(filepath.Join(tempHome(t), "nope"))
	if err == nil {
		t.Fatal("expected rejection")
	}
	// 错误信息必须点名放宽开关, 否则排查者只看到 invalid root 而不知如何合法化。
	if !strings.Contains(err.Error(), "--allow-root") {
		t.Fatalf("error must name --allow-root, got: %v", err)
	}
}

func TestRootPolicyWindowsCaseInsensitive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case sensitivity rule only applies on windows")
	}
	home := os.Getenv("USERPROFILE")
	if home == "" {
		t.Skip("no USERPROFILE")
	}
	p := RootPolicy{Allowed: []string{filepath.Clean(home)}}
	if !p.Contains(strings.ToUpper(home)) {
		t.Fatal("windows path comparison must be case-insensitive")
	}
}
package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fenjue-agent/internal/mount"
	"fenjue-agent/internal/platform"
)

// testCfg 造一个完全落在临时目录里的配置。
//
// 关键点: 必须把 USERPROFILE 与 HOME 都指到临时目录。Enable 会写备份到 ~/.fenjue/trash,
// 而测试如果不重定向, 就会往使用者真实的统一库里写备份 —— 测试污染生产数据是最难查的一类问题。
func testCfg(t *testing.T) *platform.Config {
	t.Helper()
	base := t.TempDir()
	t.Setenv("USERPROFILE", base)
	t.Setenv("HOME", base)

	roots := map[string]string{
		"memory": filepath.Join(base, "library", "memory"),
		"skills": filepath.Join(base, "library", "skills"),
	}
	cfg := &platform.Config{
		Schema:     platform.ExpectedSchema,
		Version:    "1.1.0",
		SourcePath: filepath.Join(base, "platforms.json"),
		Roots:      roots,
	}
	for _, id := range []string{"a", "b"} {
		home := filepath.Join(base, "homes", id)
		if err := os.MkdirAll(home, 0o755); err != nil {
			t.Fatalf("mkdir home %s: %v", home, err)
		}
		cfg.Platforms = append(cfg.Platforms, &platform.Platform{
			ID:          id,
			Label:       id,
			Support:     "ga",
			OSSupported: true,
			Home:        home,
			HomeKnown:   true,
			Mounts: []platform.Mount{{
				Kind:     "link",
				From:     filepath.Join(home, "skills"),
				To:       roots["skills"],
				Resolved: true,
			}},
			Injects: []platform.Inject{{
				Path:     filepath.Join(home, "AGENTS.md"),
				Mode:     "marker",
				Resolved: true,
			}},
		})
	}
	cfg.Presets = []*platform.Preset{
		{Name: "both", Label: "Both ends", Platforms: []string{"a", "b"}},
	}
	return cfg
}

func linkExists(t *testing.T, path string) bool {
	t.Helper()
	info, err := mount.Probe(path)
	if err != nil {
		return false
	}
	return info.IsLink
}

func TestApplyPresetUnknown(t *testing.T) {
	cfg := testCfg(t)
	_, err := ApplyPreset(cfg, "nope", PresetActionEnable, false)
	if err == nil {
		t.Fatal("an unknown preset must be an error, not a silent no-op")
	}
	var nf *PresetNotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want PresetNotFoundError, got %T: %v", err, err)
	}
}

func TestApplyPresetBadAction(t *testing.T) {
	cfg := testCfg(t)
	if _, err := ApplyPreset(cfg, "both", "wipe", false); err == nil {
		t.Fatal("an unknown action must be rejected before anything is touched")
	}
	// 反例断言: 拒绝之后不能留下任何链接。
	for _, p := range cfg.Platforms {
		if linkExists(t, p.Mounts[0].From) {
			t.Fatalf("platform %s got linked despite a rejected action", p.ID)
		}
	}
}

func TestApplyPresetDryRunTouchesNothing(t *testing.T) {
	cfg := testCfg(t)
	res, err := ApplyPreset(cfg, "both", PresetActionEnable, true)
	if err != nil {
		t.Fatalf("dry-run must not fail: %v", err)
	}
	if !res.OK || len(res.Members) != 2 {
		t.Fatalf("dry-run must report every member: ok=%v members=%d", res.OK, len(res.Members))
	}
	for _, p := range cfg.Platforms {
		if linkExists(t, p.Mounts[0].From) {
			t.Fatalf("dry-run created a link on %s", p.ID)
		}
		if _, err := os.Stat(p.Mounts[0].From); err == nil {
			t.Fatalf("dry-run created the mount point on %s", p.ID)
		}
	}
}

func TestApplyPresetEnableThenDisable(t *testing.T) {
	cfg := testCfg(t)

	res, err := ApplyPreset(cfg, "both", PresetActionEnable, false)
	if err != nil {
		t.Fatalf("enable preset: %v", err)
	}
	if !res.OK {
		t.Fatalf("enable preset reported failure: %+v", res.Members)
	}
	for _, p := range cfg.Platforms {
		if !linkExists(t, p.Mounts[0].From) {
			t.Fatalf("platform %s was not linked by the preset", p.ID)
		}
		if m := res.Members[0]; m.BackupID == "" {
			t.Fatal("each member must report a backupId so the batch can be rolled back")
		}
	}

	dis, err := ApplyPreset(cfg, "both", PresetActionDisable, false)
	if err != nil {
		t.Fatalf("disable preset: %v", err)
	}
	if !dis.OK {
		t.Fatalf("disable preset reported failure: %+v", dis.Members)
	}
	for _, p := range cfg.Platforms {
		if linkExists(t, p.Mounts[0].From) {
			t.Fatalf("platform %s is still linked after a preset disable", p.ID)
		}
	}
	// 软关闭的语义: 注入块被改写成停用短壳, 而不是被删掉。
	for _, p := range cfg.Platforms {
		data, err := os.ReadFile(p.Injects[0].Path)
		if err != nil {
			t.Fatalf("inject block missing on %s after a soft disable: %v", p.ID, err)
		}
		if len(data) == 0 {
			t.Fatalf("soft disable must leave a disabled shell, not an empty file, on %s", p.ID)
		}
	}
}

func TestApplyPresetIsIdempotent(t *testing.T) {
	cfg := testCfg(t)
	if _, err := ApplyPreset(cfg, "both", PresetActionEnable, false); err != nil {
		t.Fatalf("first enable: %v", err)
	}
	res, err := ApplyPreset(cfg, "both", PresetActionEnable, false)
	if err != nil {
		t.Fatalf("second enable must be a no-op, not an error: %v", err)
	}
	if !res.OK {
		t.Fatalf("second enable reported failure: %+v", res.Members)
	}
	for _, p := range cfg.Platforms {
		if !linkExists(t, p.Mounts[0].From) {
			t.Fatalf("re-running the preset broke the existing link on %s", p.ID)
		}
	}
}

func TestApplyPresetContinuesAfterOneMemberFails(t *testing.T) {
	cfg := testCfg(t)
	// 一个 home 未解析的平台: Enable 会拒绝它。
	broken := &platform.Platform{
		ID:          "broken",
		Label:       "Broken",
		Support:     "ga",
		OSSupported: true,
		HomeKnown:   false,
		HomeWarn:    "home unresolved in this test",
	}
	cfg.Platforms = append(cfg.Platforms, broken)
	cfg.Presets[0].Platforms = []string{"broken", "a", "b"}

	res, err := ApplyPreset(cfg, "both", PresetActionEnable, false)
	if err != nil {
		t.Fatalf("a single failing member must not abort the whole call: %v", err)
	}
	if res.OK {
		t.Fatal("overall ok must be false when any member failed")
	}
	if len(res.Members) != 3 {
		t.Fatalf("every member must be reported, got %d", len(res.Members))
	}
	if res.Members[0].OK || res.Members[0].Error == "" {
		t.Fatalf("the failing member must carry a reason: %+v", res.Members[0])
	}
	for _, p := range cfg.Platforms[:2] {
		if !linkExists(t, p.Mounts[0].From) {
			t.Fatalf("platform %s must still be processed after an earlier member failed", p.ID)
		}
	}
}

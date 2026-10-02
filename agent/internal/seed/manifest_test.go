package seed

import (
	"encoding/json"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

// clonePack 把内嵌 pack 复制成可改写的 MapFS, 供篡改反例使用。
func clonePack(t *testing.T) fstest.MapFS {
	t.Helper()
	out := fstest.MapFS{}
	err := fs.WalkDir(packFS, packRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := packFS.ReadFile(p)
		if err != nil {
			return err
		}
		out[p] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatalf("clone pack: %v", err)
	}
	return out
}

func withPackFS(t *testing.T, replacement packReader) {
	t.Helper()
	original := packFS
	packFS = replacement
	t.Cleanup(func() { packFS = original })
}

// TestVerifyPackClean 是正例腿: 未被篡改的包必须通过校验。
func TestVerifyPackClean(t *testing.T) {
	if err := VerifyPack(); err != nil {
		t.Fatalf("clean pack must pass verification, got: %v", err)
	}
}

// TestVerifyPackDetectsTamperedContent 是篡改反例腿: 内容被改一个字节也必须红。
// 判据的价值全在这条腿: 没有它, VerifyPack 只是一段没人验证过的代码。
func TestVerifyPackDetectsTamperedContent(t *testing.T) {
	m := clonePack(t)
	const target = packRoot + "/memory/MEMORY.md"
	original := m[target]
	m[target] = &fstest.MapFile{Data: append([]byte("#"), original.Data...)}
	withPackFS(t, m)

	err := VerifyPack()
	if err == nil {
		t.Fatal("tampered pack must fail verification")
	}
	if !contains(err.Error(), "sha256") && !contains(err.Error(), "size") {
		t.Fatalf("error must name the failing check, got: %v", err)
	}
}

// TestVerifyPackDetectsMissingFile: 清单登记了但文件不在, 必须红。
func TestVerifyPackDetectsMissingFile(t *testing.T) {
	m := clonePack(t)
	delete(m, packRoot+"/skills/brainstorming/SKILL.md")
	withPackFS(t, m)

	err := VerifyPack()
	if err == nil {
		t.Fatal("missing pack file must fail verification")
	}
	if !contains(err.Error(), "not in the pack") {
		t.Fatalf("error must report the missing file, got: %v", err)
	}
}

// TestVerifyPackDetectsUndeclaredFile: 文件在包里但清单没登记, 必须红。
// 少了这条, 绕过清单往包里塞内容就不会被发现。
func TestVerifyPackDetectsUndeclaredFile(t *testing.T) {
	m := clonePack(t)
	m[packRoot+"/skills/smuggled/SKILL.md"] = &fstest.MapFile{Data: []byte("not declared")}
	withPackFS(t, m)

	err := VerifyPack()
	if err == nil {
		t.Fatal("undeclared pack file must fail verification")
	}
	if !contains(err.Error(), "undeclared file") {
		t.Fatalf("error must report the undeclared file, got: %v", err)
	}
}

// TestVerifyPackDetectsVersionDrift: 清单里的 seed_version 与编译期常量不一致, 必须红。
// 这条正是版本漂移的机器拦截点: 清单与代码不同步时不允许播种。
func TestVerifyPackDetectsVersionDrift(t *testing.T) {
	m := clonePack(t)
	raw, err := m.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var doc manifestDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	doc.SeedVersion = "99.99.99-not-the-compiled-one"
	patched, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	m[manifestPath] = &fstest.MapFile{Data: patched}
	withPackFS(t, m)

	verr := VerifyPack()
	if verr == nil {
		t.Fatal("seed version drift must fail verification")
	}
	if !contains(verr.Error(), "SeedVersion") {
		t.Fatalf("error must name the version drift, got: %v", verr)
	}
}

// TestVerifyPackDetectsMissingManifest: 清单本身缺失, 必须红 (不得当成通过)。
func TestVerifyPackDetectsMissingManifest(t *testing.T) {
	m := clonePack(t)
	delete(m, manifestPath)
	withPackFS(t, m)

	if err := VerifyPack(); err == nil {
		t.Fatal("missing manifest must fail verification, never pass silently")
	}
}

// TestBootstrapRefusesTamperedPack 端到端反例: 包被篡改时 Bootstrap 必须一个文件都不写。
func TestBootstrapRefusesTamperedPack(t *testing.T) {
	roots := newRoots(t)
	m := clonePack(t)
	const target = packRoot + "/memory/MEMORY.md"
	m[target] = &fstest.MapFile{Data: append([]byte("#"), m[target].Data...)}
	withPackFS(t, m)

	if _, err := Bootstrap(roots); err == nil {
		t.Fatal("bootstrap must refuse a tampered pack")
	}
	if _, err := os.Stat(roots["memory"]); err == nil {
		t.Fatal("bootstrap must not create the root when verification fails")
	}
}

package seed

// 种子包完整性凭据: 播种前对内嵌 pack 做全量 SHA-256 校验。
//
// 存在理由: Bootstrap 的零覆盖铁律保证不破坏用户已有数据, 但无法证明写入内容
// 本身未被篡改。分发链路上一份带哈希的清单, 才能让"写进去的东西"也可验证。
//
// 铁律:
//   - 校验发生在任何写入之前 (Bootstrap 入口), 失败即 fail-closed, 不留半成品。
//   - 双向对账: 清单登记但缺失, 与实际存在但未登记, 都判为差异 (防绕过登记偷渡内容)。
//   - 已存在且非空的库仍走零覆盖, 不受校验结果影响 (校验只管"要写的内容")。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
)

const (
	manifestName   = "manifest.json"
	manifestSchema = "fenjue-seed-manifest-v1"
	manifestPath   = "pack/" + manifestName
)

// packRoot 是内嵌 FS 中种子包的根目录。
const packRoot = "pack"

type manifestEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type manifestDoc struct {
	Schema      string          `json:"schema"`
	SeedVersion string          `json:"seed_version"`
	Files       []manifestEntry `json:"files"`
}

// VerifyPack 对内嵌种子包做全量完整性校验。任一不符即返回错误。
func VerifyPack() error {
	doc, err := loadManifest()
	if err != nil {
		return err
	}
	if doc.SeedVersion != SeedVersion {
		return fmt.Errorf("seed manifest version %q does not match compiled SeedVersion %q; "+
			"regenerate with: python scripts/build_seed.py --manifest-only",
			doc.SeedVersion, SeedVersion)
	}

	declared := make(map[string]manifestEntry, len(doc.Files))
	for _, e := range doc.Files {
		if e.Path == "" || e.Path == manifestName {
			return fmt.Errorf("seed manifest has an empty or self-referential path: %q", e.Path)
		}
		if _, dup := declared[e.Path]; dup {
			return fmt.Errorf("seed manifest lists %q twice", e.Path)
		}
		declared[e.Path] = e
	}

	actual, err := listPackFiles()
	if err != nil {
		return err
	}

	var problems []string
	for rel := range actual {
		entry, ok := declared[rel]
		if !ok {
			problems = append(problems, fmt.Sprintf("undeclared file in pack: %s", rel))
			continue
		}
		sum, size, err := hashPackFile(rel)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if size != entry.Bytes {
			problems = append(problems, fmt.Sprintf("%s: size %d bytes, manifest says %d", rel, size, entry.Bytes))
		}
		if !strings.EqualFold(sum, entry.SHA256) {
			problems = append(problems, fmt.Sprintf("%s: sha256 %s, manifest says %s", rel, sum, entry.SHA256))
		}
	}
	for _, e := range doc.Files {
		if _, ok := actual[e.Path]; !ok {
			problems = append(problems, fmt.Sprintf("manifest lists %s but it is not in the pack", e.Path))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("seed pack integrity check failed (%d problem(s)):\n  - %s",
			len(problems), strings.Join(problems, "\n  - "))
	}
	return nil
}

func loadManifest() (*manifestDoc, error) {
	data, err := packFS.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("seed manifest missing (%s): %w; regenerate with: python scripts/build_seed.py --manifest-only",
			manifestPath, err)
	}
	var doc manifestDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("seed manifest is not valid JSON: %w", err)
	}
	if doc.Schema != manifestSchema {
		return nil, fmt.Errorf("seed manifest schema %q, want %q", doc.Schema, manifestSchema)
	}
	if len(doc.Files) == 0 {
		return nil, fmt.Errorf("seed manifest lists no files")
	}
	return &doc, nil
}

// listPackFiles 列出内嵌 pack 的全部文件 (正斜杠相对路径), 不含 manifest 自身。
func listPackFiles() (map[string]struct{}, error) {
	out := map[string]struct{}{}
	err := fs.WalkDir(packFS, packRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, packRoot+"/")
		if rel == manifestName {
			return nil
		}
		out[rel] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk seed pack: %w", err)
	}
	return out, nil
}

func hashPackFile(rel string) (sum string, size int64, err error) {
	full := path.Join(packRoot, rel)
	f, err := packFS.Open(full)
	if err != nil {
		return "", 0, fmt.Errorf("open pack file %s: %w", rel, err)
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("read pack file %s: %w", rel, err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fenjue-agent/internal/platform"
)

// cmdExport 是给脚本与 Agent 用的**只读**出口。
//
// 为什么需要它: 控制台只能人在浏览器里点, 而"现在到底接入了哪些端、统一库里有多少
// 技能"这类问题经常需要被脚本或另一个 Agent 回答。
//
// 安全边界 (本轮刻意收窄到最小可用面)
// ------------------------------------
//  1. **只读**: 只做 Stat / ReadDir / Read, 不创建、不修改、不删除任何东西。
//     连日志都不写 —— 一个承诺只读的命令不应该在用户主目录留下痕迹。
//  2. **不联网、不监听端口、不需要令牌**: 直接读库, 不经过 HTTP, 因此不存在
//     "令牌怎么交给 Agent" 这个还没定论的问题 (那是 G24 的写入口, 本轮明确不做)。
//  3. **默认不含正文**: 只导出元数据。正文必须显式 --include-content, 因为把记忆正文
//     打进日志或粘贴到聊天里是最容易发生的一次意外泄漏。
//  4. **绝不导出令牌**: token 文件路径在库根之外, 这里也不会去读它。
func cmdExport(args []string) {
	set, cf := newFlags("export")
	includeContent := set.Bool("include-content", false, "include file bodies (off by default: bodies leak easily)")
	format := set.String("format", "json", "json | markdown")
	set.Parse(args)
	_ = cf // 平台定义路径由 mustConfig 解析

	cfg := mustConfig(*cf.platforms)

	exp := ExportView{
		SchemaVersion: "fenjue-export-v1",
		GeneratedAt:   time.Now().Format(time.RFC3339),
		PlatformFile:  cfg.SourcePath,
		Roots:         cfg.Roots,
		Platforms:     exportPlatforms(cfg),
	}
	exp.Library = walkLibrary(cfg.Roots, *includeContent)
	if len(cfg.Presets) > 0 {
		exp.Presets = exportPresets(cfg)
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(exp); err != nil {
			fmt.Fprintln(os.Stderr, "error: encode export:", err)
			os.Exit(1)
		}
	case "markdown", "md":
		fmt.Print(exp.Markdown())
	default:
		fmt.Fprintf(os.Stderr, "error: unsupported --format %q; use json or markdown\n", *format)
		os.Exit(2)
	}
}

// ---- 视图结构 (字段名即给脚本的契约, 改字段名等于改 API) ----

type ExportPlatform struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Support   string   `json:"support"`
	HomeKnown bool     `json:"homeKnown"`
	Mounts    []string `json:"mounts"`
	Injects   int      `json:"injects"`
}

type ExportFile struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Body  string `json:"body,omitempty"`
}

type ExportLibrary struct {
	MemoryRoot   string       `json:"memoryRoot"`
	SkillsRoot   string       `json:"skillsRoot"`
	Files        int          `json:"files"`
	Bytes        int64        `json:"bytes"`
	Skills       int          `json:"skills"`
	SkillDirs    int          `json:"skillDirs"`
	MemoryFiles  int          `json:"memoryFiles"`
	NewestChange string       `json:"newestChange,omitempty"`
	Content      []ExportFile `json:"content,omitempty"`
	Note         string       `json:"note,omitempty"`
}

type ExportPreset struct {
	Name      string   `json:"name"`
	Label     string   `json:"label"`
	Platforms []string `json:"platforms"`
}

type ExportView struct {
	SchemaVersion string            `json:"schemaVersion"`
	GeneratedAt   string            `json:"generatedAt"`
	PlatformFile  string            `json:"platformFile"`
	Roots         map[string]string `json:"roots"`
	Platforms     []ExportPlatform  `json:"platforms"`
	Presets       []ExportPreset    `json:"presets,omitempty"`
	Library       ExportLibrary     `json:"library"`
}

// Markdown 面向人: 贴进 issue 或对话里能直接读。
func (v ExportView) Markdown() string {
	var b strings.Builder
	b.WriteString("# 焚诀 Global 导出行\n\n")
	b.WriteString("- 生成时间：")
	b.WriteString(v.GeneratedAt)
	b.WriteString("\n- 平台定义：")
	b.WriteString(v.PlatformFile)
	b.WriteString("\n\n## 统一库\n\n")
	fmt.Fprintf(&b, "- 记忆根：`%s`\n", v.Library.MemoryRoot)
	fmt.Fprintf(&b, "- 技能根：`%s`\n", v.Library.SkillsRoot)
	fmt.Fprintf(&b, "- 文件 %d 个 / %d 字节；技能 %d 个（目录 %d 个）；记忆文件 %d 个\n",
		v.Library.Files, v.Library.Bytes, v.Library.Skills, v.Library.SkillDirs, v.Library.MemoryFiles)
	if v.Library.NewestChange != "" {
		fmt.Fprintf(&b, "- 最近变更：%s\n", v.Library.NewestChange)
	}
	if v.Library.Note != "" {
		fmt.Fprintf(&b, "- 提示：%s\n", v.Library.Note)
	}
	b.WriteString("\n## 接入端\n\n| 端 | 名称 | 支持 | 挂载形态 | 注入项 |\n|---|---|---|---|---|\n")
	for _, p := range v.Platforms {
		mounts := strings.Join(p.Mounts, ", ")
		if mounts == "" {
			mounts = "-"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %d |\n", p.ID, p.Label, p.Support, mounts, p.Injects)
	}
	if len(v.Presets) > 0 {
		b.WriteString("\n## 预设\n\n")
		for _, pr := range v.Presets {
			fmt.Fprintf(&b, "- `%s`（%s）：%s\n", pr.Name, pr.Label, strings.Join(pr.Platforms, ", "))
		}
	}
	return b.String()
}

// ---- 采集 ----

func exportPlatforms(cfg *platform.Config) []ExportPlatform {
	out := make([]ExportPlatform, 0, len(cfg.Platforms))
	for _, p := range cfg.Platforms {
		kinds := make([]string, 0, len(p.Mounts))
		for _, m := range p.Mounts {
			kinds = append(kinds, m.Kind)
		}
		out = append(out, ExportPlatform{
			ID:        p.ID,
			Label:     p.Label,
			Support:   p.Support,
			HomeKnown: p.HomeKnown,
			Mounts:    kinds,
			Injects:   len(p.Injects),
		})
	}
	return out
}

func exportPresets(cfg *platform.Config) []ExportPreset {
	out := make([]ExportPreset, 0, len(cfg.Presets))
	for _, p := range cfg.Presets {
		out = append(out, ExportPreset{Name: p.Name, Label: p.Label, Platforms: p.Platforms})
	}
	return out
}

// walkLibrary 统计两个库根。includeContent 为真时把正文一并带出。
func walkLibrary(roots map[string]string, includeContent bool) ExportLibrary {
	lib := ExportLibrary{
		MemoryRoot: roots["memory"],
		SkillsRoot: roots["skills"],
	}
	if lib.MemoryRoot == "" {
		lib.Note = "no memory root configured"
	}
	skillsByDir := 0
	skillDirs := 0

	if lib.SkillsRoot != "" {
		entries, err := os.ReadDir(lib.SkillsRoot)
		if err != nil {
			lib.Note = joinNote(lib.Note, "skills root unavailable: "+err.Error())
		} else {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				skillDirs++
				skillFile := filepath.Join(lib.SkillsRoot, e.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					skillsByDir++
				}
				files, err := listFiles(filepath.Join(lib.SkillsRoot, e.Name()))
				if err != nil {
					continue
				}
				lib = accumulate(lib, files, includeContent)
			}
		}
	}
	lib.SkillDirs = skillDirs
	lib.Skills = skillsByDir

	if lib.MemoryRoot != "" {
		files, err := listFiles(lib.MemoryRoot)
		if err != nil {
			lib.Note = joinNote(lib.Note, "memory root unavailable: "+err.Error())
		} else {
			lib = accumulate(lib, files, includeContent)
			lib.MemoryFiles = len(files)
		}
	}
	return lib
}

func accumulate(lib ExportLibrary, files []string, includeContent bool) ExportLibrary {
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		lib.Files++
		lib.Bytes += info.Size()
		if newest := info.ModTime().UTC().Format(time.RFC3339); newest > lib.NewestChange {
			lib.NewestChange = newest
		}
		if includeContent {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			lib.Content = append(lib.Content, ExportFile{Path: f, Bytes: int64(len(data)), Body: string(data)})
		}
	}
	return lib
}

// listFiles 递归列出目录下的文件 (不跟随链接), 路径排序保证输出可复算。
func listFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
}

func joinNote(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

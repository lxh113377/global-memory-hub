// Package platform 解析 platforms.json (schema fenjue-platforms-v1) 并按 runtime.GOOS 展开路径。
// 占位符: <skills>/<memory> 展开 roots; <HERMES_HOME> 等按 homeEnv/homeDefault 解析; ~ / %VAR% 由 safeio 展开。
package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/safeio"
)

// ExpectedSchema 期望的 schema 值。
const ExpectedSchema = "fenjue-platforms-v1"

type rawMount struct {
	Kind string `json:"kind"`
	From string `json:"from"`
	To   string `json:"to"`
}

type rawInject struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type rawPlatform struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Support     string      `json:"support"`
	OS          []string    `json:"os"`
	Home        string      `json:"home"`
	HomeEnv     string      `json:"homeEnv"`
	HomeDefault string      `json:"homeDefault"`
	Mounts      []rawMount  `json:"mounts"`
	Inject      []rawInject `json:"inject"`
	Notes       string      `json:"notes"`
}

type rawSite struct {
	Primary string `json:"primary"`
	Mirror  string `json:"mirror"`
	Local   string `json:"local"`
}

// rawPreset 命名预设: 一组平台 id 的批量启停。字段全部可选, 缺省即没有预设。
//
// 为什么成员由配置决定而不是写死在代码里: 哪些端该一起开, 只有用户自己知道。
// 把它写死成 "ga 全集" 之类的语义, 换个人、换个平台矩阵就得改代码, 而且改错了没法从
// 使用现场看出来。这里刻意只提供机制, 不提供默认语义。
type rawPreset struct {
	Name      string   `json:"name"`
	Label     string   `json:"label"`
	Platforms []string `json:"platforms"`
	Note      string   `json:"note"`
}

type rawFile struct {
	Schema    string            `json:"schema"`
	Version   string            `json:"version"`
	Roots     map[string]string `json:"roots"`
	Site      *rawSite          `json:"site"`
	Platforms []rawPlatform     `json:"platforms"`
	Presets   []rawPreset       `json:"presets"`
}

// Mount 已展开的挂载计划。
type Mount struct {
	Kind     string
	From     string
	To       string
	Resolved bool
}

// Inject 已展开的注入计划。
type Inject struct {
	Path     string
	Mode     string
	Resolved bool
}

// Platform 解析后的平台定义。
type Platform struct {
	ID              string
	Label           string
	Support         string
	OSSupported     bool
	Home            string
	HomeKnown       bool
	HomeWarn        string
	homePlaceholder string
	Mounts          []Mount
	Injects         []Inject
}

// Config platforms.json 的运行时视图。
type Config struct {
	Schema     string
	Version    string
	SourcePath string
	Roots      map[string]string
	Site       SiteInfo
	Platforms  []*Platform
	Presets    []*Preset
	raw        *rawFile
}

// Preset 命名预设: 对一组平台做批量启停。语义由配置决定, 代码不内置任何分组。
type Preset struct {
	Name      string
	Label     string
	Note      string
	Platforms []string
}

// SiteInfo 站点地址段 (site.primary/mirror/local), 供握手链接与 Origin 白名单消费。
type SiteInfo struct {
	Primary string
	Mirror  string
	Local   string
}

var placeholderRe = regexp.MustCompile(`<([A-Za-z][A-Za-z0-9_]*)>`)

// Load 读取并解析 platforms.json。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("platform: read %s: %w", path, err)
	}
	var f rawFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("platform: parse %s: %w", path, err)
	}
	if f.Schema != "" && f.Schema != ExpectedSchema {
		return nil, fmt.Errorf("platform: %s: unsupported schema %q (expect %q)", path, f.Schema, ExpectedSchema)
	}
	cfg := &Config{Schema: f.Schema, Version: f.Version, SourcePath: path, raw: &f}
	if f.Site != nil {
		cfg.Site = SiteInfo{Primary: f.Site.Primary, Mirror: f.Site.Mirror, Local: f.Site.Local}
	}
	cfg.resolve(loadRootsOverride())
	presets, err := parsePresets(cfg, f.Presets)
	if err != nil {
		return nil, err
	}
	cfg.Presets = presets
	return cfg, nil
}

// parsePresets 校验并展开 presets。
//
// 未知成员 id 是**配置错误**, 因此直接让 Load 失败: 预设是批量动作, 少一个端却静默跳过,
// 用户会在"以为全开了"的假象里继续用下去。重复成员去重 (顺序保留), 空名或空成员表报错。
func parsePresets(cfg *Config, raws []rawPreset) ([]*Preset, error) {
	out := make([]*Preset, 0, len(raws))
	seen := map[string]bool{}
	for i, rp := range raws {
		name := strings.TrimSpace(rp.Name)
		if name == "" {
			return nil, fmt.Errorf("platform: presets[%d]: name is required", i)
		}
		if seen[name] {
			return nil, fmt.Errorf("platform: presets[%d]: duplicate preset name %q", i, name)
		}
		seen[name] = true
		if len(rp.Platforms) == 0 {
			return nil, fmt.Errorf("platform: preset %q: platforms must not be empty", name)
		}
		p := &Preset{Name: name, Label: strings.TrimSpace(rp.Label), Note: strings.TrimSpace(rp.Note)}
		if p.Label == "" {
			p.Label = name
		}
		memberSeen := map[string]bool{}
		for _, pid := range rp.Platforms {
			id := strings.TrimSpace(pid)
			if id == "" {
				continue
			}
			if cfg.FindPlatform(id) == nil {
				return nil, fmt.Errorf("platform: preset %q references unknown platform %q", name, id)
			}
			if memberSeen[id] {
				continue
			}
			memberSeen[id] = true
			p.Platforms = append(p.Platforms, id)
		}
		if len(p.Platforms) == 0 {
			return nil, fmt.Errorf("platform: preset %q: platforms has no usable id", name)
		}
		out = append(out, p)
	}
	return out, nil
}

// loadRootsOverride 读 ~/.fenjue/state/roots.json (POST /api/roots 的持久化产物)。
// 任何失败 (缺失/损坏/展开不了) 都返回 nil, 静默回落 platforms.json 的默认 roots。
func loadRootsOverride() map[string]string {
	p := filepath.Join(safeio.FenjueHome(), "state", "roots.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		logx.Warn("platform: parse roots override %s: %v", p, err)
		return nil
	}
	out := map[string]string{}
	for _, key := range []string{"memory", "skills"} {
		v := strings.TrimSpace(m[key])
		if v == "" {
			continue
		}
		exp, err := safeio.ExpandPath(v)
		if err != nil {
			logx.Warn("platform: expand roots override %s=%q: %v", key, v, err)
			continue
		}
		out[key] = exp
	}
	if len(out) == 0 {
		return nil
	}
	logx.Info("platform: roots override applied from %s", p)
	return out
}

// WithRoots 用 override 覆盖 roots 后重新展开全部路径 (enable body {roots?})。
func (c *Config) WithRoots(override map[string]string) *Config {
	c2 := &Config{Schema: c.Schema, Version: c.Version, SourcePath: c.SourcePath, raw: c.raw, Site: c.Site, Presets: c.Presets}
	c2.resolve(override)
	return c2
}

// FindPreset 按 name 查找预设, 未找到返回 nil。
func (c *Config) FindPreset(name string) *Preset {
	for _, p := range c.Presets {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// FindPlatform 按 id 查找平台, 未找到返回 nil。
func (c *Config) FindPlatform(id string) *Platform {
	for _, p := range c.Platforms {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (c *Config) resolve(override map[string]string) {
	roots := make(map[string]string, len(c.raw.Roots))
	for k, v := range c.raw.Roots {
		if override != nil {
			if ov, ok := override[k]; ok && strings.TrimSpace(ov) != "" {
				v = ov
			}
		}
		exp, err := safeio.ExpandPath(v)
		if err != nil {
			logx.Warn("platform: expand root %s=%q: %v", k, v, err)
			exp = v
		}
		roots[k] = exp
	}
	c.Roots = roots

	for i := range c.raw.Platforms {
		rp := &c.raw.Platforms[i]
		p := &Platform{ID: rp.ID, Label: rp.Label, Support: rp.Support}
		p.OSSupported = osSupported(rp.OS)
		c.resolveHome(rp, p)

		for _, rm := range rp.Mounts {
			from, okFrom := c.expandPath(rm.From, p)
			to, okTo := c.expandPath(rm.To, p)
			p.Mounts = append(p.Mounts, Mount{Kind: rm.Kind, From: from, To: to, Resolved: okFrom && okTo})
		}
		for _, ri := range rp.Inject {
			path, ok := c.expandPath(ri.Path, p)
			p.Injects = append(p.Injects, Inject{Path: path, Mode: ri.Mode, Resolved: ok})
		}
		c.Platforms = append(c.Platforms, p)
	}
}

// resolveHome 解析平台 home: 占位符走 homeEnv, 缺失时回落 homeDefault 并显式告警。
func (c *Config) resolveHome(rp *rawPlatform, p *Platform) {
	homeTpl := rp.Home
	if m := placeholderRe.FindStringSubmatch(homeTpl); m != nil {
		ph := "<" + m[1] + ">"
		if rp.HomeEnv != "" {
			if v := os.Getenv(rp.HomeEnv); v != "" {
				homeTpl = strings.ReplaceAll(homeTpl, ph, v)
				p.homePlaceholder = m[1]
			} else if rp.HomeDefault != "" {
				if exp, err := safeio.ExpandPath(rp.HomeDefault); err == nil {
					homeTpl = strings.ReplaceAll(homeTpl, ph, exp)
					p.homePlaceholder = m[1]
					p.HomeWarn = fmt.Sprintf("env %s not set, fallback to %s", rp.HomeEnv, rp.HomeDefault)
				} else {
					p.HomeWarn = fmt.Sprintf("env %s not set and default %q invalid: %v", rp.HomeEnv, rp.HomeDefault, err)
				}
			} else {
				p.HomeWarn = fmt.Sprintf("env %s not set and no homeDefault configured", rp.HomeEnv)
			}
		} else {
			p.HomeWarn = fmt.Sprintf("placeholder %s has no homeEnv mapping", ph)
		}
	}
	if exp, err := safeio.ExpandPath(homeTpl); err == nil && !strings.ContainsAny(exp, "<>") {
		p.Home = exp
		p.HomeKnown = true
	} else {
		p.HomeKnown = false
		if err != nil {
			p.HomeWarn = strings.TrimSpace(p.HomeWarn + "; expand home: " + err.Error())
		}
	}
}

// expandPath 展开 roots 占位符 / home 占位符 / ~ 与 %VAR%; 未知占位符标记未解析。
func (c *Config) expandPath(tpl string, p *Platform) (string, bool) {
	s := tpl
	for _, m := range placeholderRe.FindAllStringSubmatch(tpl, -1) {
		ph := "<" + m[1] + ">"
		if v, ok := c.Roots[m[1]]; ok {
			s = strings.ReplaceAll(s, ph, v)
			continue
		}
		if p.homePlaceholder == m[1] && p.HomeKnown {
			s = strings.ReplaceAll(s, ph, p.Home)
			continue
		}
		logx.Warn("platform %s: unresolved placeholder %s in %q", p.ID, ph, tpl)
		return tpl, false
	}
	exp, err := safeio.ExpandPath(s)
	if err != nil {
		logx.Warn("platform %s: expand %q: %v", p.ID, tpl, err)
		return tpl, false
	}
	return exp, true
}

func osSupported(list []string) bool {
	if len(list) == 0 {
		return true
	}
	for _, o := range list {
		if o == runtime.GOOS {
			return true
		}
	}
	return false
}

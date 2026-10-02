// Package platform 解析 platforms.json (schema fenjue-platforms-v1) 并按 runtime.GOOS 展开路径。
// 占位符: <skills>/<memory> 展开 roots; <HERMES_HOME> 等按 homeEnv/homeDefault 解析; ~ / %VAR% 由 safeio 展开。
package platform

import (
	"encoding/json"
	"fmt"
	"os"
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

type rawFile struct {
	Schema    string            `json:"schema"`
	Version   string            `json:"version"`
	Roots     map[string]string `json:"roots"`
	Site      *rawSite          `json:"site"`
	Platforms []rawPlatform     `json:"platforms"`
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
	raw        *rawFile
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
	cfg.resolve(nil)
	return cfg, nil
}

// WithRoots 用 override 覆盖 roots 后重新展开全部路径 (enable body {roots?})。
func (c *Config) WithRoots(override map[string]string) *Config {
	c2 := &Config{Schema: c.Schema, Version: c.Version, SourcePath: c.SourcePath, raw: c.raw, Site: c.Site}
	c2.resolve(override)
	return c2
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

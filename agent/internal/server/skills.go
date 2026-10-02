package server

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// skillInfo GET /api/skills 的单条目 (来自 SKILL.md frontmatter)。
type skillInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var (
	fmNameRe = regexp.MustCompile(`(?m)^name:\s*(.+?)\s*$`)
	fmDescRe = regexp.MustCompile(`(?m)^description:\s*(.+?)\s*$`)
)

// handleSkills GET /api/skills: 实时扫描技能根, 返回各技能 SKILL.md 的 name/description。
func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed; use GET")
		return
	}
	root := s.cfg.Roots["skills"]
	skills := []skillInfo{}
	note := ""
	entries, err := os.ReadDir(root)
	if err != nil {
		note = "skills root unavailable: " + err.Error()
	} else {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, e.Name(), "SKILL.md"))
			if err != nil {
				continue
			}
			si := skillInfo{ID: e.Name(), Name: e.Name()}
			si.Name, si.Description = parseFrontmatter(string(data))
			if si.Name == "" {
				si.Name = e.Name()
			}
			skills = append(skills, si)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"roots":  s.cfg.Roots,
		"skills": skills,
		"total":  len(skills),
		"note":   note,
	})
}

// parseFrontmatter 从 SKILL.md 的 YAML 头部取 name 与 description (容忍引号)。
func parseFrontmatter(text string) (string, string) {
	cut := text
	if strings.HasPrefix(text, "---") {
		if i := strings.Index(text[3:], "\n---"); i >= 0 {
			cut = text[3 : 3+i]
		}
	}
	name := ""
	desc := ""
	if m := fmNameRe.FindStringSubmatch(cut); m != nil {
		name = trimQuotes(m[1])
	}
	if m := fmDescRe.FindStringSubmatch(cut); m != nil {
		desc = trimQuotes(m[1])
	}
	return name, desc
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

package logx

import (
	"strings"
	"testing"
)

// 审计行的格式是判据载体: 排查命令 `grep "\[AUDIT\]" agent.log` 依赖它,
// 故格式必须有正例, 也必须有反例(证明 sanitize 真的在干活, 而不是空转)。

func TestAuditLineFormat(t *testing.T) {
	got := AuditLine("enable", `C:\Users\x\.fenjue\memory`, true, "1 platform")
	want := `[AUDIT] op=enable target=C:\Users\x\.fenjue\memory ok=true detail=1 platform`
	if got != want {
		t.Fatalf("format drifted:\n got %q\nwant %q", got, want)
	}
}

func TestAuditLineFlattensNewlines(t *testing.T) {
	// 反例腿: 一条审计若能注入换行, 就能在日志里伪装成两条独立记录。
	got := AuditLine("enable", "a\nb", false, "x\r\ny\n[AUDIT] op=fake target=z ok=true detail=spoofed")
	if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
		t.Fatalf("audit line must not contain raw newlines: %q", got)
	}
	// 关键: 值里嵌入的标记必须被转义, 否则排查用的 grep "\[AUDIT\]" 会命中伪造行。
	// 本条是 sanitize 里「方括号转义」那段存在的唯一理由, 删掉它测试就要红。
	if n := strings.Count(got, AuditPrefix); n != 1 {
		t.Fatalf("must contain exactly one unescaped %s marker, got %d: %q", AuditPrefix, n, got)
	}
	if !strings.Contains(got, `\[AUDIT\]`) {
		t.Fatalf("embedded marker must survive in escaped form, got: %q", got)
	}
}

func TestAuditLineStripsControlChars(t *testing.T) {
	got := AuditLine("op\x00x", "ta\x07rget", true, "d\x1be")
	if strings.ContainsAny(got, "\x00\x07\x1b") {
		t.Fatalf("control chars must be stripped: %q", got)
	}
}

func TestAuditLineEmptyValuesBecomeDash(t *testing.T) {
	got := AuditLine("", "", false, "   ")
	for _, want := range []string{"op=-", "target=-", "detail=-"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}
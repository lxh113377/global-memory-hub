package logx

import (
	"fmt"
	"strings"
)

// AuditPrefix 是审计行的固定前缀。固定前缀是刻意的: 排查时只需
// `grep "\[AUDIT\]" agent.log`, 不必知道字段顺序。
const AuditPrefix = "[AUDIT]"

// Audit 落一条结构化审计行。
//
// 为什么单独一 fun 而不是复用 Info: 审计行要能被机器 grep, 格式不能随 Info 的措辞变化;
// 而且 Info 记的是「发生了什么」, Audit 记的是「谁被允许改了什么、成没成」。
//
// 级别取 INFO(只进 agent.log, 不上 stderr)。审计是**记录**不是**告警**——
// 真有失败时调用方已经收到非 0 返回码与 WARN 日志, 审计再往上冒只会淹没 stderr。
//
// 字段顺序固定为 op / target / ok / detail, detail 放最后且允许含空格。
// 值里的换行会被压平, 否则一条审计可以伪装成多条。
func Audit(op, target string, ok bool, detail string) {
	logf(LevelInfo, "%s", AuditLine(op, target, ok, detail))
}

// AuditLine 组装一条审计行。导出它是为了让测试能断言格式, 而不必去抓日志文件。
func AuditLine(op, target string, ok bool, detail string) string {
	return fmt.Sprintf("%s op=%s target=%s ok=%t detail=%s",
		AuditPrefix, sanitize(op), sanitize(target), ok, sanitize(detail))
}

// sanitize 让一个值无法伪装成审计结构:
//  1. 换行/制表压成空格 —— 否则一条审计可以伪装成多条独立记录;
//  2. 其余控制字符丢弃;
//  3. 方括号转义 —— 否则值里嵌入 "[AUDIT] op=fake …" 就能伪造出第二条审计行,
//     而排查命令恰好就是 grep "\[AUDIT\]"。
// 空值统一成 "-", 避免 grep 时看到 target= 后面空着、像漏了字段。
func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\r' || r == '\n' || r == '\t':
			b.WriteRune(' ')
		case r == '[':
			b.WriteString(`\[`)
		case r == ']':
			b.WriteString(`\]`)
		case r < 0x20 || r == 0x7f:
			// 其余不可见控制字符直接丢弃。
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "-"
	}
	return out
}
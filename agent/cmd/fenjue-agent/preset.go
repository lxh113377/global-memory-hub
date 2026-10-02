package main

import (
	"encoding/json"
	"fmt"
	"os"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/state"
)

// cmdPreset 命名预设的批量启停 (CLI 面)。
//
// 与 HTTP 面共用 state.ApplyPreset, 因此两边的行为、备份号与错误文案完全一致;
// 预设是"一次动多个端"的动作, 两个入口给出的答案不必须相同就会让人怀疑哪个是真的。
func cmdPreset(args []string) {
	set, cf := newFlags("preset")
	action := set.String("action", state.PresetActionEnable, "enable | disable")
	dryRun := set.Bool("dry-run", false, "resolve members and report, touch nothing")
	// 位置参数(预设名)与 flag 混排都要支持: `preset daily --action enable` 与 `preset --action enable daily`。
	rest, positional := splitPositional(args, map[string]bool{"platforms": true, "origin": true, "port": true, "action": true})
	set.Parse(rest)
	if len(positional) == 0 {
		fmt.Fprintln(os.Stderr, "error: usage: fenjue-agent preset <preset-name> [--action enable|disable] [--dry-run]")
		os.Exit(2)
	}
	name := positional[0]
	initLogging()
	defer logx.Close()
	cfg := mustConfig(*cf.platforms)

	res, err := state.ApplyPreset(cfg, name, *action, *dryRun)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
}
// uninstall.go - G11: 彻底卸载与更新检查。
// 卸载语义: 先逐端 disable(可还原), 再把 ~/.fenjue 整体改名搬走 —— 不做磁盘直删, 移回原名即完全还原。
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/safeio"
	"fenjue-agent/internal/state"
)

func cmdUninstall(args []string) {
	set, cf := newFlags("uninstall")
	yes := set.Bool("yes", false, "also move ~/.fenjue aside (to ~/.fenjue.uninstalled-<ts>)")
	hard := set.Bool("hard", true, "disable mode: hard=true removes inject blocks, soft keeps disabled shells")
	set.Parse(args)
	initLogging()
	defer logx.Close()
	cfg := mustConfig(*cf.platforms)

	for _, p := range cfg.Platforms {
		res, err := state.Disable(cfg, p.ID, !*hard)
		if err != nil {
			fmt.Printf("disable %s: error: %v\n", p.ID, err)
			continue
		}
		fmt.Printf("disabled %s: %d changes\n", p.ID, len(res.Changes))
	}

	home := safeio.FenjueHome()
	if !*yes {
		fmt.Printf("app data kept at %s (run with --yes to move it aside)\n", home)
		return
	}
	dst := home + ".uninstalled-" + time.Now().Format("20060102-150405")
	if err := os.Rename(home, dst); err != nil {
		fmt.Fprintf(os.Stderr, "error: move %s: %v (is the agent still running? stop it first)\n", home, err)
		os.Exit(1)
	}
	fmt.Printf("app data moved to %s\n", dst)
	fmt.Println("uninstall done; delete that folder to finish, or move it back to restore.")
}

func cmdVersion(args []string) {
	if len(args) > 0 && args[0] == "--check" {
		checkUpdate()
		return
	}
	fmt.Println(version)
}

// checkUpdate 查询 GitHub latest release; 网络失败静默降级为 latest=unknown, 不报错退出。
func checkUpdate() {
	out := struct {
		Current         string `json:"current"`
		Latest          string `json:"latest"`
		UpdateAvailable bool   `json:"update_available"`
	}{Current: version, Latest: "unknown"}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/lxh113377/global-memory-hub/releases/latest", nil)
	if err == nil {
		req.Header.Set("User-Agent", "fenjue-agent/"+version)
		if resp, err := client.Do(req); err == nil {
			defer resp.Body.Close()
			var payload struct {
				TagName string `json:"tag_name"`
			}
			if json.NewDecoder(resp.Body).Decode(&payload) == nil && payload.TagName != "" {
				out.Latest = payload.TagName
				out.UpdateAvailable = payload.TagName != "v"+version
			}
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.Encode(out)
}

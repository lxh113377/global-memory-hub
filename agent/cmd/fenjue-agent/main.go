// fenjue-agent: global-memory-hub 的本地伴随程序。
// 子命令: serve(默认) / verify / enable <id> / disable <id> / version。
// 全部只监听 127.0.0.1; 路径只做 ~ / %USERPROFILE% / $HOME 展开, 不含个人绝对路径。
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fenjue-agent/internal/logx"
	"fenjue-agent/internal/platform"
	"fenjue-agent/internal/safeio"
	"fenjue-agent/internal/server"
	"fenjue-agent/internal/state"
)

const version = server.Version

type originList []string

func (o *originList) String() string     { return strings.Join(*o, ",") }
func (o *originList) Set(v string) error { *o = append(*o, v); return nil }

type cliFlags struct {
	port      *int
	platforms *string
	origins   originList
	soft      *bool
}

func main() {
	sub := "serve"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "serve":
		cmdServe(args)
	case "verify":
		cmdVerify(args)
	case "enable":
		cmdToggle(args, true)
	case "disable":
		cmdToggle(args, false)
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", sub)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `fenjue-agent - global-memory-hub local companion

usage:
  fenjue-agent serve    [--port 7799] [--platforms <path>] [--origin <url>]...
  fenjue-agent verify   [--platforms <path>]
  fenjue-agent enable <id>   [--platforms <path>]
  fenjue-agent disable <id>  [--soft=true] [--platforms <path>]

serve hosts the local console on 127.0.0.1 only and prints a handshake link.
`)
}

func newFlags(sub string) (*flag.FlagSet, *cliFlags) {
	cf := &cliFlags{}
	set := flag.NewFlagSet("fenjue-agent "+sub, flag.ExitOnError)
	cf.port = set.Int("port", 7799, "HTTP port (listens on 127.0.0.1 only)")
	cf.platforms = set.String("platforms", "", "path to platforms.json (default: auto-discover repo root)")
	set.Var(&cf.origins, "origin", "extra allowed Origin, repeatable")
	cf.soft = set.Bool("soft", true, "disable: soft=true keeps marker block as disabled shell")
	return set, cf
}

func resolvePlatformsPath(p string) (string, error) {
	if p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("platforms file not found: %s", p)
		}
		return p, nil
	}
	cands := []string{"platforms.json", filepath.Join("..", "platforms.json"), filepath.Join("..", "..", "platforms.json")}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			abs, err := filepath.Abs(c)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
	}
	return "", errors.New("platforms.json not found; pass --platforms <path> (default expects repo root layout)")
}

func mustConfig(path string) *platform.Config {
	p, err := resolvePlatformsPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	cfg, err := platform.Load(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	for _, pl := range cfg.Platforms {
		if pl.HomeWarn != "" {
			logx.Warn("platform %s: %s", pl.ID, pl.HomeWarn)
		}
	}
	return cfg
}

func initLogging() {
	logx.Init(filepath.Join(safeio.FenjueHome(), "logs", "agent.log"))
}

func cmdServe(args []string) {
	set, cf := newFlags("serve")
	set.Parse(args)
	initLogging()
	defer logx.Close()
	cfg := mustConfig(*cf.platforms)

	token, err := currentToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	dist, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: embed dist:", err)
		os.Exit(1)
	}
	srv := server.New(cfg, token, *cf.port, cf.origins, dist)

	addr := fmt.Sprintf("127.0.0.1:%d", *cf.port)
	fmt.Printf("fenjue-agent %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("platforms: %s\n", cfg.SourcePath)
	fmt.Printf("token file: %s\n", safeio.TokenPath())
	fmt.Printf("Handshake: http://127.0.0.1:%d/console#token=%s\n", *cf.port, token)
	fmt.Printf("listening on %s\n", addr)
	logx.Info("serve start on %s (platforms=%s)", addr, cfg.SourcePath)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "error: listen %s: %v (port in use? try --port)\n", addr, err)
		logx.Error("listen %s: %v", addr, err)
		os.Exit(1)
	}
}

// currentToken 每次启动生成新的 32 字节随机 hex token, 写 ~/.fenjue/token。
func currentToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(b)
	if err := safeio.WriteFile(safeio.TokenPath(), []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write token file: %w", err)
	}
	return token, nil
}

// splitPositional 把 args 拆成 (可解析的 flag 串, 位置参数)。valueFlags 是带值的 flag 名(不含 -)。
func splitPositional(args []string, valueFlags map[string]bool) ([]string, []string) {
	var rest, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			rest = append(rest, a)
			name := strings.TrimLeft(a, "-")
			if j := strings.IndexByte(name, '='); j >= 0 {
				continue // -x=v 形式不带独立值
			}
			if valueFlags[name] && i+1 < len(args) {
				i++
				rest = append(rest, args[i])
			}
		} else {
			positional = append(positional, a)
		}
	}
	return rest, positional
}

func cmdVerify(args []string) {
	set, cf := newFlags("verify")
	set.Parse(args)
	initLogging()
	defer logx.Close()
	cfg := mustConfig(*cf.platforms)
	snap := state.BuildSnapshot(cfg)
	for _, p := range snap.Platforms {
		fmt.Printf("platform %s (%s, %s): %s\n", p.ID, p.Label, p.Support, p.Status)
		for _, m := range p.Mounts {
			fmt.Printf("  %-9s %s -> %s  %s\n", m.Kind, m.From, m.To, m.Status)
		}
		for _, inj := range p.Inject {
			fmt.Printf("  inject    %s  %s\n", inj.Path, inj.Status)
		}
	}
	rep := state.Verify(cfg)
	fmt.Printf("\nverify: total=%d ok=%d broken=%d mismatch=%d missing=%d\n",
		rep.Total, rep.OK, len(rep.Broken), len(rep.Mismatch), len(rep.Missing))
}

func cmdToggle(args []string, enable bool) {
	sub := "enable"
	if !enable {
		sub = "disable"
	}
	set, cf := newFlags(sub)
	// 手动分离位置参数(id)与 flag, 兼容 `enable tl --platforms x` 与 `enable --platforms x tl`。
	rest, positional := splitPositional(args, map[string]bool{"platforms": true, "origin": true, "port": true})
	set.Parse(rest)
	initLogging()
	defer logx.Close()
	cfg := mustConfig(*cf.platforms)
	id := ""
	if len(positional) > 0 {
		id = positional[0]
	}
	if id == "" {
		fmt.Fprintf(os.Stderr, "error: usage: fenjue-agent %s <platform-id>\n", sub)
		os.Exit(2)
	}
	var res *state.OpResult
	var err error
	if enable {
		res, err = state.Enable(cfg, id)
	} else {
		res, err = state.Disable(cfg, id, *cf.soft)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	enc.Encode(res)
	fmt.Print(buf.String())
}

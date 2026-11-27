package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davidparkercodes/outworld/internal/app"
	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/internal/gateway"
	"github.com/davidparkercodes/outworld/pkg/outworld"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var version = "0.3.0"

const usage = `outworld: privacy gateway for OpenAI-compatible LLM APIs

Usage:
  outworld serve   [--config outworld.yaml]
  outworld mask    [--config outworld.yaml] [--scope name]   < in > out
  outworld unmask  [--config outworld.yaml] [--scope name]   < in > out
  outworld dry-run [--config outworld.yaml] [--scope name] [--api openai|anthropic] [file]
  outworld keygen
  outworld health  [--config outworld.yaml]
  outworld version

The config path defaults to $OUTWORLD_CONFIG, then ./outworld.yaml.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "mask":
		err = maskCmd(os.Args[2:], os.Stdin, os.Stdout)
	case "unmask":
		err = unmaskCmd(os.Args[2:], os.Stdin, os.Stdout)
	case "dry-run":
		err = dryRunCmd(os.Args[2:], os.Stdin, os.Stdout)
	case "health":
		err = healthCmd(os.Args[2:])
	case "keygen":
		var k string
		if k, err = vault.GenerateKey(); err == nil {
			fmt.Println(k)
		}
	case "version", "--version", "-v":
		fmt.Println("outworld", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, errorLine(err))
		if errors.Is(err, outworld.ErrLeak) {
			os.Exit(3)
		}
		os.Exit(1)
	}
}

type common struct {
	config string
	scope  string
}

func parseFlags(name string, args []string, withScope bool) (*common, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	c := &common{}
	fs.StringVar(&c.config, "config", "", "path to outworld.yaml")
	if withScope {
		fs.StringVar(&c.scope, "scope", "", "date-shift scope (defaults to policy.date_shift_scope)")
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	c.config = app.ConfigPath(c.config)
	return c, nil
}

func loadEngine(c *common) (*config.Config, *outworld.Engine, error) {
	cfg, err := config.Load(c.config)
	if err != nil {
		return nil, nil, err
	}
	eng, err := app.BuildEngine(cfg)
	if err != nil {
		return nil, nil, err
	}
	if c.scope == "" {
		c.scope = cfg.Policy.DateShiftScope
	}
	return cfg, eng, nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if strings.EqualFold(cfg.Log.Format, "text") {
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}

func serve(args []string) error {
	c, err := parseFlags("serve", args, false)
	if err != nil {
		return err
	}
	cfg, eng, err := loadEngine(c)
	if err != nil {
		return err
	}
	defer eng.Vault().Close()
	log := newLogger(cfg)
	app.WarnDroppedForms(log, eng.Vault().DroppedForms())
	srv, err := gateway.New(cfg, eng, gateway.Options{Logger: log})
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	return srv.ListenAndServe(ctx)
}

func healthCmd(args []string) error {
	c, err := parseFlags("health", args, false)
	if err != nil {
		return err
	}
	cfg, err := config.Load(c.config)
	if err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health: status %d", resp.StatusCode)
	}
	fmt.Println("ok")
	return nil
}

func maskCmd(args []string, in io.Reader, out io.Writer) error {
	c, err := parseFlags("mask", args, true)
	if err != nil {
		return err
	}
	_, eng, err := loadEngine(c)
	if err != nil {
		return err
	}
	defer eng.Vault().Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	masked, err := maskText(context.Background(), eng.NewSession(c.scope), string(data))
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, masked)
	return err
}

func maskText(ctx context.Context, sess *outworld.Session, text string) (string, error) {
	sess.SetPriming(true)
	_, err := sess.Mask(ctx, text)
	sess.SetPriming(false)
	if err != nil {
		return "", err
	}
	masked, err := sess.Mask(ctx, text)
	if err != nil {
		return "", err
	}
	if lerr := gateway.SessionLeak(sess); lerr != nil {
		return "", lerr
	}
	return masked, nil
}

func errorLine(err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, "outworld: ") {
		return msg
	}
	return "outworld: " + msg
}

func unmaskCmd(args []string, in io.Reader, out io.Writer) error {
	c, err := parseFlags("unmask", args, true)
	if err != nil {
		return err
	}
	_, eng, err := loadEngine(c)
	if err != nil {
		return err
	}
	defer eng.Vault().Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, eng.NewSession(c.scope).Rehydrate(string(data)))
	return err
}

func dryRunCmd(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	c := &common{}
	var api string
	fs.StringVar(&c.config, "config", "", "path to outworld.yaml")
	fs.StringVar(&c.scope, "scope", "", "date-shift scope (defaults to policy.date_shift_scope)")
	fs.StringVar(&api, "api", "", "request format: openai or anthropic (default: whichever has a masked upstream)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c.config = app.ConfigPath(c.config)
	if fs.NArg() > 1 {
		return errors.New("dry-run takes at most one file")
	}
	if fs.NArg() == 1 && fs.Arg(0) != "-" {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	cfg, eng, err := loadEngine(c)
	if err != nil {
		return err
	}
	defer eng.Vault().Close()
	switch api {
	case "":
		api = config.APIOpenAI
		if cfg.Policy.MaskedUpstream == "" && cfg.Policy.Anthropic.MaskedUpstream != "" {
			api = config.APIAnthropic
		}
	case config.APIOpenAI, config.APIAnthropic:
	default:
		return fmt.Errorf("unknown --api %q", api)
	}
	srv, err := gateway.New(cfg, eng, gateway.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		return err
	}
	rep, err := srv.DryRun(context.Background(), api, dryRunBody(data), nil, c.scope)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		return err
	}
	if rep.Blocked == "outworld_leak_blocked" {
		return &outworld.LeakError{Violations: rep.Findings}
	}
	return nil
}

func dryRunBody(data []byte) []byte {
	var obj map[string]any
	if json.Unmarshal(data, &obj) == nil {
		for _, k := range []string{"messages", "prompt", "input", "system"} {
			if _, ok := obj[k]; ok {
				return data
			}
		}
	}
	body, _ := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "user", "content": string(data)}}})
	return body
}

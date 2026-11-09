package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	Listen      string
	VaultPath   string
	UpstreamURL string
	Dictionary  string
	NERURL      string
	DateShift   bool
	Entities    map[string]string
}

func yamlQuote(s string) string { return strconv.Quote(s) }

func (c Config) YAML() string {
	var b strings.Builder
	fmt.Fprintf(&b, "listen: %s\n", c.Listen)
	b.WriteString("vault:\n")
	fmt.Fprintf(&b, "  path: %s\n", yamlQuote(c.VaultPath))
	b.WriteString("  key_env: OUTWORLD_VAULT_KEY\n")
	b.WriteString("upstreams:\n")
	fmt.Fprintf(&b, "  cloud: {base_url: %s, api_key_env: CLOUD_API_KEY}\n", yamlQuote(c.UpstreamURL))
	b.WriteString("policy:\n")
	b.WriteString("  default_route: masked\n")
	b.WriteString("  masked_upstream: cloud\n")
	fmt.Fprintf(&b, "  date_shift: %t\n", c.DateShift)
	if c.Dictionary != "" {
		fmt.Fprintf(&b, "  dictionary: %s\n", yamlQuote(c.Dictionary))
	}
	keys := make([]string, 0, len(c.Entities))
	for k := range c.Entities {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+": "+c.Entities[k])
	}
	fmt.Fprintf(&b, "  entities: {%s}\n", strings.Join(parts, ", "))
	if c.NERURL != "" {
		fmt.Fprintf(&b, "ner: {url: %s, timeout: 120s}\n", yamlQuote(c.NERURL))
	}
	return b.String()
}

func FreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func RandomKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func Build(ctx context.Context, moduleDir, outDir string) (string, error) {
	if _, err := os.Stat(filepath.Join(moduleDir, "cmd", "outworld")); err != nil {
		return "", fmt.Errorf("gateway source not found at %s/cmd/outworld: %w", moduleDir, err)
	}
	bin := filepath.Join(outDir, "outworld")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/outworld")
	cmd.Dir = moduleDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build ./cmd/outworld failed: %w\n%s", err, out)
	}
	return bin, nil
}

type Process struct {
	URL  string
	cmd  *exec.Cmd
	Log  *os.File
	done chan error
}

func Start(ctx context.Context, bin, configPath, logPath string, env []string, listen string) (*Process, error) {
	logf, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin, "serve", "--config", configPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		logf.Close()
		return nil, err
	}
	p := &Process{URL: "http://" + listen, cmd: cmd, Log: logf, done: make(chan error, 1)}
	go func() { p.done <- cmd.Wait() }()
	if err := p.waitHealthy(ctx, 60*time.Second); err != nil {
		p.Stop()
		tail, _ := os.ReadFile(logPath)
		return nil, fmt.Errorf("%w\ngateway log:\n%s", err, lastLines(string(tail), 30))
	}
	return p, nil
}

func (p *Process) waitHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	c := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case err := <-p.done:
			p.done <- err
			return fmt.Errorf("gateway exited before becoming healthy: %v", err)
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := c.Get(p.URL + "/healthz")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("gateway not healthy at %s/healthz after %s", p.URL, timeout)
}

func (p *Process) Stop() {
	if p.cmd.Process != nil {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-p.done:
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
			<-p.done
		}
	}
	p.Log.Close()
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

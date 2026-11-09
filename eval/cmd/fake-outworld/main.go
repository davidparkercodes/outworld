package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/internal/shim"
	"github.com/davidparkercodes/outworld/eval/score"
)

var (
	reListen = regexp.MustCompile(`(?m)^listen:\s*(\S+)`)
	reBase   = regexp.MustCompile(`cloud:\s*\{base_url:\s*("[^"]*"|[^,}\s]+)`)
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "usage: fake-outworld serve --config <file>")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", "", "config file")
	_ = fs.Parse(os.Args[2:])
	raw, err := os.ReadFile(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lm, bm := reListen.FindSubmatch(raw), reBase.FindSubmatch(raw)
	if lm == nil || bm == nil {
		fmt.Fprintln(os.Stderr, "config missing listen or upstreams.cloud.base_url")
		os.Exit(1)
	}
	base := string(bm[1])
	if u, err := strconv.Unquote(base); err == nil {
		base = u
	}
	s := shim.Passthrough(base)
	if os.Getenv("FAKE_OUTWORLD_MODE") == "oracle" {
		c, err := corpus.Load(os.Getenv("FAKE_OUTWORLD_CORPUS"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		s = shim.Oracle(base, c.Items, score.DefaultPolicy())
	}
	fmt.Fprintf(os.Stderr, "fake-outworld listening on %s -> %s\n", lm[1], base)
	if err := http.ListenAndServe(string(lm[1]), s.Handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

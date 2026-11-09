package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidparkercodes/outworld/eval/corpus"
)

func main() {
	out := flag.String("out", "corpus", "output directory for corpus.json and people.csv")
	seed := flag.Uint64("seed", corpus.DefaultSeed, "generator seed")
	flag.Parse()
	c, dict := corpus.Generate(*seed)
	if err := c.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid corpus:", err)
		os.Exit(1)
	}
	data, err := c.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, "corpus.json"), data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, "people.csv"), corpus.MarshalDictionary(dict), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spans := 0
	for _, it := range c.Items {
		spans += len(it.Spans)
	}
	fmt.Printf("wrote %d items, %d spans, %d dictionary entries to %s\n", len(c.Items), spans, len(dict), *out)
}

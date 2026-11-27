package outworld

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func TestTechNameMaskScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	ctx := context.Background()
	size := scaletest.Size(64 << 10)
	for _, c := range []struct{ name, known, line string }{
		{"assistant_name", "Claude Shannon wrote it.", "Hi Claude, the API is down. Claude said so. "},
		{"homonym_ruby", "Ruby Smith called about the invoice.", "Ruby wants a refund. Please ask Ruby to call. "},
		{"homonym_adam", "Adam Smith called about the invoice.", "Adam said hi. Thanks Adam, see you. "},
		{"homonym_phrase", "Ruby Smith called about the invoice.", "Ruby code was wrong after the npm install of ruby_gem. "},
		{"homonym_dot_slash", "Ruby Smith called about the invoice.", "I paid Ruby.Then Ruby/Adam/Julia left. "},
		{"homonym_unbroken", "Ruby Smith called about the invoice.", "Ruby code "},
		{"homonym_fences", "Ruby Smith called about the invoice.", "```\nx = Ruby.run(1)\n```\nRuby called. `Ruby` it`s ok, Ruby.\n"},
		{"homonym_glued", "Ruby Smith called about the invoice.", "rubysmith smithruby ruby smith RubySmith rubysmithson called. "},
		{"homonym_pair_paragraph", "Ruby Kelly called about the invoice.", "ruby kelly ruby.kelly99 kellyruby, ruby\tkelly and some other words here "},
		{"homonym_ticks", "Ruby Smith called about the invoice.", "Ruby ` called ``Ruby`` and ~~~ Ruby. "},
	} {
		scaletest.Check(t, "mask "+c.name, size, func(n int) func() {
			e := newEngine(t, nil)
			if _, err := e.NewSession("seed").Mask(ctx, c.known); err != nil {
				t.Fatal(err)
			}
			text := strings.Repeat(c.line, n/len(c.line)+1)[:n]
			scope := uniqueScope("p")
			return func() {
				if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestHyphenChainMaskScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	ctx := context.Background()
	size := scaletest.Size(96 << 10)
	for _, c := range []struct{ name, seed, unit string }{
		{"chain_anna_fresh", "", "Anna-Kelly-"},
		{"chain_ruby_fresh", "", "Ruby-Kelly-"},
		{"chain_anna_vault", "Anna Kelly called about the invoice.", "Anna-Kelly-"},
		{"chain_ruby_vault", "Ruby Kelly called about the invoice.", "Ruby-Kelly-"},
	} {
		scaletest.Check(t, "mask "+c.name, size, func(n int) func() {
			e := newEngine(t, nil)
			if c.seed != "" {
				if _, err := e.NewSession("seed").Mask(ctx, c.seed); err != nil {
					t.Fatal(err)
				}
			}
			text := strings.Repeat(c.unit, n/len(c.unit)+1)[:n]
			scope := uniqueScope("c")
			return func() {
				if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

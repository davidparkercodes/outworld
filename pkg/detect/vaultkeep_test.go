package detect

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func vaultKnownForTest(w string) bool {
	switch w {
	case "Ruby", "Kelly", "Julia":
		return true
	}
	return false
}

func TestVaultNameKeepCaseConvention(t *testing.T) {
	cases := []struct {
		text, name string
		keep       bool
	}{
		{"Run `Ruby --version` first.", "Ruby", true},
		{"```\nRuby.run(1)\n```", "Ruby", true},
		{"Call Ruby.run(1) now.", "Ruby", false},
		{"Use Ruby.new.call[0] here.", "Ruby", false},
		{"Call Config.Ruby now.", "Ruby", false},
		{"Module.Ruby is loaded.", "Ruby", false},
		{"System.Crystal fails.", "Crystal", false},
		{"Use Ruby::Version here.", "Ruby", false},
		{"Open lib/Ruby/core.rb please.", "Ruby", false},
		{"Look in /usr/lib/Ruby/core for it.", "Ruby", false},
		{"Load ./Ruby/setup now.", "Ruby", false},
		{"See https://example.com/Ruby/docs today.", "Ruby", false},
		{"See ruby-lang for the docs.", "ruby", true},
		{"Load ruby_core now.", "ruby", true},
		{"Edit my_ruby_script.rb now.", "ruby", true},
		{"Open ruby.yml please.", "ruby", true},
		{"met ruby.she said no", "ruby", false},
		{"Open ruby.kelly.rb now.", "ruby", false},
		{"ruby::kelly called.", "ruby", false},
		{"ruby/kelly called.", "ruby", false},
		{"Open ruby.lock now.", "ruby", false},
		{"Open ruby.go now.", "ruby", false},
		{"Open ruby.a.b now.", "ruby", true},
		{"In the 90` Ruby worked there, in the 2000` she left.", "Ruby", false},
		{"`Kelly, Ruby`", "Ruby", false},
		{"`Ruby`'s car broke.", "Ruby", false},
		{"`Ruby` 3.2 is out.", "Ruby", true},
		{"See the Ruby-lang site.", "Ruby", false},
		{"Load the Ruby_core module.", "Ruby", false},
		{"Read Ruby.Version first.", "Ruby", false},
		{"Ruby.Version of events is wrong.", "Ruby", false},
		{"Ask about Ruby.Version 2 is due.", "Ruby", false},
		{"Thanks\nRuby.Version helped.", "Ruby", false},
		{"Ruby.she said no.", "Ruby", false},
		{"Ruby.later we will talk.", "Ruby", false},
		{"Waiting on Ruby.not sure when.", "Ruby", false},
		{"Ruby.so tired today.", "Ruby", false},
		{"Ruby.and Julia came.", "Ruby", false},
		{"Ruby.mom called.", "Ruby", false},
		{"Ruby.pls call me.", "Ruby", false},
		{"Ruby.ok fine.", "Ruby", false},
		{"I texted Ruby.go home now.", "Ruby", false},
		{"Ruby.txt me later.", "Ruby", false},
		{"Ruby.lock the door.", "Ruby", false},
		{"Ruby.md at the clinic called.", "Ruby", false},
		{"Ruby.c is my sister.", "Ruby", false},
		{"I met Ruby.txt me back.", "Ruby", false},
		{"Ruby.foo.bar is her blog.", "Ruby", false},
		{"Ruby-test went well for her.", "Ruby", false},
		{"The Ruby-dev team won the game.", "Ruby", false},
		{"Ruby-api called me.", "Ruby", false},
		{"Ruby-core is my sister.", "Ruby", false},
		{"Ruby_core called me today.", "Ruby", false},
		{"The Ruby-setup crew won.", "Ruby", false},
		{"Ruby-script for the play", "Ruby", false},
		{"Ruby:: called me.", "Ruby", false},
		{"Ruby::called me.", "Ruby", false},
		{"Ruby/mom_and_dad will drive.", "Ruby", false},
		{"Ruby/report.pdf is from her.", "Ruby", false},
		{"Ruby/Adam/Julia to cover Friday.", "Adam", false},
		{"Ruby/me to cover the shift.", "Ruby", false},
		{"Ruby/2pm pickup.", "Ruby", false},
		{"mom/Ruby to cover Friday.", "Ruby", false},
		{"Ruby version of events on 9/15 differs.", "Ruby", false},
		{"Ruby script for his/her part is done.", "Ruby", false},
		{"Ruby code is 4411 -- she changed it.", "Ruby", false},
		{"Ruby=4411 for the gate.", "Ruby", false},
		{"Ruby->school->home.", "Ruby", false},
		{"Email Ruby@corp.com today.", "Ruby", false},
		{"I paid Ruby.Then I left.", "Ruby", false},
		{"Ruby.K wrote back.", "Ruby", false},
		{"K_Ruby called.", "Ruby", false},
		{"Kelly.Ruby called.", "Ruby", false},
		{"Anne.Julia called.", "Julia", false},
		{"My ex-Ruby called.", "Ruby", false},
		{"Use Ruby-3 builds.", "Ruby", false},
		{"Notes from Julia-2 review.", "Julia", false},
		{"I can`t reach Ruby today.", "Ruby", false},
		{"She`s with Ruby now.", "Ruby", false},
		{"A lone ` then Ruby called.", "Ruby", false},
		{"Open `code\n\nRuby called` later.", "Ruby", false},
		{"Use ``Ruby`` here.", "Ruby", true},
		{"Mixed ``Ruby` runs.", "Ruby", false},
		{"```\nprint(1)\nRuby called.", "Ruby", false},
		{"~~~\nRuby.run(1)\n~~~", "Ruby", true},
		{"````\nx\n```\nRuby.run(1)\n````", "Ruby", true},
		{"~~~\nx\n```\nRuby called.", "Ruby", false},
		{"    Ruby called me, indented.", "Ruby", false},
		{"Run this:\n    ```\n    Ruby build\n    ```\nok", "Ruby", false},
		{"Run this:\n\t```\n\tRuby build\n\t```\nok", "Ruby", false},
		{"```\nx = 1\n    ```\nRuby build", "Ruby", false},
		{"   ```\nRuby build\n   ```\nok", "Ruby", true},
		{"Set RUBY_VERSION now.", "RUBY", false},
		{"Set `RUBY_VERSION` now.", "RUBY", true},
		{"``Ruby'' said hi and ``Tom'' too.", "Ruby", false},
		{"`Ruby' said hi and `Tom' too.", "Ruby", false},
		{"`Ruby\u2019 said hi and `Tom\u2019 too.", "Ruby", false},
		{"Escaped \\`Ruby\\` said hi.", "Ruby", false},
		{"`Ruby` said hi.", "Ruby", true},
		{"`Ruby build`", "Ruby", true},
		{"a \\`b\\` Ruby said hi \\`c\\`", "Ruby", false},
		{"I took James` car, then Ruby said hi, then Chris` dog ran.", "Ruby", false},
		{"``Ruby``", "Ruby", true},
		{"`` `Ruby` ``", "Ruby", true},
		{"`Ruby's` gem", "Ruby", false},
		{"`Ruby Kelly`", "Ruby", false},
		{"`Ruby and Tom`", "Ruby", false},
		{"```\nx = Ruby\nKelly.save()\n```", "Ruby", false},
		{"```\nx = Ruby\ny = Tom\n```", "Ruby", false},
		{"```\n" + strings.Repeat("x = compute_value(1) + offset\n", 40) + "```\nPlease tell Ruby the fix is in.", "Ruby", false},
	}
	for _, c := range cases {
		i := strings.LastIndex(c.text, c.name)
		if got := VaultNameKeep(c.text, i, i+len(c.name), vaultKnownForTest); got != c.keep {
			t.Errorf("%q (%s): keep=%v, want %v", c.text, c.name, got, c.keep)
		}
	}
	for _, in := range []string{"https://facebook.com/Ruby.Kelly", "See https://example.com/Ruby/Kelly today.", "Ruby.Kelly called.", "ruby.kelly called.", "Config.Ruby.Kelly"} {
		i := strings.Index(strings.ToLower(in), "ruby")
		if VaultNameKeep(in, i, i+4, vaultKnownForTest) {
			t.Errorf("%q: a vault-held name in the same token must mask", in)
		}
	}
}

func TestCodeRegionsAfterLongBlocks(t *testing.T) {
	for _, n := range []int{300, 1000, 10000} {
		body := strings.Repeat("x = compute_value(1) + offset\n", n/30+1)
		text := "```python\n" + body + "Ruby.inside(1)\n```\nRuby called me about the refund. Ruby is upset."
		in := strings.Index(text, "Ruby.inside")
		if !VaultNameKeep(text, in, in+4, nil) {
			t.Errorf("%d: name inside a closed block not kept", n)
		}
		after := strings.LastIndex(text, "Ruby")
		if VaultNameKeep(text, after, after+4, nil) {
			t.Errorf("%d: name after a closed block kept", n)
		}
		open := "```\n" + body + "Ruby called."
		k := strings.LastIndex(open, "Ruby")
		if VaultNameKeep(open, k, k+4, nil) {
			t.Errorf("%d: name after an unclosed fence kept", n)
		}
	}
}

func TestCodeRegionLookupScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	unit := "```\nx = Ruby.run(1)\n```\nRuby called. `Ruby` it`s ok, Ruby ``x` and ~~~\n"
	scaletest.Check(t, "vault keep over fences", scaletest.Size(256<<10), func(n int) func() {
		text := strings.Repeat(unit, n/len(unit)+1)[:n]
		var at []int
		for i := 0; ; {
			k := strings.Index(text[i:], "Ruby")
			if k < 0 {
				break
			}
			at = append(at, i+k)
			i += k + 4
		}
		return func() {
			regionCache.mu.Lock()
			regionCache.r = nil
			regionCache.mu.Unlock()
			for _, i := range at {
				VaultNameKeep(text, i, i+4, vaultKnownForTest)
			}
		}
	})
}

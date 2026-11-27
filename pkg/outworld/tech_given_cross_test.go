package outworld

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
)

func TestTechTermGivenNamesAcrossRequests(t *testing.T) {
	ctx := context.Background()
	for _, n := range techGivenHomonyms {
		same, err := newEngine(t, nil).NewSession("one").Mask(ctx, n+" Smith called about the invoice. "+n+" wants a refund.")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(same, " "+n+" ") || strings.HasPrefix(same, n+" ") {
			t.Errorf("%s: lone mention in the same request left raw in %q", n, same)
		}
	}
	for _, n := range techSurnameHomonyms {
		e := newEngine(t, nil)
		same, _ := e.NewSession("a").Mask(ctx, "Mary "+n+" called about the invoice. Ms "+n+" wants a refund.")
		later, _ := e.NewSession("b").Mask(ctx, "Please ask Mary "+n+" to review the PR.")
		if strings.Contains(same, n) || strings.Contains(later, n) {
			t.Errorf("%s: surname left raw in %q / %q", n, same, later)
		}
	}
	for _, n := range techGivenHomonyms {
		e := newEngine(t, nil)
		first, err := e.NewSession("one").Mask(ctx, n+" Smith called about the invoice.")
		if err != nil {
			t.Fatal(err)
		}
		given := strings.Fields(first)[0]
		if given == n {
			t.Errorf("%s Smith: given name left raw in %q", n, first)
			continue
		}
		for _, later := range []string{n + " said he will call back tomorrow.", "Please ask " + n + " to review the PR.", "Thanks " + n + ", see you Monday.", "Hi " + n + ",\nfollowing up."} {
			s := e.NewSession("two")
			out, err := s.Mask(ctx, later)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out, n) || !strings.Contains(out, given) {
				t.Errorf("%q after %q: got %q, want %q as the same surrogate", later, first, out, given)
			}
			if back := s.Rehydrate(out); back != later {
				t.Errorf("%q: restored %q", later, back)
			}
		}
	}
	for _, in := range []string{"Hi Claude,", "Claude said the API is down"} {
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); out != in {
			t.Errorf("%q: got %q", in, out)
		}
	}
}

var techGivenHomonyms = strings.Fields("Ada Adam Apollo Athena Atlas Aurora Bert Cassandra Crystal Elmo Ernie Haskell Hermes Homer Hugo Jade Jasmine Julia Pascal Ruby Travis Venus Watson")

var techSurnameHomonyms = strings.Fields("Bash Dart Docker Helm Jenkins Jira Kafka Lua Mars Perl Rust Sass Scala Shell Slack Spring Swift Vue")

func TestHomonymsListMatchesTables(t *testing.T) {
	for _, w := range append(append([]string{}, techGivenHomonyms...), techSurnameHomonyms...) {
		if !detect.IsTechTerm(w) || detect.IsAssistantName(w) {
			t.Errorf("%s is not a non-assistant tech term", w)
		}
	}
}

func TestHomonymLaterUseIsUniform(t *testing.T) {
	ctx := context.Background()
	keep := []string{"Run `%s --version` first.", "Open `lib/%s/core.rb` today."}
	mask := []string{"the %s package", "Upgrade %s to the latest version.", "%s config is broken.", "Restart %s after the deploy.", "%s logs show errors.", "Migrate from %s to Go.", "%s compose is up.", "%s topic lag is high.", "%s throws a null pointer.", "%s wants a refund.", "Is %s free on Monday?", "Forward this to %s.", "%s said yes.", "Ask %s about it.", "%s's account is overdue.", "Please call %s tomorrow.", "Thanks %s.", "Ask %s about the server config.", "%s said the deploy failed.", "%s's build is broken.", "Dr. %s restarted the server."}
	check := func(seed, n string) {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, seed); err != nil {
			t.Fatal(err)
		}
		for _, tpl := range keep {
			in := strings.ReplaceAll(tpl, "%s", n)
			if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
				t.Errorf("%q in vault: %q became %q, want unchanged", seed, in, out)
			}
		}
		for _, tpl := range mask {
			if (n == "Docker" && strings.Contains(tpl, "compose")) || (n == "Kafka" && strings.Contains(tpl, "topic")) {
				continue
			}
			in := strings.ReplaceAll(tpl, "%s", n)
			if out, _ := e.NewSession("later").Mask(ctx, in); strings.Contains(out, n) {
				t.Errorf("%q in vault: %q left raw as %q", seed, in, out)
			}
		}
	}
	for _, n := range techGivenHomonyms {
		check(n+" Smith called.", n)
	}
	for _, n := range techSurnameHomonyms {
		check("John "+n+" called.", n)
	}
	for in, n := range map[string]string{"Docker compose is up.": "Docker", "Kafka topic lag is high.": "Kafka", "install the ruby gem": "Ruby", "a crystal vase": "Crystal", "Use `Ruby.new` here.": "Ruby", "Use `Pascal::Core` here.": "Pascal"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, n+" Smith called.")
		if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
			t.Errorf("%s in vault: %q became %q", n, in, out)
		}
	}
	for in, n := range map[string]string{"Write Jasmine tests.": "Jasmine", "the Crystal 1.10 compiler": "Crystal", "Pascal case naming": "Pascal", "Use the Adam optimizer.": "Adam", "Travis CI is red.": "Travis", "Apollo client cache": "Apollo", "Load the Pascal_core module.": "Pascal", "Use Pascal::Core here.": "Pascal", "Open lib/Ruby/core.rb today.": "Ruby", "Ruby 3.3 is installed.": "Ruby", "Write Jasmine tests and run them with --watch.": "Jasmine", "Build the Crystal 1.10 compiler from source.": "Crystal", "Ruby on Rails is fine.": "Ruby", "Pascal case naming in the user_service module": "Pascal", "Use the Adam optimizer in PyTorch.": "Adam", "Hugo site build failed.": "Hugo", "Travis CI is red after the npm upgrade.": "Travis", "Apollo client cache is imported from lib/cache.ts": "Apollo"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, n+" Smith called.")
		if out, _ := e.NewSession("later").Mask(ctx, in); hasWholeWord(out, n) {
			t.Errorf("%s in vault, prose use: %q left raw as %q", n, in, out)
		}
	}
	for _, in := range []string{"Adam wants a refund.", "Ruby 3.3 is installed."} {
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); out != in {
			t.Errorf("fresh vault: %q became %q", in, out)
		}
	}
}

var personSentenceTemplates = []string{
	"%s and Mary will join the call.",
	"%s said the Python build broke.",
	"Meeting with %s about the release.",
	"Lunch with %s and the client tomorrow.",
	"%s has a test tomorrow at school.",
	"I drove %s to the site this morning.",
	"Thanks to %s the build is green.",
	"%s from the build team called.",
	"%s fixed the build yesterday.",
	"%s is our release manager.",
	"%s broke the build again.",
	"Can %s review the code?",
	"My daughter %s has a test on Friday.",
	"Did %s see the error?",
	"%s is sick, so the deploy is postponed.",
	"Please call %s about the server outage.",
	"Interview with %s for the backend role.",
	"Give %s 5 minutes.",
	"Pay %s 20 dollars for lunch.",
	"I saw %s 2 days ago.",
	"%s, the server is down again.",
	"%s and I will deploy tonight.",
	"%s, Mary and I met for lunch.",
	"Send the docs to %s by Friday.",
	"%s was promoted to lead of the API team.",
	"Our new hire %s starts Monday on the platform team.",
	"%s Smith fixed the build.",
	"%s Okafor and the client met today.",
	"I met %s yesterday at the conference.",
	"Please email %s today about the contract.",
	"%s wants a refund.",
	"%s said hi.",
	"Is %s free on Monday?",
	"Forward this to %s.",
	"Thanks %s, see you Monday.",
	"Hi %s,\nThanks for the update.",
	"Dear %s,\nPlease see the attached notes.",
	"Hey %s!\nGreat work on the launch.",
	"Thanks,\n%s",
	"%s's build is broken.",
	"Ask %s about the build.",
	"Tell %s the release shipped.",
	"%s and Mary are coming to dinner.",
	"%s or Mary can pick up the kids.",
	"I learned %s was in the hospital.",
	"We used %s as our babysitter.",
	"%s 2 is the name of her second kid.",
}

var strongTechTemplates = []string{
	"Run `%s --version` first.",
	"Open `lib/%s/core.rb` today.",
}

var flippedTechTemplates = []string{
	"Open lib/%s/core.rb today.",
	"%s 2.0 is installed.",
	"the %s package",
	"Upgrade %s to the latest version.",
	"The %s plugin crashed on startup.",
	"%s release notes are out.",
	"%s config is broken.",
	"Restart %s after the deploy.",
	"%s logs show errors.",
	"%s compose is up.",
	"%s topic lag is high.",
	"%s throws a null pointer.",
	"Install %s on the server.",
	"%s docs say otherwise.",
	"Migrate from %s to Go.",
}

func TestVaultKnownHomonymPersonSentencesMask(t *testing.T) {
	ctx := context.Background()
	for _, n := range append(append([]string{}, techGivenHomonyms...), nameLikeTechHomonyms...) {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, n+" Smith called about the invoice."); err != nil {
			t.Fatal(err)
		}
		for _, tpl := range append(append([]string{}, personSentenceTemplates...), flippedTechTemplates...) {
			in := strings.ReplaceAll(tpl, "%s", n)
			out, _ := e.NewSession("later").Mask(ctx, in)
			for _, w := range strings.FieldsFunc(out, func(r rune) bool { return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') }) {
				if w == n {
					t.Errorf("%s in vault: %q left raw as %q", n, in, out)
					break
				}
			}
		}
		for _, tpl := range strongTechTemplates {
			in := strings.ReplaceAll(tpl, "%s", n)
			if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
				t.Errorf("%s in vault: %q became %q, want unchanged", n, in, out)
			}
		}
	}
	for in, n := range map[string]string{"Write `Jasmine` tests.": "Jasmine", "install the ruby gem": "Ruby"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, n+" Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
			t.Errorf("%s in vault: %q became %q", n, in, out)
		}
	}
}

var nameLikeTechHomonyms = strings.Fields("Mercury Jupiter Saturn Neptune Pluto")

func TestNameLikeTechTermsFollowPersonRules(t *testing.T) {
	ctx := context.Background()
	for _, n := range nameLikeTechHomonyms {
		for _, tpl := range []string{"Hi %s,", "Dear %s,", "Hey %s!", "I met %s yesterday at the conference.", "Please email %s today about the contract.", "%s said the build failed again.", "Thanks,\n%s", "Meeting with %s at 3pm."} {
			in := strings.ReplaceAll(tpl, "%s", n)
			if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); strings.Contains(out, n) {
				t.Errorf("%q left raw as %q", in, out)
			}
		}
	}
	for _, in := range []string{"Hi Kubernetes,", "Dear Grafana,", "Hi Terraform,", "Hi Postgres,", "Hi Redis,", "Hi Nginx,"} {
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); out != in {
			t.Errorf("%q became %q", in, out)
		}
	}
}

func TestAssistantNameAnchoredInRequestMasks(t *testing.T) {
	ctx := context.Background()
	for _, in := range []string{
		"Claude Smith called about the invoice. Claude wants a refund.",
		"Claude Smith called about the invoice. Claude needs a refund.",
		"Claude Dupont called about the invoice. Claude wants a refund.",
		"Claude Shannon called. Claude wants a refund.",
		"Siri Larsen called about the invoice. Siri wants a refund.",
		"Alexa Moreno called about the invoice. Alexa wants a refund.",
		"Gemini Okafor called about the invoice. Gemini wants a refund.",
		"Claude Shannon wrote the paper. Thanks Claude!",
	} {
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		first := strings.Fields(in)[0]
		if strings.Contains(out, first) {
			t.Errorf("%q: %s left raw in %q", in, first, out)
		}
		sur := strings.Fields(out)[0]
		if strings.Count(out, sur) < 2 {
			t.Errorf("%q: the two mentions do not share a surrogate in %q", in, out)
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q: restored %q", in, back)
		}
	}
	for _, in := range []string{"Hi Claude,", "Ask Claude to summarize this."} {
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); out != in {
			t.Errorf("fresh engine: %q became %q", in, out)
		}
	}
}

func TestAnchoredHomonymMasksDespiteTechWords(t *testing.T) {
	ctx := context.Background()
	names := append(append(append([]string{}, techGivenHomonyms...), nameLikeTechHomonyms...), "Claude", "Siri", "Alexa", "Gemini")
	for _, n := range names {
		for _, in := range []string{
			n + " Smith called. " + n + " will deploy the new build tomorrow.",
			n + " Smith maintains our " + n + " 3.3 services.",
		} {
			s := newEngine(t, nil).NewSession("p")
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out, n) {
				t.Errorf("%q: %s left raw in %q", in, n, out)
				continue
			}
			sur := strings.Fields(out)[0]
			if strings.Count(out, sur) < 2 {
				t.Errorf("%q: mentions do not share a surrogate in %q", in, out)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("%q: restored %q", in, back)
			}
		}
		anch := newEngine(t, nil).NewSession("p")
		anch.Mask(ctx, n+" Smith called.")
		in := "Run `" + n + " --version` first."
		if out, _ := anch.Mask(ctx, in); out != in {
			t.Errorf("%q: code span changed in %q", in, out)
		}
		both := n + " Smith called.\n\nRun `" + n + " --version` first."
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, both); !detect.IsAssistantName(n) && strings.Contains(out, "`"+n+" --version`") {
			t.Errorf("%q: a masked mention in the same text did not void the code keep in %q", both, out)
		}
		same := n + " Smith called. Run `" + n + " --version` first."
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, same); !detect.IsAssistantName(n) && strings.Contains(out, "`"+n+" --version`") {
			t.Errorf("%q: same-paragraph pair did not void the code keep in %q", same, out)
		}
	}
}

func TestAmbiguousCompoundNounsStillMaskPeople(t *testing.T) {
	ctx := context.Background()
	for _, in := range []string{"Ruby client meeting moved to Friday.", "Ruby is our release manager.", "Did Ruby see the error?", "Thanks to Ruby the build is green.", "My daughter Ruby has a test on Friday.", "Ruby site visit is tomorrow."} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, "Ruby Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); strings.Contains(out, "Ruby") {
			t.Errorf("%q left raw as %q", in, out)
		}
	}
}

func TestHyphenatedDoubleGivenNamesMask(t *testing.T) {
	ctx := context.Background()
	for _, full := range []string{"Anne-Julia", "Anne-Ruby", "Mary-Jade", "Mary-Crystal", "Mary-Ada", "Mary-Claude"} {
		second := strings.SplitN(full, "-", 2)[1]
		same := full + " Kelly called. " + full + " wants a refund."
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(ctx, same)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, second) {
			t.Errorf("%q: %s left raw in %q", same, second, out)
		}
		if back := s.Rehydrate(out); back != same {
			t.Errorf("%q: restored %q", same, back)
		}
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, full+" Kelly called.")
		later := full + " wants a refund."
		if out, _ := e.NewSession("later").Mask(ctx, later); strings.Contains(out, second) {
			t.Errorf("later %q: %s left raw in %q", later, second, out)
		}
	}
	for in, n := range map[string]string{"See ruby-lang for the docs.": "Ruby", "Open the julia-lang-dev repo.": "Julia", "Open the julia-lang-kit repo.": "Julia"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, n+" Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
			t.Errorf("%s in vault: %q became %q", n, in, out)
		}
	}
	for in, n := range map[string]string{"Run Ruby-build tests.": "Ruby", "Run Ruby-on-rails tests.": "Ruby"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, titleWord(n)+" Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); hasWholeWord(out, n) {
			t.Errorf("%s in vault: %q left raw as %q", n, in, out)
		}
	}
}

func TestTechGivenWithStreetWordSurname(t *testing.T) {
	ctx := context.Background()
	people := []string{
		"Crystal Lane called this morning.",
		"Can you ask Crystal Lane about the invoice?",
		"Jade Park called about the invoice.",
	}
	for _, in := range people {
		out, err := newEngine(t, nil).NewSession("one").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "Crystal") || strings.Contains(out, "Lane") || strings.Contains(out, "Jade") {
			t.Errorf("%q: person left raw in %q", in, out)
		}
	}
	streets := []string{
		"We live on Crystal Lane now.",
		"Turn left onto Crystal Lane.",
		"Crystal Lane is closed for repairs.",
		"Crystal Lane Elementary is closed.",
	}
	for _, in := range streets {
		out, err := newEngine(t, nil).NewSession("one").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out != in {
			t.Errorf("%q: street changed to %q", in, out)
		}
	}
	out, err := newEngine(t, nil).NewSession("one").Mask(ctx, "Ship it to 12 Crystal Lane.")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Crystal") || !strings.Contains(out, "Lane") {
		t.Errorf("street address not masked as an address: %q", out)
	}
	s := newEngine(t, nil).NewSession("one")
	if _, err := s.Mask(ctx, "Crystal Lane called this morning."); err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"Please ask Crystal Lane to sign.", "Crystal called back."} {
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "Crystal") || strings.Contains(out, "Lane") {
			t.Errorf("known %q: left raw in %q", in, out)
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("known %q: rehydrated to %q", in, back)
		}
	}
}

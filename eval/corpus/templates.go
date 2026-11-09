package corpus

import (
	"fmt"
	"math/rand/v2"
	"time"
)

const TaskPrompt = "You read email threads for a busy professional. First list every dated event as a JSON array of objects with fields \"date\" (YYYY-MM-DD) and \"person\" (full name of the person the event is about). Then add a two sentence summary."

type gen struct {
	r      *rand.Rand
	people []*Person
}

func (g *gen) cast(n int) []*Person {
	idx := g.r.Perm(len(g.people))
	out := make([]*Person, 0, n)
	if g.r.IntN(2) == 0 {
		common := 30 + g.r.IntN(len(g.people)-30)
		out = append(out, g.people[common])
	}
	for _, i := range idx {
		if len(out) == n {
			break
		}
		dup := false
		for _, p := range out {
			if p == g.people[i] {
				dup = true
			}
		}
		if !dup {
			out = append(out, g.people[i])
		}
	}
	g.r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (g *gen) day() time.Time {
	return time.Date(2026, time.Month(1+g.r.IntN(12)), 1+g.r.IntN(28), 8+g.r.IntN(10), g.r.IntN(60), 0, 0, time.UTC)
}

func (g *gen) after(d time.Time) time.Time { return d.AddDate(0, 0, 1+g.r.IntN(20)) }

func (g *gen) org() Org { return orgs[g.r.IntN(len(orgs))] }

func (g *gen) loc() string { return locations[g.r.IntN(len(locations))] }

func (g *gen) fmt() string { return DateFormats[g.r.IntN(len(DateFormats)-1)] }

func (g *gen) yfmt() string {
	f := []string{"iso", "long", "short", "us", "eu", "dotted"}
	return f[g.r.IntN(len(f))]
}

var fillers = []string{
	"Will you confirm when you can?",
	"I hope the numbers hold.",
	"Costs rose again this quarter.",
	"The grant paperwork may slip a week.",
	"Please mark it on the shared calendar.",
	"We start at dawn if the weather holds.",
	"The bill for the summer session is attached.",
	"To be frank, the art budget is thin.",
	"There is a grace period of ten days.",
	"No need to chase this one.",
	"Every penny counts this year.",
	"Sterling work on the draft, by the way.",
	"The ivy on the north wall needs cutting back.",
	"It was a joy to see the final cut.",
	"In good faith we extended the deadline.",
}

func (g *gen) filler() string { return fillers[g.r.IntN(len(fillers))] }

func (g *gen) refVariant(p *Person) string {
	vs := []string{"full", "first", "first", "nickname", "possessive", "title-last", "lower", "caps", "last-first", "markdown"}
	return vs[g.r.IntN(len(vs))]
}

func (g *gen) header(b *builder, from *Person, to []*Person, d time.Time, subject string) {
	b.t("From: ").p(from, "full").t(" <").email(from, g.r.IntN(2)).t(">\nTo: ")
	for i, p := range to {
		if i > 0 {
			b.t(", ")
		}
		b.p(p, "full").t(" <").email(p, g.r.IntN(2)).t(">")
	}
	b.t("\nDate: ").date(d, "header").t("\nSubject: " + subject + "\n\n")
}

func sep(b *builder) { b.t("\n\n---\n\n") }

type template struct {
	name string
	fn   func(g *gen, b *builder)
}

var templates = []template{
	{"scheduling", tplScheduling},
	{"onboarding", tplOnboarding},
	{"security", tplSecurity},
	{"billing", tplBilling},
	{"medical", tplMedical},
	{"moving", tplMoving},
	{"incident", tplIncident},
	{"family", tplFamily},
	{"claim", tplClaim},
	{"newsletter", tplNewsletter},
}

func tplScheduling(g *gen, b *builder) {
	c := g.cast(3)
	a, bb, cc := c[0], c[1], c[2]
	o, o2 := g.org(), g.org()
	d0, l := g.day(), g.loc()
	d1 := g.after(d0)
	d2 := g.after(d1)
	g.header(b, a, []*Person{bb}, d0, "Moving the review")
	b.t("Hi ").p(bb, "first").t(",\n\nCould we move the ").org(o).t(" review to ").date(d1, g.yfmt()).t("? I'm in ").loc(l).t(" until ").date(d0.AddDate(0, 0, 3), "monthday").t(", so a call works better. My cell is ").phone(a, g.r.IntN(4)).t(".\n\n"+g.filler()+"\n\nThanks,\n").p(a, "first")
	b.fact(d1, a)
	sep(b)
	g.header(b, bb, []*Person{a, cc}, d0.Add(3*time.Hour), "Re: Moving the review")
	b.t("Hi ").p(a, "nickname").t(",\n\n").date(d1, "weekday").t(" works. I'll loop in ").p(cc, "full").t(" from ").org(o2).t("; ").p(cc, "possessive").t(" team owns the forecast. You can reach ").p(cc, "first").t(" at ").email(cc, 0).t(".\n\n").p(bb, "first")
	sep(b)
	g.header(b, cc, []*Person{a, bb}, d0.Add(26*time.Hour), "Re: Moving the review")
	b.t("Confirmed for ").date(d2, "iso").t(" at 10:00 in ").loc(l).t(". ").p(a, "caps").t(", please bring the signed SOW. ").p(bb, g.refVariant(bb)).t(" said the deck is nearly done.\n\n-- ").p(cc, "first")
	b.fact(d2, cc)
}

func tplOnboarding(g *gen, b *builder) {
	c := g.cast(3)
	hr, hire, ec := c[0], c[1], c[2]
	o := g.org()
	d0 := g.day()
	start := g.after(d0)
	g.header(b, hr, []*Person{hire}, d0, "Welcome aboard, paperwork")
	b.t("Hi ").p(hire, "first").t(",\n\nWelcome to ").org(o).t("! Your first day is ").date(start, g.fmt()).t(". Please reply with your SSN, home address, date of birth and direct deposit details so payroll can set you up.\n\n").p(hr, "full").t("\nPeople Operations, ").org(o)
	b.fact(start, hire)
	sep(b)
	g.header(b, hire, []*Person{hr}, d0.Add(5*time.Hour), "Re: Welcome aboard, paperwork")
	s, sv := ssn(g.r)
	acct, rt := account(g.r), routing(g.r)
	b.t("Hi ").p(hr, "first").t(",\n\nHere you go:\n\n- Legal name: ").p(hire, "full").t("\n- SSN: ")
	b.e(TypeGovID, s, Digits(s), "ssn-"+sv, Digits(s))
	b.t("\n- DOB: ").date(hire.Birth, g.yfmt()).t("\n- Address: ").address(hire, true).t("\n- Routing number: ")
	b.e(TypeFinancial, rt, rt, "routing")
	b.t("\n- Account number: ")
	b.e(TypeFinancial, acct, acct, "account")
	b.t("\n- Emergency contact: ").p(ec, "full").t(", ").phone(ec, g.r.IntN(4)).t("\n\n"+g.filler()+"\n\nBest,\n").p(hire, g.refVariant(hire))
	b.fact(hire.Birth, hire)
}

func tplSecurity(g *gen, b *builder) {
	c := g.cast(2)
	u, it := c[0], c[1]
	o := g.org()
	d0 := g.day()
	code := otp(g.r)
	link, tok := resetLink(g.r, o.Domain)
	pw := password(g.r)
	key, kv := apiKey(g.r)
	b.t("From: ").org(o).t(" Accounts <no-reply@"+o.Domain+">\nTo: ").email(u, 0).t("\nDate: ").date(d0, "header").t("\nSubject: Your verification code\n\n")
	b.t("Hi ").p(u, "first").t(",\n\nYour verification code is ").secret(code, "otp").t(". It expires in 10 minutes.\n\nIf you asked to reset your password, use this link: ").secret(link, "reset-link", tok).t("\n\nIf this wasn't you, call us at 1-800-555-0199.")
	sep(b)
	g.header(b, u, []*Person{it}, d0.Add(20*time.Minute), "Fwd: Your verification code")
	b.t(g.filler()+" ").p(it, "first").t(", I keep getting these codes and I didn't ask for them. My temporary password was ").secret(pw, "password").t(" and I changed it on ").date(d0, g.fmt()).t(". Also, the deploy script still has this in it:\n\n    export API_KEY=").secret(key, kv).t("\n\nCan you rotate it? Thanks, ").p(u, g.refVariant(u))
	b.fact(d0, u)
}

func tplBilling(g *gen, b *builder) {
	c := g.cast(2)
	cust, agent := c[0], c[1]
	o := orgs[2+g.r.IntN(2)*7]
	d0 := g.day()
	stmt := d0.AddDate(0, 0, -10)
	cn, cv := card(g.r)
	acct := account(g.r)
	g.header(b, cust, []*Person{agent}, d0, "Disputed charge")
	b.t("Hello,\n\nI'm disputing a charge of $1,284.50 on my card ").e(TypeFinancial, cn, Digits(cn), "card-"+cv, Digits(cn)).t(" from the statement dated ").date(stmt, g.yfmt()).t(". The account number is ").e(TypeFinancial, acct, acct, "account").t(". Please send the paperwork to ").address(cust, g.r.IntN(2) == 0).t(".\n\n"+g.filler()+"\n\n").p(cust, "full").t("\n").phone(cust, g.r.IntN(4))
	b.fact(stmt, cust)
	sep(b)
	g.header(b, agent, []*Person{cust}, d0.Add(48*time.Hour), "Re: Disputed charge")
	b.t("Dear ").p(cust, "title-last").t(",\n\nThanks for contacting ").org(o).t(". We opened case 55-").t(randDigits(g.r, 5)).t(" and issued a provisional credit on ").date(d0.AddDate(0, 0, 2), g.fmt()).t(". A replacement card ending in ").t(Digits(cn)[len(Digits(cn))-4:]).t(" will ship this week.\n\nRegards,\n").p(agent, "full").t(", Card Services")
	b.fact(d0.AddDate(0, 0, 2), cust)
}

func tplMedical(g *gen, b *builder) {
	c := g.cast(3)
	pt, doc := c[0], c[1]
	o := orgs[12]
	appt := g.day()
	b.t("From: ").org(o).t(" <reminders@"+o.Domain+">\nTo: ").email(pt, 0).t("\nDate: ").date(appt.AddDate(0, 0, -3), "header").t("\nSubject: Appointment reminder\n\n")
	b.t("Dear ").p(pt, "full").t(" (DOB ").date(pt.Birth, g.yfmt()).t("),\n\nThis is a reminder of your appointment with Dr. ").p(doc, "title-last")
	b.t(" on ").date(appt, g.fmt()).t(" at our ").loc(g.loc()).t(" office. We have your number as ").phone(pt, g.r.IntN(4)).t(" and your address as ").address(pt, true).t(".\n\nPlease bring your insurance card.")
	b.fact(appt, pt)
	sep(b)
	g.header(b, pt, []*Person{doc}, appt.AddDate(0, 0, -2), "Re: Appointment reminder")
	b.t("Dr. ").p(doc, "title-last").t(", quick note before ").date(appt, "weekday").t(": I'm the ")
	b.quasi("left-handed oboist who sat in row C at the " + orgs[4].Name + " winter concert")
	b.t(", so you'll recognise me. My sister ").p(c[2], "first").t(" will drive me.\n\n").p(pt, g.refVariant(pt))
}

func tplMoving(g *gen, b *builder) {
	c := g.cast(2)
	tenant, pm := c[0], c[1]
	o := orgs[11]
	d0 := g.day()
	out := g.after(d0)
	acct := account(g.r)
	g.header(b, tenant, []*Person{pm}, d0, "Move-out and deposit")
	b.t("Hi ").p(pm, "first").t(",\n\nI'm moving out of ").address(tenant, true).t(" on ").date(out, g.fmt()).t(". My new address will be ")
	na := Address{Number: fmt.Sprintf("%d", 100+g.r.IntN(800)), Road: roads[g.r.IntN(len(roads))], City: cities[g.r.IntN(len(cities))].city, State: "CO", Zip: "80" + randDigits(g.r, 3)}
	full := na.Street() + ", " + na.City + ", " + na.State + " " + na.Zip
	b.ep(TypeAddress, full, full, "new-address", tenant.ID, na.Street())
	b.t(". Please refund the deposit to account ").e(TypeFinancial, acct, acct, "account").t(".\n\n"+g.filler()+"\n\n").p(tenant, "full").t(", ").phone(tenant, g.r.IntN(4))
	b.fact(out, tenant)
	sep(b)
	g.header(b, pm, []*Person{tenant}, d0.Add(30*time.Hour), "Re: Move-out and deposit")
	b.t("Thanks ").p(tenant, "first").t(". ").org(o).t(" will do the walkthrough on ").date(out.AddDate(0, 0, -1), g.yfmt()).t(". ").p(tenant, "possessive").t(" deposit is usually back within 21 days.\n\n").p(pm, "full")
	b.fact(out.AddDate(0, 0, -1), tenant)
}

func tplIncident(g *gen, b *builder) {
	c := g.cast(3)
	lead, oncall, sre := c[0], c[1], c[2]
	o := orgs[13]
	d := g.day()
	key, kv := apiKey(g.r)
	b.t("## Incident review ").date(d, "iso").t("\n\nONCALL: ").p(oncall, "caps").t("\nINCIDENT LEAD: ").p(lead, "caps").t("\n\n")
	b.t("Timeline (").org(o).t(", ").loc(g.loc()).t(" region):\n\n- ").date(d, "monthday").t(" 02:14 UTC: ").p(oncall, "first").t(" paged by the error budget alert.\n- 02:31: ").p(sre, "lower").t(" found the expired credential: `").secret(key, kv).t("`.\n- 03:05: rotated and redeployed.\n\nOwner pages: https://wiki.").t(o.Domain+"/people/").p(oncall, "url-slug").t(" and https://wiki."+o.Domain+"/people/").p(sre, "url-slug").t("\n\nFollow-up owner: ").p(lead, "markdown").t(" (").email(lead, 0).t("). " + g.filler())
	b.fact(d, oncall)
}

func tplFamily(g *gen, b *builder) {
	c := g.cast(3)
	a, sib, kid := c[0], c[1], c[2]
	bday := g.day()
	b.t("hey ").p(sib, "lower").t("!!\n\nso for ").p(kid, "possessive").t(" birthday on ").date(bday, "ordinal").t(" we're doing the park by our place, ").address(a, false).t(". ")
	b.p(sib, "nickname").t(" can you bring the cake? "+g.filler()+" ").p(kid, "first").t(" is in ")
	b.quasi(fmt.Sprintf("the %s-grade class at the only Montessori school in %s", []string{"second", "third", "fourth"}[g.r.IntN(3)], cities[g.r.IntN(len(cities))].city))
	b.t(" now, so invite the class.\n\ncall me at ").phone(a, g.r.IntN(4)).t(" or text, either is fine\n\nlove, ").p(a, "nickname")
	b.end()
	b.begin("assistant").t("Sounds fun! I've noted ").p(kid, "first").t("'s party on ").date(bday, "long").t(".")
	b.begin("user").t("thanks. also remind me that ").p(sib, "full").t(" owes me $40 and ").p(a, "possessive").t(" car is in the shop until ").date(g.after(bday), g.fmt()).t(".")
	b.tag("multi-turn")
	b.fact(bday, kid)
}

func tplClaim(g *gen, b *builder) {
	c := g.cast(2)
	ins, adj := c[0], c[1]
	o := orgs[6]
	loss := g.day()
	s, sv := ssn(g.r)
	pol := fmt.Sprintf("SIG-%s-%s", randDigits(g.r, 6), randDigits(g.r, 2))
	g.header(b, ins, []*Person{adj}, g.after(loss), "Claim for water damage")
	b.t("Hello ").p(adj, "title-last").t(",\n\nI'm filing a claim under policy ").e(TypeFinancial, pol, pol, "policy-number").t(" for water damage at ").address(ins, true).t(" on ").date(loss, g.fmt()).t(". For identity verification my SSN is ").e(TypeGovID, s, Digits(s), "ssn-"+sv, Digits(s)).t(" and my date of birth is ").date(ins.Birth, g.yfmt()).t(". Reach me at ").phone(ins, g.r.IntN(4)).t(" or ").email(ins, 1).t(".\n\n").p(ins, "full")
	b.fact(loss, ins)
	sep(b)
	g.header(b, adj, []*Person{ins}, g.after(loss).Add(50*time.Hour), "Re: Claim for water damage")
	insp := g.after(g.after(loss))
	b.t("Dear ").p(ins, "title-last").t(",\n\n").org(o).t(" assigned me to your claim. I can inspect on ").date(insp, g.fmt()).t(". "+g.filler()+"\n\n").p(adj, "full").t(", Senior Adjuster\n").phone(adj, g.r.IntN(4))
	b.fact(insp, ins)
}

func tplNewsletter(g *gen, b *builder) {
	c := g.cast(4)
	o := g.org()
	d := g.day()
	b.t("# ").org(o).t(" team update, ").date(d, "long").t("\n\n")
	b.t("**Welcome** to ").p(c[0], "markdown").t(", who joins from ").loc(g.loc()).t(". Say hi at [").p(c[0], "full").t("](mailto:").email(c[0], 0).t(").\n\n")
	b.t("**Kudos** to ").p(c[1], "markdown-mid").t(" for shipping the billing migration. ").p(c[1], "possessive").t(" demo is on ").date(g.after(d), g.fmt()).t(".\n\n")
	b.t("**Farewell** to ").p(c[2], "full").t(" (profile: https://people."+o.Domain+"/").p(c[2], "url-slug").t("). Last day ").date(g.after(d), g.yfmt()).t(".\n\n")
	b.t("**Birthdays**: ").p(c[3], g.refVariant(c[3])).t(" turns ").t(fmt.Sprintf("%d", 2026-c[3].Birth.Year())).t(" on ").date(time.Date(2026, c[3].Birth.Month(), c[3].Birth.Day(), 0, 0, 0, 0, time.UTC), "monthday").t(". " + g.filler())
	b.fact(d, c[0])
}

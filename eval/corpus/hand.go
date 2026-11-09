package corpus

import "time"

func d(y int, m time.Month, day, h, min int) time.Time {
	return time.Date(y, m, day, h, min, 0, 0, time.UTC)
}

func (g *gen) byID(id string) *Person {
	for _, p := range g.people {
		if p.ID == id {
			return p
		}
	}
	panic("no person " + id)
}

func handwritten(g *gen) []Item {
	var out []Item
	add := func(id string, fn func(b *builder)) {
		b := newBuilder("hand-"+id, "handwritten", id)
		b.system(TaskPrompt)
		fn(b)
		out = append(out, b.build())
	}

	will, rose := g.byID("P31"), g.byID("P32")
	add("will-and-rose", func(b *builder) {
		b.tag("common-word-name")
		b.t("Will you check this before Friday? ").p(will, "first").t(" will review the lease, and ").p(rose, "first").t(" said the rent rose 4% on ").date(d(2026, time.March, 2, 0, 0), "long").t(". If ").p(will, "first").t(" can't, ").p(rose, "possessive").t(" brother will. ").p(will, "full").t(" signs as guarantor; ").p(rose, "full").t(" is the tenant. Email ").email(will, 0).t(" or ").email(rose, 0).t(". A rose by any other name, as they say.")
		b.fact(d(2026, time.March, 2, 0, 0), rose)
	})

	mari := g.byID("P01")
	add("many-ways", func(b *builder) {
		b.tag("same-person-many-ways")
		b.t("Re: ").p(mari, "caps").t(" contract renewal\n\n").p(mari, "full").t(" called this morning. ").p(mari, "nickname").t(" wants the renewal signed by ").date(d(2026, time.April, 17, 0, 0), "us").t(". I told ").p(mari, "title-last").t(" we need legal first. ").p(mari, "possessive").t(" assistant will send the redlines; cc ").email(mari, 0).t(". Her page is https://team.copperline.example/").p(mari, "url-slug").t(". Filed under ").p(mari, "last-first").t(". Sign-off: ")
		b.quasi("M.Q.")
		b.t(" Also: ").p(mari, "lower").t(" prefers calls after 4pm, ").p(mari, "markdown").t(" is the decision maker.")
		b.fact(d(2026, time.April, 17, 0, 0), mari)
	})

	may, june := g.byID("P35"), g.byID("P21")
	add("may-june", func(b *builder) {
		b.tag("common-word-name", "nickname")
		b.t("Plan for spring: ").p(may, "first").t(" will meet ").p(june, "nickname").t(" in May, probably ").date(d(2026, time.May, 12, 0, 0), "short").t(". Then ").p(june, "full").t(" flies out on ").date(d(2026, time.June, 3, 0, 0), "eu").t(". It may rain in June, so ").p(may, "full").t(" booked the covered pavilion. ").p(june, "title-last").t(" confirmed by phone at ").phone(june, 0).t(".")
		b.fact(d(2026, time.May, 12, 0, 0), may)
		b.fact(d(2026, time.June, 3, 0, 0), june)
	})

	bart := g.byID("P26")
	add("name-only-in-identifiers", func(b *builder) {
		b.tag("name-in-email", "name-in-url")
		b.t("Please forward the signed NDA to ").email(bart, 0).t(" and book a slot at https://cal.example/").p(bart, "url-slug").t("/30min before ").date(d(2026, time.September, 9, 0, 0), "iso").t(". The shared folder is https://drive.example/u/").p(bart, "url-slug").t("/contracts.")
	})

	priya, thad := g.byID("P05"), g.byID("P02")
	add("multi-turn-history", func(b *builder) {
		b.tag("multi-turn")
		b.t("Summarize: ").p(priya, "full").t(" (").email(priya, 0).t(") asked ").p(thad, "full").t(" to approve the ").org(orgs[1]).t(" budget by ").date(d(2026, time.July, 30, 0, 0), "long").t(".")
		b.begin("assistant").p(priya, "first").t(" needs ").p(thad, "possessive").t(" approval on the ").org(orgs[1]).t(" budget by ").date(d(2026, time.July, 30, 0, 0), "monthday").t(".")
		b.begin("user").t("Draft a reply from ").p(thad, "nickname").t(" to ").p(priya, "nickname").t(" saying yes, and cc ").email(thad, 0).t(".")
		b.fact(d(2026, time.July, 30, 0, 0), thad)
	})

	add("secrets-dump", func(b *builder) {
		b.tag("secrets")
		k1, k1v := apiKey(g.r)
		k2, k2v := apiKey(g.r)
		pw := password(g.r)
		b.t("Here is my .env, why does the job fail?\n\n```\nOPENAI_API_KEY=").secret(k1, k1v).t("\nSTORAGE_KEY=").secret(k2, k2v).t("\nDATABASE_URL=").secret("postgres://svc_reports:"+pw+"@db.internal.example:5432/reports", "connection-string", pw).t("\nADMIN_PASSWORD=").secret(pw, "password").t("\nSESSION_TOKEN=").secret("eyJhbGciOiJIUzI1NiJ9."+randFrom(g.r, base62, 40)+"."+randFrom(g.r, base62, 43), "jwt").t("\n```\n\nThe 2FA backup code they gave me was ").secret(otp(g.r), "otp").t(".")
	})

	add("financial-table", func(b *builder) {
		b.tag("financial")
		c1, c1v := card(g.r)
		c2, c2v := card(g.r)
		b.t("| Holder | Instrument | Number |\n|---|---|---|\n| ").p(g.byID("P14"), "full").t(" | Visa | ").e(TypeFinancial, c1, Digits(c1), "card-"+c1v, Digits(c1)).t(" |\n| ").p(g.byID("P17"), "full").t(" | Card | ").e(TypeFinancial, c2, Digits(c2), "card-"+c2v, Digits(c2)).t(" |\n| ").p(g.byID("P22"), "full").t(" | IBAN | ").e(TypeFinancial, "GB82 WEST 1234 5698 7654 32", "GB82WEST12345698765432", "iban", "GB82WEST12345698765432").t(" |\n\nWhich of these expire first? Totals are due ").date(d(2026, time.October, 1, 0, 0), "dotted").t(".")
	})

	add("quasi-only", func(b *builder) {
		b.tag("quasi-identifier")
		b.t("Can you write a short thank-you note to ")
		b.quasi("the only female captain at Fire Station 9 in Spokane, who won the 2019 county chili cook-off")
		b.t("? She pulled ").p(g.byID("P40"), "first").t(" out of the river on ").date(d(2026, time.August, 14, 0, 0), "long").t(". Her son is ")
		b.quasi("the goalkeeper for the under-12 team that won the regional final in Eugene")
		b.t(".")
		b.fact(d(2026, time.August, 14, 0, 0), g.byID("P40"))
	})

	ravi := g.byID("P20")
	add("chat-log", func(b *builder) {
		b.tag("chat-handles")
		b.t("[09:14] ").p(ravi, "lower").t(": morning, did ").p(g.byID("P43"), "first").t(" push the fix?\n[09:15] ").p(g.byID("P43"), "handle").t(": yes, ").p(g.byID("P43"), "first").t(" here, to be frank it was ugly\n[09:16] ").p(ravi, "lower").t(": (").p(ravi, "nickname").t(" approves) ship it on ").date(d(2026, time.November, 20, 0, 0), "monthday").t("\n[09:17] @").p(ravi, "first").t(": ping ").p(g.byID("P48"), "first").t(" to chase the vendor")
		b.fact(d(2026, time.November, 20, 0, 0), ravi)
	})

	corwin := g.byID("P18")
	add("all-caps-legal", func(b *builder) {
		b.tag("all-caps")
		b.t("THIS LEASE IS MADE ON ").e(TypeDate, "MARCH 1, 2026", "2026-03-01", "caps", "2026-03-01", "March 1, 2026").t(" BETWEEN ").p(corwin, "caps").t(" (\"TENANT\"), RESIDING AT ")
		a := corwin.Address
		up := func(s string) string {
			out := []byte(s)
			for i, c := range out {
				if c >= 'a' && c <= 'z' {
					out[i] = c - 32
				}
			}
			return string(out)
		}
		full := up(a.Street() + ", " + a.City + ", " + a.State + " " + a.Zip)
		b.ep(TypeAddress, full, full, "caps", corwin.ID, up(a.Street()), a.Street())
		b.t(", AND ").e(TypeOrg, "IVORYGATE LEGAL LLP", "Ivorygate Legal LLP", "caps").t(" AS AGENT FOR THE LANDLORD. TENANT SSN: ")
		s, sv := ssn(g.r)
		b.e(TypeGovID, s, Digits(s), "ssn-"+sv, Digits(s)).t(".")
		b.fact(d(2026, time.March, 1, 0, 0), corwin)
	})

	kobby := g.byID("P24")
	add("person-org-overlap", func(b *builder) {
		b.tag("org-person-overlap")
		b.t("Our counsel is ").e(TypeOrg, "Fairweather & Hollowell LLP", "Fairweather & Hollowell LLP", "person-like-org").t(". Separately, ").p(kobby, "title-last").t(" (no relation to the firm) asked for the file by ").date(d(2026, time.February, 6, 0, 0), "short").t(". ").e(TypeOrg, "Fairweather & Hollowell LLP", "Fairweather & Hollowell LLP", "person-like-org").t(" will respond for us. ").p(kobby, "nickname").t(" can be reached at ").email(kobby, 0).t(".")
		b.fact(d(2026, time.February, 6, 0, 0), kobby)
	})

	rose2, grant := g.byID("P32"), g.byID("P33")
	add("quoted-reply-chain", func(b *builder) {
		b.tag("quoted-reply")
		b.t("Sounds good, I'll take it from here.\n\n").p(grant, "first").t("\n\nOn ").date(d(2026, time.March, 3, 9, 14), "short").t(" at 9:14 AM ").p(rose2, "full").t(" <").email(rose2, 0).t("> wrote:\n> ").p(grant, "first").t(", the grant report is due ").date(d(2026, time.March, 20, 0, 0), "weekday").t(".\n> Can you send it to ").p(g.byID("P34"), "full").t("? I hope ").p(g.byID("P34"), "first").t(" is back from ").loc("Halifax").t(".\n>\n> On ").date(d(2026, time.March, 2, 17, 2), "short").t(" ").p(g.byID("P39"), "full").t(" wrote:\n>> The bill is attached. ").p(g.byID("P36"), "first").t(" will mark it paid.\n>> -- ").p(g.byID("P39"), "first").t(", ").phone(g.byID("P39"), 2)
		b.fact(d(2026, time.March, 20, 0, 0), grant)
	})

	return out
}

package main

var itemsWork = []def{
	it("support-ticket", "last-first,title-last,card,first-initial",
		sys(`You draft customer support replies. Be brief, apologize once, state what was done.`),
		usr(`Write the closing email to the customer.

Ticket #48213  Priority: High  Channel: email
Customer: {{p|omar|last-first|Abdelrahman, Omar}}
Email: {{e|omar.abdelrahman@example.net}}
Phone: {{ph|+1 (646) 555-0123}}
Opened: {{d|2026-11-02}} 08:17 UTC

Customer writes:
"I was charged twice for my November subscription. Card ending 4242 (full number {{f|card-spaced|4242 4242 4242 4242}} if you need it, exp 09/28). Please refund one of them. {{p|omar|first|Omar}}"

Agent notes ({{p|dawn|first-initial|Dawn M.}}):
- Verified identity via last 4 + billing ZIP.
- Duplicate charge confirmed, refund issued {{d|2026-11-03|11/3}}.
- Mr. {{p|omar|title-last|Abdelrahman}} asked that we stop calling his wife's number ({{ph|646-555-0190}}) which is still on the account.`)),

	it("crm-notes", "initials,caps,common-word-name,person-like-org",
		usr(`Account: {{o|Hashemi Brothers Rugs|person-like-org}}
Owner: {{p|rich|full|Rich Tavares}}
Last touch: {{d|2026-10-29|10/29/26}}

Notes:
{{d|2026-10-14|10/14}} - {{p|rich|initials|RT}}: Intro call with {{p|layla|full|Layla Haddad}} (ops manager). Rich text editor in the quoting tool keeps eating her formatting, she wants plain text export.
{{d|2026-10-21|10/21}} - {{p|rich|initials|RT}}: {{p|layla|first|Layla}} looped in their IT guy, {{p|yusuf|first|Yusuf}}. {{p|yusuf|full|Yusuf Al-Khatib}} <{{e|yusuf@hashemi-rugs.example}}>, direct {{ph|503-555-0136}}.
{{d|2026-10-29|10/29}} - {{p|rich|initials|RT}}: {{p|yusuf|caps|AL-KHATIB}} approved the pilot. PO to follow. {{p|layla|initials|LH}} wants invoices addressed to {{a|908 Corbel Street, Suite 12, Portland, OR 97209}}.

Draft my follow-up email to both of them and suggest a next step.`)),

	it("support-chat", "first-name,title-last,reset-link,misspelled-name",
		usr(`QA review: did the agent follow the verification policy? Transcript:

Chat started 14:02
Visitor ({{p|fatou|first|Fatou}}): hi, I can't log in. it says my account is locked
Agent ({{p|daniel|first|Daniel}}): Hi {{p|fatou|first|Fatou}}, happy to help. Can I have the email on the account?
Visitor ({{p|fatou|first|Fatou}}): {{e|fatou.diallo@example.com}}
Agent ({{p|daniel|first|Daniel}}): Thanks. For verification, the last four of the phone number on file?
Visitor ({{p|fatou|first|Fatou}}): 0148. full is {{ph|917 555 0148}}
Agent ({{p|daniel|first|Daniel}}): Perfect. I've sent a reset link: {{s|reset-link|https://id.brightledger.example/reset?token=9fK2mQx7LpV4sT8wZb3nYc6h|9fK2mQx7LpV4sT8wZb3nYc6h}}
Visitor ({{p|fatou|first|Fatou}}): got it thx. also my surname is spelled wrong in my profile, it's {{p|fatou|last|Diallo}} not {{p|fatou|misspelled|Dialo}}
Agent ({{p|daniel|first|Daniel}}): Fixed, Ms. {{p|fatou|title-last|Diallo}}. Anything else?`)),

	it("support-ticket", "caps,multi-line-address,person-like-org,first-name",
		usr(`Return label request, order R-20931

SHIP TO:
{{p|kwabena|caps|KWABENA ASANTE-YEBOAH}}
C/O {{o|ASANTE FABRICS LTD|person-like-org}}
{{a|4410 GREENBRIAR PKWY APT 9C
ATLANTA GA 30331}}

Customer called twice ({{p|kwabena|first|Kwabena}} says the first label printed blank). Callback number {{ph|470-555-0102}}. He prefers email: {{e|k.asante.yeboah@example.org}}. Second caller on the account is his sister {{p|+Ama Asante-Yeboah|first|Ama}}, do not discuss billing with her.

Write an internal note for the next agent, 3 bullets max.`)),

	it("support-email", "caps-shouting,uk-phone,ascii-fold,bank-account,diacritics",
		sys(`You triage inbound complaints for a delivery company: summarize, classify urgency, propose a reply.`),
		usr(`Subject: STILL NO DELIVERY!!!

I ordered a washing machine on {{d|2026-10-28|28th October}} and your driver "{{p|bogdan|first|Bogdan}}" called me from {{ph|07700 900171}} saying he was outside. HE WAS NOT. I live at {{a|3 Wren Close, Stockport SK4 2BN}}, not Wren Court. My neighbour Mrs {{p|+Morag MacGregor|title-last|MacGregor}} saw the van drive past. I want a refund to my account, sort code {{f|sort-code|40-47-84}} acct {{f|account|70872490}}, or I will be contacting Trading Standards.

{{p|kasia|full|Katarzyna Dąbrowska}} (your system has me as "{{p|kasia|ascii-fold|Dabrowska Katarzyna}}", which is also wrong)`)),

	it("recruiting", "last-first,initial-last,first-initial,passport,product-name",
		usr(`Interview debrief: Senior Data Engineer, req 2291

Candidate: {{p|arjun|last-first|Venkataraman, Arjun}}
Panel: {{p|miles|initial-last|M. Abara}} (hiring manager), {{p|ayesha|first|Ayesha}}, {{p|zdenek|first-initial|Zdeněk N.}}
Recruiter: {{p|emily|full|Emily Carter}}

{{p|ayesha|first|Ayesha}}: strong on SQL, weaker on streaming. "He knew {{o|Kafka|product}} internals better than I do though."
{{p|zdenek|first|Zdeněk}}: lean hire. {{p|arjun|first|Arjun}} was candid that he left {{o|Bluefin Analytics}} after a dispute with his manager.
{{p|miles|first|Miles}}: hire. Wants to check the reference from Dr. {{p|priya|title-last|Raghunathan}} first ({{e|priyanka.raghunathan@uni-meridian.example}}).
Comp: currently 148k base, asking 165k. Visa: H-1B transfer, passport no. {{g|passport|Z4417826}} on file.
Next step: {{p|emily|first|Emily}} to call {{p|arjun|first|Arjun}} Friday on {{ph|669-555-0175}}.

Turn this into a hiring committee summary.`)),

	it("hr-case", "quasi-identifier,common-word-name,initial-last",
		sys(`You help HR partners write neutral, factual case summaries. Never add facts.`),
		usr(`Confidential, HR case notes. Please structure this into a formal summary without changing facts.

On Tuesday the complainant ({{q|the only female welder on the night shift at the Tulsa plant, who joined this spring from the apprenticeship programme}}) reported that her supervisor, {{p|bill|first|Bill}}, has repeatedly assigned her cleanup duty instead of fabrication work. She says {{p|bill|full|Bill Oyelaran}} told her "the bill for your mistakes comes out of my bonus". Witness: {{p|pat|full|Pat Ngata}}, who asked not to be named in the written record. Complainant does not want her name used at this stage either. HR partner: {{p|siobhan|initial-last|S. O'Riordan}}.`)),

	it("hr-onboarding", "ssn,nickname,routing,first-initial,form",
		usr(`New hire paperwork, please check it for missing or inconsistent fields:

Legal name: {{p|harpreet|full|Harpreet Kaur Sandhu}}
Preferred name: {{p|harpreet|nickname|Preet}}
SSN: {{g|ssn-dashed|666-41-8830}}
DOB: {{d|1991-07-23}}
Address: {{a|1529 Juniper Bend, Sacramento, CA 95818}}
Direct deposit: routing {{f|routing|123456780}}, account {{f|account|000482915573}}
Emergency contact: {{p|rohan|full|Rohan Mehta}} (spouse), {{ph|(916) 555-0158}}
Start date: {{d|2026-12-07|Monday, December 7}}
Manager: {{p|amber|first-initial|Amber N.}}
Laptop shipped to home address, tracking sent to {{e|preet.sandhu@example.com}}`)),

	it("recruiting", "lowercase-name,url-slug,person-like-org,common-word-name",
		usr(`is this legit? and draft a polite no either way

"hey! saw your post. we're hiring for a staff role at {{o|Tannenbaum & Reed|person-like-org}} (yes, the {{p|olive|full|Olive Tannenbaum}}, she still runs eng). would love to intro you to {{p|drew|lower|drew kessler}} who leads platform. you two overlapped at {{o|Bluefin}} I think? my cal: https://cal.quillhire.example/{{p|sarah|url-slug|sarah-jennings}}/30min or just text {{ph|628-555-0119}}. {{p|sarah|first|Sarah}}"`)),

	it("legal-letter", "caps,title-last,possessive,person-like-org,address",
		usr(`I got this today. What are my options and what should I not say if I call them?

VIA EMAIL AND CERTIFIED MAIL

RE: {{p|skip|caps|HENSLEY}} v. {{o|CASTELLANO CONSTRUCTION LLC|person-like-org}}, Case No. 26-CV-04417

Dear Mr. {{p|rob|title-last|Castellano}}:

This firm represents {{p|skip|full|Skip Hensley}} in connection with the injuries he sustained on {{d|2026-08-14|August 14, 2026}} at your job site at {{a|2200 Quarry Road, Asheville, NC 28806}}. Mr. {{p|skip|possessive|Hensley}}'s medical expenses to date exceed $48,000. We demand that you preserve all site logs and contact your insurer. Absent a response by {{d|2026-12-01|December 1, 2026}}, we will file without further notice.

Very truly yours,
{{p|siobhan|full|Siobhán O'Riordan}}, Esq.
{{o|O'Riordan Hale LLP|person-like-org}}
{{e|soriordan@oriordanhale.example}} | {{ph|(828) 555-0127}}`)),

	it("legal-lease", "caps,particle-name,iban,title-last",
		usr(`My landlord sent this. Is clause 4 enforceable?

"THIS AGREEMENT is made on {{d|2026-09-01|1 September 2026}} between {{p|willem|caps|WILLEM VAN DER HEIJDEN}} ("Landlord") of {{a|Prinsengracht 512-II, 1017 KH Amsterdam}} and {{p|fatima|caps|FATIMA ZAHRA EL IDRISSI}} ("Tenant"). [...] 4. The Tenant shall not keep animals without the written consent of Mr {{p|willem|title-last|van der Heijden}}."

I pay by transfer to {{f|iban|NL91 ABNA 0417 1643 00}} every month, never late. He has known about the cat since day one.
{{p|fatima|first|Fatima}}`)),

	it("legal-declaration", "family-name-first,last-first,diacritics,gov-id",
		usr(`Declaration draft, please fix grammar only and keep every fact as is:

I, {{p|bao|full|Trần Quốc Bảo}}, born {{d|1988-02-12|12 February 1988}} in {{l|Đà Nẵng}}, declare that I have lived with my partner {{p|joao|last-first|CARVALHO DE SOUZA, João Pedro}} since March 2023 at {{a|96 Halsey Street, Apt 4, Brooklyn, NY 11216}}. My alien registration number is {{g|a-number|A098-765-432}}. We share a joint account and a lease. {{p|joao|first|João Pedro}} supports my application fully, and so do his parents.`)),

	it("legal-notes", "mononym,first-name,title-last,common-word-name",
		usr(`From my notes after the call with the estate lawyer:
- executor is now aunt {{p|moira|first|Moira}}, not uncle {{p|declan|first|Declan}} (he declined)
- the will names "my nephew {{p|thiha|full|Thiha}}" for the {{l|Mandalay}} flat. lawyer says a single name is fine, it's his full legal name
- {{p|declan|full|Declan O'Shaughnessy}} still has the original deed. she will write to him
- lawyer = Ms. {{p|noa|title-last|Ben-David}}, {{e|noa@bendavid-law.example}}, fee 350/hr

Will the will hold up if {{p|thiha|full|Thiha}} is abroad? what should I ask next time`)),

	it("meeting-transcript", "caps,speaker-labels,nickname,same-surface-tool-and-person,place-as-name",
		sys(`You turn meeting transcripts into action items with owners.`),
		usr(`[00:00:04] {{p|meiling|caps|MEI-LING TSAI}}: Okay, recording. This is the Q4 roadmap sync for {{d|2026-11-17|November 17th}}.
[00:00:09] {{p|rohan|caps|ROHAN MEHTA}}: Before we start, is {{p|jasper|first|Jasper}} joining?
[00:00:12] {{p|meiling|caps|MEI-LING TSAI}}: {{p|jasper|first|Jasper}}'s out sick. And to be clear the reporting migration is about {{o|Jasper|product}} the tool, so nothing is blocked on him.
[00:00:21] {{p|rohan|caps|ROHAN MEHTA}}: Ha. Fine. So, {{p|meiling|nickname|Mei}}, the {{l|Austin}} datacenter move.
[00:00:25] {{p|meiling|caps|MEI-LING TSAI}}: Right. {{l|Austin}} goes dark the first week of April. {{p|rohan|first|Rohan}}, you own the cutover plan. {{p|diego|first|Diego}} owns comms.
[00:00:33] {{p|diego|caps|DIEGO HERNÁNDEZ VARGAS}}: Can I get that in writing? Last time "{{p|diego|first|Diego}} owns comms" turned into {{p|diego|first|Diego}} owns everything.
[00:00:38] {{p|meiling|caps|MEI-LING TSAI}}: Noted, Mr. {{p|diego|title-last|Hernández}}.`)),

	it("meeting-transcript", "speaker-labels,first-initial,last-name,product-name,diacritics",
		usr(`standup transcript, need a 3 bullet summary for the channel:

Speaker 1 ({{p|soren|first|Søren}}): yesterday I finished the chart for the gateway, today pairing with {{p|akwasi|first|Akwasi}} on the {{o|Thanos|product}} retention thing.
Speaker 2 ({{p|akwasi|first-initial|Akwasi B.}}): yeah, and {{o|Loki|product}} is dropping logs again. I think it's {{p|soren|possessive|Søren}}'s sidecar change to be honest.
Speaker 1 ({{p|soren|first|Søren}}): my change is innocent until proven guilty.
Speaker 3 (Unknown): can someone add {{p|soren|last|Kjeldgaard}} to the on-call calendar, he's not on it
Speaker 2 ({{p|akwasi|first-initial|Akwasi B.}}): {{p|akwasi|last|Boateng}} isn't either. I'll do both.`)),

	it("meeting-minutes", "initials,mononym,first-initial,title-last,common-word-name",
		usr(`Minutes, Maple Grove Tenants' Association, {{d|2026-11-05|5 Nov 2026}}
Present: {{p|adjei|full|J. R. Adjei}} (chair), {{p|milica|full|Milica Jovanović}} (secretary), {{p|oksana|full|Oksana Hrytsenko}}, {{p|lane|full|Lane Pruitt}}, {{p|thiha|full|Thiha}}
Apologies: {{p|pat|first-initial|Pat N.}}

1. Bike lane on Garfield Ave. {{p|lane|initials|LP}} reported the council will repaint the lane in spring. {{p|adjei|initials|JRA}} to write to Cllr. {{p|tom|title-last|Fletcher}}.
2. Boiler. {{p|oksana|initials|OH}} obtained two quotes; {{p|milica|initials|MJ}} to circulate. {{p|thiha|full|Thiha}} proposed a levy of $40 per unit. Carried.
3. AOB. Mr. {{p|adjei|title-last|Adjei}} thanked {{p|milica|first|Milica}} for five years as secretary.

Please write these up as a friendly newsletter paragraph for the building.`)),

	it("clinical-note", "caps,last-first,initial-last,dob,first-name",
		sys(`You are a clinical documentation assistant. Rewrite notes for a patient-friendly after-visit summary.`),
		usr(`Pt: {{p|jasper|last-first|OKORO, JASPER}}  DOB: {{d|1962-03-14|03/14/1962}}  MRN: 00482913
Seen {{d|2026-11-12|12-Nov-2026}} by Dr. {{p|ayesha|initial-last|A. Siddiqui}}

S: 64M presents w/ 3 wks exertional dyspnea. Wife ({{p|folake|first|Folake}}) reports he "sleeps propped up on three pillows". Hx HTN, T2DM.
O: BP 152/94, HR 88, SpO2 95% RA. Bibasilar crackles. 2+ pitting edema.
A: New-onset HF, likely HFrEF.
P: Echo this week. Start furosemide 40 mg daily. F/u 2 wks. Mr. {{p|jasper|title-last|Okoro}} agrees w/ plan. Daughter {{p|+Ngozi Okoro|first|Ngozi}} is HCP, cell {{ph|857-555-0111}}.`)),

	it("clinical-letter", "caps,title-last,nhs-number,uk-address,quasi-identifier",
		usr(`Can you explain this letter to me in plain language? It's about my sister.

Dear Dr {{p|eilidh|title-last|MacLeod}},

Re: Mrs {{p|layla|caps|Layla HADDAD}}, d.o.b. {{d|1979-05-22|22/05/1979}}, NHS no. {{g|nhs|999 123 4560}}
{{a|14 Tern Avenue, Leith, Edinburgh EH6 7PQ}}

Thank you for referring this {{q|47-year-old deputy head of the only Gaelic-medium primary school in Leith}} with a six month history of migraine with aura. She was reviewed in clinic on {{d|2026-11-03|3rd November 2026}}. MRI brain was normal. I have started propranolol 40 mg twice daily and will review in three months. Mrs {{p|layla|title-last|Haddad}} has been advised not to drive during an aura.

Yours sincerely,
Dr {{p|hamish|full|Hamish MacGregor}}, Consultant Neurologist`)),

	it("clinical-discharge", "ascii-fold,diacritics,family-name-first,title-last",
		usr(`my mom's discharge papers say this, can you explain in plain english?

"Pt {{p|chau|ascii-fold|Nguyen Thi Minh Chau}}, 71F, admitted {{d|2026-11-08|11/08/26}} w/ CAP. Tx ceftriaxone + azithro x5d. D/c home {{d|2026-11-13|11/13/26}} in care of son. PCP f/u Dr. {{p|kenji|title-last|Morishita}} 1 wk. Return if SOB, fever >101, confusion."

fwiw her name is actually spelled {{p|chau|full|Nguyễn Thị Minh Châu}}, they always drop the accents. she goes by {{p|chau|first|Châu}}. should I get that corrected before the follow-up?`)),

	it("clinical-lab", "last-first,dob,policy-number",
		usr(`what does this mean? should I be worried

Lab report
Patient: {{p|emily|full|Emily Carter}}   Sex: F   DOB: {{d|1994-07-09|July 9, 1994}}
Ordering provider: {{p|arjun|last-first|Venkataraman, A.}}, MD
Collected: {{d|2026-10-30}} 07:42   Reported: {{d|2026-10-31}}

TSH 6.8 mIU/L (H)   Free T4 0.9 ng/dL
Insurance: {{o|Castlerock Mutual}}, member ID {{f|policy-number|CRM-88104-2217}}, group 40031
Results released to patient portal. Questions: {{ph|415-555-0191}}.`)),
}

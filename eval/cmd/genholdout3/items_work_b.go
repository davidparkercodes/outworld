package main

var itemsWorkB = []def{
	work("offer-letter", "hr,letter",
		usr(`Proofread this offer letter and check the dates are consistent with a four-week notice period.

{{o|Corvane Freight Ltd}}
{{a|Unit 14, Tidewater Business Park, 220 Quayside Road, Bristol BS1 9ZZ}}

{{d|2026-06-08|8 June 2026}}

PRIVATE AND CONFIDENTIAL
Ms {{p|akosua}}
{{a|Flat 3, 41 Osu Badu Street, Airport West, Accra}}

Dear {{p|akosua|first}},

I am delighted to offer you the position of Senior Customs Analyst, reporting to {{p|dale}}, based at our {{l|Rotterdam}} office ({{a|Waalhaven Oostzijde 85, 3087 BM Rotterdam}}).

Start date: {{d|2026-08-17|Monday 17 August 2026}}, subject to your permit being granted.
Salary: EUR 74,000 gross per year, paid monthly on the 25th.
Relocation: up to EUR 6,000 against receipts.
Probation: two months, ending {{d|2026-10-16|16 October 2026}}.

To set you up on payroll we will need, before your first day: a copy of your passport, your Dutch BSN once issued, and an IBAN in your name. If you do not yet have a Dutch account, {{p|anouk}} in payroll ({{e|payroll.nl@corvanefreight.example}}, {{ph|+31 10 555 0144}}) can explain the options.

Please sign and return this letter by {{d|2026-06-12|12 June 2026}}. If you have questions, call me on {{ph|0117 496 0105}} or my mobile.

Yours sincerely,

{{p|ingrid}}
People Director

cc: {{p|dale|initial-last|D. Ferreira-Stone}}, {{p|rashida|initial-last|R. Begum}}

I, {{p|akosua|caps}}, accept the offer on the terms above.
Signed: ______________  Date: ______________`)).on("2026-08-17", "akosua", "2026-06-12", "akosua"),

	work("expense-report", "finance,expenses",
		usr(`Check this expense claim against policy (economy flights, hotels under EUR 180, meals under EUR 60 a day) and list what finance will bounce.

EXPENSE CLAIM {{n|identifier|EXP-2026-0619}}
Claimant: {{p|mateus}} ({{p|mateus|nickname}}), cost centre {{n|number|4410}}
Trip: {{l|Bristol}} to {{l|Porto}} to {{l|Lisbon}}, customer visits, {{d|2026-05-11|11/05/2026}} to {{d|2026-05-14|14/05/2026}}
Approver: {{p|lucinda}}

Date        Item                                             Amount    Paid with
{{d|2026-05-11|11/05}}       Flight BRS-OPO, {{o|TAP|company}}, booking ref QK7P2M              EUR 212.40   corp card 4000 **** **** 5556
{{d|2026-05-11|11/05}}       Taxi airport to {{o|Hotel Infante Sagres}}                  EUR 31.00    cash
{{d|2026-05-11|11/05}}       Dinner with {{p|beatriz}} ({{o|Douro Vinhos Transportes}})   EUR 96.50    corp card
{{d|2026-05-12|12/05}}       Hotel, 1 night                                   EUR 174.00   corp card
{{d|2026-05-12|12/05}}       Train Porto Campanhã to Lisboa Oriente           EUR 33.90    personal card
{{d|2026-05-13|13/05}}       Hotel {{l|Lisbon}}, 2 nights                             EUR 398.00   corp card
{{d|2026-05-13|13/05}}       Lunch, self                                      EUR 18.20    personal card
{{d|2026-05-14|14/05}}       Flight LIS-BRS, changed fare class               EUR 344.10   corp card
{{d|2026-05-14|14/05}}       Gift for {{p|beatriz|nickname}} (port, {{n|eponym|Taylor's}} 10 year)           EUR 42.00    personal card

Reimburse personal spend (EUR 94.10) to: {{p|mateus|caps|MATEUS FIGUEIREDO DOS ANJOS}}, IBAN {{f|pt-iban|PT02 0035 0683 0000 4821 9307 1}}, BIC CGDIPTPL
Full corporate card number for the audit sample: {{f|card-visa|4000 0566 5566 5556}}
Notes: {{p|beatriz|first}} is the buyer, dinner was her, me and her colleague {{p|joaopedro|first}}. Fare change on the 14th because the customer moved the meeting, {{p|lucinda|nickname}} approved by text.`)),

	work("standup-initials", "meeting,initials-only",
		usr(`turn into a status update for the channel, bullet per person, full first names not initials. key is at the bottom.

standup {{d|2026-04-08|weds 8 apr}}
{{p|obinna|initials|OC}} - customs list done, 23 ours 18 theirs. blocked on {{p|siobhan|initials|SNB}} for the PoA
{{p|siobhan|initials|SNB}} - spoke to {{p|halvard|last}}, PoA signed, scan coming today. will unblock {{p|obinna|initials|OC}} by lunch
{{p|tevita|initials|TFL}} - rota comparison half done, {{p|ingrid|initials|IAT}} owes me {{l|bergen}} numbers
{{p|ingrid|initials|IAT}} - numbers tonight. also: {{p|rashida|initials|RB}} pls book the big room for the 17th
{{p|rashida|initials|RB}} - booked. absence report sent to {{p|marguerite|initials|MDB}} + {{p|tevita|initials|TFL}} yesterday
{{p|marguerite|initials|MDB}} - nothing from me. out thurs/fri
{{p|jurgen|initials|JW}} - {{l|duisburg}} contract countersigned!! invoice 1 goes out {{d|2026-04-15|15.4.}}
{{p|penelope|initials|PAS}} - last week. handover doc is in the drive. ask {{p|leilani|initials|LK}} after i'm gone

key: {{p|obinna|initials|OC}}={{p|obinna|first}}, {{p|siobhan|initials|SNB}}={{p|siobhan|first}}, {{p|tevita|initials|TFL}}={{p|tevita|first}}, {{p|ingrid|initials|IAT}}={{p|ingrid|first}}, {{p|rashida|initials|RB}}={{p|rashida|first}}, {{p|marguerite|initials|MDB}}={{p|marguerite|first}}, {{p|jurgen|initials|JW}}={{p|jurgen|first}}, {{p|penelope|initials|PAS}}={{p|penelope|first}}, {{p|leilani|initials|LK}}={{p|leilani|first}}`)),

	work("call-transcript", "transcript,speaker-labels",
		usr(`Pull out the commitments each side made on this call. Auto-transcript, so some names are mangled.

Call recording {{d|2026-03-24|24 March 2026}} 15:02, {{o|Corvane Freight}} / {{o|Aegean Reefer Lines}}
Speakers: {{p|jordan|first-initial|JORDAN M-H}}, {{p|stavros|first-initial|STAVROS P}}, UNKNOWN 1

{{p|jordan|caps-first}}: Thanks for making time, {{p|stavros|first}}. I have {{p|ngozi|first}} with me taking notes.
{{p|stavros|caps-first}}: Good afternoon. Before we start, it is {{p|stavros|misspelled|Stavro}}, not {{x|wrong-name|Stephen}}, your system keeps writing {{x|wrong-name|Stephen}}.
{{p|jordan|caps-first}}: Apologies. I will get that fixed. So, the reefer lane {{l|Piraeus}} to {{l|Southampton}}.
{{p|stavros|caps-first}}: Yes. We need eighteen slots a week from the first of June, not twelve. My operations lady, Mrs {{p|gulsen|misspelled|Gulshen}}, no sorry, that is the Turkish client, I mean Mrs {{p|ludovica|title-last|Sanfilippo}}, she sent the forecast.
UNKNOWN 1: We have it. This is {{p|ngozi|first}}. The forecast came from {{e|l.sanfilippo@aegeanreefer.example}} on the {{d|2026-03-19|nineteenth of March}}.
{{p|jordan|caps-first}}: Eighteen is possible if you commit for six months. Price per slot stays where it is.
{{p|stavros|caps-first}}: Six months, okay. But I want a named contact for weekends. Last time I called the number, it was, wait, {{ph|oh one one seven, four nine six, oh one oh five|spoken}}, nobody answered.
{{p|jordan|caps-first}}: That is the office line. I will give you {{p|tevita|misspelled|Tavita}}, our depot supervisor, his mobile is {{ph|07700 900388}}. T, E, V, I, T, A.
{{p|stavros|caps-first}}: {{p|tevita|first}}. Good. And the paper contract goes to our lawyer, Mister {{p|bertrand|title-last|Vasseur}} in {{l|Marseille}}, not to me.
{{p|jordan|caps-first}}: Understood. {{p|ngozi|first}}, can you send the draft to Maître {{p|bertrand|title-last|Vasseur}} by Friday?
UNKNOWN 1: Yes. Friday the twenty-seventh.
{{p|stavros|caps-first}}: Then we are agreed. Greet {{p|marguerite|first}} from me.`)).on("2026-03-19", "ludovica"),

	work("vendor-onboarding-form", "form,bank-details",
		usr(`Validate this supplier form before I key it into the ERP. Anything missing or inconsistent?

NEW SUPPLIER SET-UP FORM (FIN-07)
Requested by: {{p|reed}}            Date: {{d|2026-02-12|12-Feb-2026}}
Approved by: {{p|chad}}             Date: ____________

SUPPLIER
Legal name: {{o|Ten Brinke Koeltransport B.V.}}
Trading as: {{o|TBK Cold Chain}}
Registered address: {{a|Nijverheidsweg 31, 3771 ME Barneveld}}, Netherlands
KvK number: {{x|company-register-number|08142937}}
VAT: NL004829173B01

CONTACTS
Director: {{p|maarten}} (sole shareholder)
Accounts receivable: {{p|anouk}}, {{e|debiteuren@tbkcoldchain.example}}, {{ph|+31 342 555 013}}
Director direct: {{e|maarten@tbkcoldchain.example}} / {{ph|06 5550 0172}}

BANK
Account holder: {{p|maarten|initial-last|M. ten Brinke}} h/o {{o|TBK Cold Chain}}
IBAN: {{f|nl-iban|NL49 RABO 0318 4927 56}}
BIC: RABONL2U
Bank letter attached: yes (dated {{d|2026-01-30|30 januari 2026}})

IDENTITY CHECK (sole traders and director-owned companies only)
ID type: Dutch passport      Number: {{g|nl-passport|NXR7P41K2}}
BSN (for withholding declaration): {{g|nl-bsn|217849362}}
Date of birth: {{d|1971-09-04|04-09-1971}}

NOTES
Called {{p|maarten|title-last|ten Brinke}} to verify the IBAN on a number from their website, not from the form, per fraud policy. He confirmed the last four characters (2756). {{p|anouk|first}} asks that remittance advice goes to the debiteuren mailbox only. First PO: {{n|order-number|PO-4500078213}}.`)),

	work("teams-chat-typos", "chat,lowercase,typos",
		usr(`what is {{p|ingrid|lower-first}} actually asking me to do here?? i cant tell if im in trouble

[{{d|2026-05-22|22/05/2026}} 16:41] {{p|ingrid}}: hi {{p|brook|lower-first}} quick one
[{{d|2026-05-22|22/05/2026}} 16:41] {{p|ingrid}}: did u send the q2 headcount file to {{p|jurgen|ascii-fold|jurgen}} or to {{p|jordan|lower-first}}
[{{d|2026-05-22|22/05/2026}} 16:43] {{p|brook}}: to {{p|jurgen|ascii-fold|jurgen}} i think?? he asked for it tues
[{{d|2026-05-22|22/05/2026}} 16:43] {{p|brook}}: wait no. i sent it to {{e|j.weissgerber@corvanefreight.example}} and it bounced so i sent to {{e|jordan.mwangi-hale@corvanefreight.example}} bc autocomplete
[{{d|2026-05-22|22/05/2026}} 16:44] {{p|ingrid}}: ok so {{p|jordan|lower-first}} has salaries for the whole depot incl {{p|tevita|lower-first}}'s and {{p|obinna|misspelled|obina}}'s
[{{d|2026-05-22|22/05/2026}} 16:44] {{p|ingrid}}: not ideal
[{{d|2026-05-22|22/05/2026}} 16:45] {{p|brook}}: omg. im so sorry. shoud i ask him to delete
[{{d|2026-05-22|22/05/2026}} 16:46] {{p|ingrid}}: no i will. pls dont msg him. also the right address for {{p|jurgen|initials|jw}} is {{e|juergen.weissgerber@corvanefreight.example}} with the ue
[{{d|2026-05-22|22/05/2026}} 16:46] {{p|ingrid}}: can u write down what happend, 3 lines, times, send to me + {{p|rashida|lower-first}} today. its a data incident technically so {{p|rashida|misspelled|rashda}} has to log it
[{{d|2026-05-22|22/05/2026}} 16:47] {{p|brook}}: ok. doing it now
[{{d|2026-05-22|22/05/2026}} 16:47] {{p|ingrid}}: ur not in trouble. just be careful w autocomplete. theres a {{x|common-word|brook}} by the depot we can throw the laptop in if it happens again
[{{d|2026-05-22|22/05/2026}} 16:48] {{p|brook}}: 😭`)),

	work("company-newsletter", "newsletter,announcements",
		usr(`Shorten this newsletter to fit one screen. Keep all the names, people get upset when they are cut.

THE MANIFEST, {{o|Corvane Freight}} staff news, June issue

Welcomes. {{p|akosua}} joins the customs team in {{l|Rotterdam}} in August. {{p|khin|full|Khin Thida Soe}}, who most of you know as {{p|khin|honorific|Ma Thida}}, moves from the {{l|Yangon}} agency to our {{l|Singapore}} desk. {{p|wulan|first}} (yes, one name, please stop asking IT to add a surname) is now lead for the {{l|Surabaya}} lane.

Farewells. {{p|penelope}} left for {{l|Auckland}} in February; her out-of-hours playbook lives on. {{p|lucinda|nickname}} {{p|lucinda|last}} hands the Hungarian accounts to {{p|reed}} and becomes Head of Key Accounts.

Milestones. Twenty years for {{p|jurgen}}, who started as a driver in {{l|Duisburg}} in 2006 and has never once filed his mileage on time. Ten years for {{p|marguerite}}. Five for {{p|tevita}} and {{p|leilani}}, who also got married in March. Congratulations to the {{p|tevita|last}} / {{p|leilani|last}} household.

Baby news. {{p|ngozi}} and her husband {{p|oluwaseun|nickname}} welcomed a daughter, {{p|+Adaora|first}}, on {{d|2026-05-02|2 May}}. Yes, that {{p|oluwaseun|nickname}}, from {{o|Lagos Freightline}}. We are told there is no conflict of interest.

Sport. The five-a-side team lost 7-1 to {{o|Halden & Pryce|company-eponym}}. Top scorer (our only scorer): {{p|hemi}}. {{p|baraka}} ran the {{l|Bristol}} 10k in 38:12, a depot record. {{p|hunter}} would like it known he was hunting for parking and missed the start.

Quiz answer from last month: the unit named after {{n|public-figure|James Watt}} is the watt, and nobody got the {{n|eponym|Beaufort}} scale question right except {{p|dale|first}}.`)),

	work("payroll-bank-change", "payroll,bank-details",
		usr(`Is this request legit or does it look like a payroll diversion scam? Both emails below.

--- email 1 ---
From: {{p|jurgen}} <{{e|juergen.weissgerber@corvanefreight.example}}>
To: {{e|payroll@corvanefreight.example}}
Date: {{d|2026-05-18|Mo., 18. Mai 2026 07:55}}
Subject: Neue Bankverbindung

Hallo {{p|anouk|first}},

ich bin umgezogen und habe die Bank gewechselt. Bitte ab dem Juni-Lauf auf das neue Konto überweisen:

Kontoinhaber: {{p|jurgen}}
IBAN: {{f|de-iban|DE53 2105 0170 0012 3485 67}}
BIC: NOLADE21KIE

Alte IBAN zur Kontrolle: {{f|de-iban|DE80 7002 0270 0038 4915 02}}
Neue Anschrift: {{a|Kieler Straße 204, 24143 Kiel}}
Steuer-ID: {{g|de-steuerid|65291047637}}, Personalnummer {{n|identifier|D-00317}}

Danke und Gruß
{{p|jurgen|first}}

--- email 2 ---
From: {{p|jurgen|ascii-fold|Jurgen Weissgerber}} <{{e|j.weissgerber.corvane@freemail.example}}>
To: {{e|payroll@corvanefreight.example}}
Date: {{d|2026-05-19|Tue, 19 May 2026 22:14}}
Subject: URGENT change of bank details

Hello,
I need update my salary account before this month payment. New account: {{f|gb-iban|GB78 BARC 2004 1583 9026 47}}. Please confirm when done and do not call as I am travelling. Kind regards {{p|jurgen|ascii-fold|Jurgen}}`)).on("2026-05-18", "jurgen"),

	work("ooo-forward-chain", "email,forward",
		usr(`Who do I actually need to contact about the customs hold? I have followed three out-of-office messages and I am lost.

---------- Forwarded message ----------
From: {{p|siobhan}} <{{e|siobhan.nib@corvanefreight.example}}>
Subject: Automatic reply: Customs hold {{n|identifier|CVF-91102-IE}}

Go raibh maith agat. I am on leave until {{d|2026-07-20|20 July}}. For customs matters contact {{p|obinna}} ({{e|obinna.c@corvanefreight.example}}). For {{l|Dublin}} port, call {{p|saoirse}} on {{ph|+353 20 915 5078}}.
{{p|siobhan|native-script|Siobhán}}

---------- Forwarded message ----------
From: {{p|obinna}} <{{e|obinna.c@corvanefreight.example}}>
Subject: Automatic reply: FW: Customs hold {{n|identifier|CVF-91102-IE}}

I am at a conference in {{l|Lagos}} {{d|2026-07-06|6 to 10 July}} with limited email. Urgent: {{p|khin|first}} on the {{l|Singapore}} desk, or my manager {{p|marguerite}}. I will reply to everything by {{d|2026-07-13|Monday 13 July}}.
{{p|obinna|nickname}}

---------- Forwarded message ----------
From: {{p|marguerite}} <{{e|m.delacroix-benoit@corvanefreight.example}}>
Subject: Réponse automatique : TR: FW: Customs hold {{n|identifier|CVF-91102-IE}}

Bonjour, je suis absente jusqu'au {{d|2026-07-14|14 juillet}} inclus. En cas d'urgence, merci de contacter {{p|ingrid}} au {{ph|+47 55 55 50 19}}.
I am out of the office until {{d|2026-07-14|July 14}}. For urgent matters please contact {{p|ingrid}}.
{{p|marguerite|initial-last|M. Delacroix-Benoît}}

And the original, from the customer's broker:
From: {{p|saoirse|ascii-fold|Saoirse Nic Gearailt}} <{{e|saoirse@dublinportagents.example}}>
Date: {{d|2026-07-07|7 July 2026}}
"Revenue are holding the container pending an EORI correction. Somebody at {{o|Corvane|company}} needs to ring me today. {{p|saoirse|first}}."`)),
}

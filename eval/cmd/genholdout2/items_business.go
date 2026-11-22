package main

var itemsBusiness = []def{
	business("email-thread", "email,quoted-reply,signature", usr(`Draft a reply from me ({{p|ingrid|first}}) accepting the new dates but pushing back on the price increase. Polite, firm, short.

From: {{p|hamish}} <{{e|hamish.macgillivray@glenmore-logistics.example}}>
To: {{p|ingrid}} <{{e|ingrid.solbakken@polarfjord.example}}>
Cc: {{p|aino|first}} <{{e|aino.kaariainen@polarfjord.example}}>
Date: {{d|2026-03-05|Thu, 5 Mar 2026}} 16:42:10 +0000
Subject: RE: RE: Renewal terms 2026/27

{{p|ingrid|first}},

Thanks for turning this around. We can do the {{d|2026-04-01|1st of April}} start instead of {{d|2026-03-15|15 March}}, but the 6% uplift stands, I'm afraid. {{p|angharad|first}} in our finance team can talk you through the fuel index if useful.

Kind regards,
{{p|hamish|first}}

{{p|hamish}} | Key Accounts
{{o|Glenmore Logistics Ltd|company}}
{{a|Unit 14, Harbour Trade Park, Leith, Edinburgh EH6 7ZZ}}
T: {{ph|0131 496 0142}} | M: {{ph|07700 900814}}

> On {{d|2026-03-04|4 Mar 2026}}, at 09:17, {{p|ingrid}} wrote:
> Hi {{p|hamish|first}},
> We would like to move the start to April and keep the current rate card for year one.
> {{p|aino|first}} (cc) has the volumes. I am out on Friday; call {{p|aino|possessive}} mobile, {{ph|+358 40 555 0142}}, if it is urgent.
> Mvh
> {{p|ingrid|first}}
>
>> On {{d|2026-02-27|27 Feb 2026}}, at 14:03, {{p|hamish|initial-last|H. MacGillivray}} wrote:
>> Please find attached the renewal proposal. Prices valid until {{d|2026-03-31|31/03/2026}}.`)),

	business("meeting-notes", "notes,initials", usr(`Clean up my notes into minutes with decisions and action items (owner, due date).

pricing sync {{d|2026-03-03|3/3}}
there: {{p|destiny|first}}, {{p|kobby|nickname}}, {{p|shirin|first}}, {{p|lukas|first}} (late), {{p|yael|first}} dialed in
- {{p|destiny|initials|DA}}: tier 2 churn up 3 pts since jan. {{p|kobby|initials|KAB}} thinks its the annual-only change
- {{p|shirin|first}} wants to A/B monthly back in for DACH first. {{p|lukas|lower-first|lukas}} ok w/ that if legal signs off, he will ask {{p|hannelore|title-last|Dr. Grünwald}}
- {{p|yael|first}}: {{o|Mason & Reed Partners|company-eponym}} (the reseller, not our Mason) want net-60. no decision
- {{p|kobby|initials|KAB}} to send cohort sheet by fri; {{p|destiny|initials|DA}} to draft customer email by the 10th
- next sync {{d|2026-03-17|17 March}} 10:00 CET, {{p|shirin|possessive}} calendar
- parking lot: {{p|lukas|last-part|Trettenbach}} raised SOC 2 wording again`)),

	business("hr-pip", "hr,sensitive", usr(`Rewrite this performance improvement plan so it is specific and fair, and remove anything that reads as personal criticism.

PERFORMANCE IMPROVEMENT PLAN
Employee: {{p|cliff}} (ID 31187)          Manager: {{p|june}}
Department: Customer Operations               HR partner: {{p|nastya}}
Plan period: {{d|2026-03-16|March 16, 2026}} to {{d|2026-05-15|May 15, 2026}}

Background. Since {{d|2025-11-03|November 3}}, {{p|cliff|first}} has missed the first-response target in 9 of 17 weeks. On {{d|2026-01-22|January 22}} and again on {{d|2026-02-12|February 12}} a customer escalated directly to {{p|june|title-last|Ms. Oyelaran}}. {{p|cliff|first}} disclosed on {{d|2026-02-13|2/13}} that he is caring for a parent after surgery; accommodations were discussed with {{p|nastya|nickname}} ({{p|nastya|formal|A. S. Volkonskaya}}, HR) and are documented separately.

Expectations.
1. First response within 4 business hours on 90% of tickets.
2. Weekly 1:1 with {{p|june|first}} every Tuesday.
3. Shadow {{p|blessing}} for two shifts by {{d|2026-03-27|3/27}}.

Review dates: {{d|2026-04-14|April 14}}, {{d|2026-05-15|May 15}}.
Signed: ____________ ({{p|cliff|caps}})      ____________ ({{p|june|caps}})`)),

	business("offer-letter", "hr,address", usr(`Check this offer letter for anything missing under UK employment law and fix the tone. Keep all the facts.

{{o|Fennwick & Lowe LLP|company-eponym}}
{{a|22 Chancery Row, London WC2A 9ZZ}}

{{d|2026-03-02|2nd March 2026}}

PRIVATE & CONFIDENTIAL
{{p|precious|title-full|Ms Precious Ndlovu}}
{{a|Flat 3, 41 Abbeville Road
London SW4 9ZZ}}

Dear {{p|precious|first}},

We are delighted to offer you the position of Senior Associate, reporting to {{p|jack}}, starting on {{d|2026-05-05|Tuesday 5 May 2026}}. Your salary will be £92,000 per annum, paid monthly into the account you nominate. Please bring your passport and proof of your National Insurance number ({{g|nino|QQ 98 76 54 A}}, as given on your application) on your first day.

This offer is open until {{d|2026-03-16|16/03/2026}}. If you have questions, call me on {{ph|020 7946 0718}} or write to {{e|people@fennwick-lowe.example}}.

Yours sincerely,
{{p|pearl}}
Head of People`)),

	business("legal-demand", "legal,all-caps", usr(`Summarize this letter in plain English for the client and list the deadlines.

WITHOUT PREJUDICE SAVE AS TO COSTS

RE: {{p|earl|caps}} v. {{o|TALLOWFIELD HOLDINGS INC.|company}}; Claim No. 26-CV-00418

Dear Counsel:

This firm represents {{p|earl|caps}} ("MR. {{p|earl|caps-last|WHITCOMBE}}"), residing at {{a|1180 County Road 14, Bangor, ME 04401}}, in connection with the termination of his employment on {{d|2025-12-19|December 19, 2025}}. Mr. {{p|earl|last}} was employed for thirty-one years. Your client's HR Director, {{p|carol}}, informed him by telephone, without notice, that his position had been eliminated; his replacement, we understand, is twenty-six.

Demand is made for (i) back pay from {{d|2025-12-19|12/19/2025}}; (ii) reinstatement of pension contributions to plan account {{f|account|PL-0044-719203}}; and (iii) the sum of $240,000. Absent a response by {{d|2026-03-27|March 27, 2026}}, we are instructed to file.

Very truly yours,
{{p|shandiin}}, Esq.
{{o|Yazzie Castellano LLP|company-eponym}}
{{ph|(207) 555-0126}} | {{e|syazzie@yazzie-castellano.example}}
cc: {{p|reed|initial-last|R. Castellano}}`)),

	business("nda", "legal,gov-id", usr(`Is clause 1 enforceable as written? Suggest tighter wording.

MUTUAL NON-DISCLOSURE AGREEMENT

This Agreement is made on {{d|2026-03-09|the 9th day of March 2026}} between:

(1) {{o|KUPFERHAIN GMBH|company}}, a company registered in Germany under HRB 000000, whose registered office is at {{a|Kupferhainstraße 14, 50667 Köln}} (the "Company"), represented by its Geschäftsführer {{p|lukas}}; and

(2) {{p|sasha|caps}}, holder of Ukrainian passport no. {{g|passport|FA000142}}, born {{d|1994-10-28|28.10.1994}}, of {{a|Görlitzer Straße 14, 10997 Berlin}} (the "Consultant", also known professionally as {{p|sasha|nickname-last|Sasha Tkachenko}}).

1. Each party shall keep confidential all information disclosed by the other for a period of ninety-nine years.

Signed for the Company: {{p|lukas|initial-last|L. von Trettenbach}}
Signed by the Consultant: {{p|sasha|initial-last|O. Tkachenko}}
Witness: {{p|gosia}}, {{d|2026-03-09|09.03.2026}}`)),

	business("crm-note", "crm,lowercase,nickname", usr(`turn my call notes into a CRM entry with next steps

call w {{p|agus|lower|agustín ferreyra}} @ {{o|Ferreyra Hermanos SRL|company-eponym}}, {{d|2026-03-04|4/3}} 15:30 ART
- {{p|agus|lower-nickname|agus}} is the son, runs ops. dad ({{p|+Ernesto Ferreyra|title-full|Don Ernesto Ferreyra}}) still signs everything
- they have 14 trucks, want telematics on all by winter
- budget ok but needs to be invoiced in pesos, CUIT {{g|cuit|30-00000000-7}}
- send quote to {{e|agustin@ferreyrahnos.example}} cc {{e|administracion@ferreyrahnos.example}}
- his cell {{ph|+54 9 11 5550-0142}}, whatsapp only, never before 10
- {{p|chema|nickname}} from our mexico office knows him from a trade show, loop him in
- follow up next wednesday`)),

	business("invoice-de", "invoice,german,financial", usr(`Extrahiere Rechnungsnummer, Betrag, Fälligkeit und Bankverbindung als JSON.

{{o|Schönberger Haustechnik GmbH|company-eponym}}
{{a|Lindenallee 14, 04109 Leipzig}}
Tel. {{ph|030 23125 118}} · {{e|buchhaltung@schoenberger-haustechnik.example}}

Rechnung Nr. 2026-0318                                   Leipzig, den {{d|2026-03-04|4. März 2026}}

Herrn
{{p|bjorn}}
{{a|Auenstraße 7
04105 Leipzig}}

Leistungsdatum: {{d|2026-02-26|26.02.2026}}
Pos. 1  Wartung Gastherme                     1 x 189,00 EUR
Pos. 2  Anfahrt                               1 x  35,00 EUR
Zwischensumme 224,00 EUR, zzgl. 19 % USt 42,56 EUR, Gesamt 266,56 EUR

Zahlbar ohne Abzug bis zum {{d|2026-03-18|18.03.2026}} auf folgendes Konto:
IBAN {{f|iban|DE02 1203 0000 0000 2020 51}}, BIC {{f|swift|BYLADEM1001}}
Steuernummer {{g|tax-id|231/000/00142}}

Es bediente Sie: {{p|+Uwe Schönberger|title-full|Herr Uwe Schönberger}}. Bei Rückfragen wenden Sie sich bitte an Frau {{p|hannelore|last}}.`)),

	business("support-chat", "support,transcript,financial", usr(`Score this chat against our QA rubric (greeting, verification, resolution, tone) and point out any policy breach.

[14:02:11] {{p|malou|nickname}} (agent): Thank you for contacting {{o|Orchardgate|company}} support, this is {{p|malou|nickname}}. Who am I speaking with?
[14:02:40] Visitor 88213: {{p|kainoa|lower|kainoa kahananui}}
[14:02:58] {{p|malou|nickname}} (agent): Thanks {{p|kainoa|first}}! Can you confirm the email and billing zip on the account?
[14:03:31] Visitor 88213: {{e|kainoa.k@example.com}} and 96712. my card got charged twice
[14:04:02] {{p|malou|nickname}} (agent): I see two charges on {{d|2026-03-01|March 1}}. Can you confirm the last four?
[14:04:20] Visitor 88213: its {{f|card-visa|4000 0566 5566 5556}} exp 08/28 cvv {{s|cvv|314}}
[14:04:26] {{p|malou|nickname}} (agent): Oh, please do not send the full number here! I only need the last four.
[14:05:10] Visitor 88213: sorry. 5556. address is {{a|59-014 Kamehameha Hwy, Haleiwa, HI 96712}} if you need it
[14:06:45] {{p|malou|nickname}} (agent): Refund issued, 5 to 7 business days. My supervisor {{p|jun|first}} will email a confirmation. Anything else, Mr. {{p|kainoa|last}}?
[14:07:02] Visitor 88213: no thats it mahalo {{p|malou|lower-nickname|malou}}
[14:07:05] System: chat ended. Transcript sent to {{e|kainoa.k@example.com}}.`)),

	business("angry-customer", "support,all-caps,financial", usr(`Write a calm reply. Do not repeat any of his account details back to him.

Subject: THIRD TIME ASKING

I HAVE NOW WRITTEN THREE TIMES. MY NAME IS {{p|rusty|caps-nickname-last|RUSTY KOWALEWSKI}} ({{p|rusty|caps}} ON THE ACCOUNT). ACCOUNT NUMBER {{f|account|4471-0092-118}}. YOU TOOK $89.99 ON {{d|2026-02-03|FEB 3}} AND AGAIN ON {{d|2026-03-03|MARCH 3RD}} FOR A PLAN I CANCELLED IN JANUARY. YOUR AGENT "{{p|destiny|caps-first|DESTINY}}" PROMISED A CALL BACK ON {{ph|406-555-0177}} AND NOBODY CALLED.

REFUND IT TO THE CARD ENDING 0005 ({{f|card-amex|3782 822463 10005}}) OR I DISPUTE IT WITH MY BANK.

{{p|rusty|first}} J. {{p|rusty|last}}
{{a|2210 Rimrock Road, Billings, MT 59102}}
Sent from my phone`)),

	business("sales-email-es", "email,spanish", usr(`Corrige la ortografía y hazlo un poco más formal, sin cambiar los datos.

Asunto: Propuesta para {{o|Gutiérrez Olvera y Asociados|company-eponym}}

Estimado Lic. {{p|chema|last}}:

Fue un gusto conversar con usted y con la Ing. {{p|ximena|first-surname|Ximena Hernández}} el pasado {{d|2026-02-26|jueves 26 de febrero}}. Como quedamos, le envio la propuesta para las 40 licencias. El precio es válido hasta el {{d|2026-03-31|31 de marzo de 2026}}.

Para la factura necesitamos su RFC ({{p|ximena|nickname}} me dijo por teléfono que es {{g|rfc|XAXX010101000}}, por favor confirme) y el domicilio fiscal: {{a|Av. Insurgentes Sur 1400, Piso 14, Col. Del Valle, 03100 Ciudad de México}}.

Quedo a sus órdenes en el {{ph|+52 55 5550 0188}} o en {{e|ventas@tidepool-labs.example}}.

Saludos cordiales,
{{p|montse}}
Ejecutiva de cuenta`)),

	business("candidate-feedback", "recruiting,hr", usr(`Combine these interview scorecards into one hiring recommendation. Flag anything that is not job related.

Candidate: {{p|tahmina}} (Staff Data Engineer), referred by {{p|zainab|first}}
Profile: https://profiles.example.com/in/{{p|tahmina|url-slug|tahmina-chowdhury-data}}  |  {{e|tahmina.chowdhury@example.com}}  |  {{ph|+880 1700 000142}}

{{p|dilshan}} (system design), {{d|2026-03-02|2 Mar}}: Strong hire. {{p|tahmina|first}} drove the whole conversation. Best candidate since {{p|arjun|first}}.
{{p|priya|initial-last|P. Venkataraman}} (coding), {{d|2026-03-02|2 Mar}}: Hire. Clean solution in 25 minutes. She mentioned she would need to relocate from {{l|Dhaka}} and that her husband is also looking; not sure that matters.
{{p|harpreet|nickname}} (values), {{d|2026-03-03|3 Mar}}: Lean hire. {{q|the only woman on the platform team at her current company}}, which she talked about thoughtfully. Asked about visa sponsorship.
{{p|arjun|initials-last|A. R. Iyer}} (hiring manager), {{d|2026-03-04|4 Mar}}: Hire at Staff. Comp expectation 1.4 crore. References: her former director {{p|+Farhana Rahman}} ({{e|farhana.rahman@example.org}}).`)),

	business("expense-claim", "finance,financial", usr(`Does this claim comply with the policy below it? Itemize what gets rejected.

EXPENSE CLAIM EC-2026-0417
Claimant: {{p|wanjiru}} ({{p|wanjiru|nickname}}), Nairobi office    Approver: {{p|sbu}}
Trip: Nairobi to Johannesburg, {{d|2026-02-16|16 Feb}} to {{d|2026-02-19|19 Feb 2026}}

{{d|2026-02-16|16/02}}  Flight NBO-JNB, booking ref QX7R2M               KES 61,400
{{d|2026-02-16|16/02}}  Taxi, airport to hotel                            ZAR 480
{{d|2026-02-17|17/02}}  Dinner with client ({{p|annelize|first}} and {{p|sbu|nickname}}), 3 people  ZAR 2,150
{{d|2026-02-18|18/02}}  Hotel, 3 nights                                   ZAR 6,900
{{d|2026-02-19|19/02}}  Minibar                                           ZAR 310

Paid with corporate card {{f|card-mastercard|5200 8282 8282 8210}} except the taxi (cash).
Reimburse cash items to M-Pesa {{ph|+254 700 555 077}} or bank: {{o|Savannah Commercial Bank|company}}, acct {{f|account|0110-0042-7719}}, SWIFT {{f|swift|SVCBKENA}}.

POLICY (excerpt): Meals with clients up to ZAR 600 per head. Alcohol and minibar are not reimbursable. Claims must be filed within 30 days.`)),

	business("postmortem", "incident,timeline", usr(`Write the customer-facing summary of this incident. No employee names in the public version.

INC-2217 Checkout outage, {{d|2026-03-07|Saturday 7 March 2026}}
Incident commander: {{p|caglar}}    Comms: {{p|ria}}    Scribe: {{p|stefan|first}}

Timeline (UTC)
21:02  Alert: checkout p99 above 8 s. Paged {{p|caglar|ascii-fold|Caglar Yildirim}} ({{ph|+90 532 555 01 42}}).
21:09  {{p|caglar|first}} acks from his phone, opens bridge. {{p|zsofia}} joins.
21:20  {{p|zsofia|first}} finds the connection pool pinned at 200 after deploy {{n|hash|7c2e91f}} by {{p|ria|handle|epapadimitriou}}.
21:31  Rollback started. {{p|stefan|first}} notes the status page still says green; {{p|ria|nickname}} updates it.
21:44  Recovered. 1,912 failed orders.
22:30  {{p|caglar|last}} hands over to {{p|mido}} for the night.

Follow-ups
- {{p|zsofia|last-first|Szabó, Zs.}}: pool limit alarm, due {{d|2026-03-13|13 March}}
- {{p|ria|nickname}}: refund script, coordinate with {{p|mido|nickname}}, due {{d|2026-03-10|Tue 10 Mar}}
- Largest affected customer: {{o|Abara Foods|company-eponym}}; account manager is {{p|destiny|first}}'s brother-in-law, so route through {{p|stefan|last}} instead.`)),

	business("reference-check", "hr,quasi-identifier", usr(`Summarize this reference call in three bullet points for the hiring committee, anonymized.

Reference for {{p|gudrun}}, taken {{d|2026-02-24|24. febrúar 2026}} by {{p|soren}}.

Referee: her former head of department at the university hospital in {{l|Akureyri}}, who asked not to be named in writing. Reached on {{ph|+354 555 0142}}.

Notes: "{{p|gudrun|first}} was {{q|the only paediatric cardiologist north of Reykjavík for six years}}. Everybody in town knows who that is, so please be careful how you write this up." Left in {{d|2024-08-31|August 31, 2024}} when her husband, {{q|the harbour master in Akureyri}}, took a post abroad. No concerns. Would rehire. She prefers email: {{e|gudrun.sig@example.org}}.`)),

	business("slack-export", "chat,handles,lowercase,typo", usr(`Summarize the decision from this channel export and who is blocked on whom.

#proj-migration  {{d|2026-03-05|2026-03-05}}
09:14 <{{p|jiwoo|handle|jiwoo.seo}}> morning. @{{p|meilin|handle|mei}} is the cutover still sat the 14th?
09:15 <{{p|meilin|handle|mei}}> yes unless {{p|kaho|lower-nickname|felix}} finds something in the dry run
09:15 <{{p|meilin|handle|mei}}> @{{p|kaho|handle|felix.tsang}} ??
09:31 <{{p|kaho|handle|felix.tsang}}> dry run ok. one thing, the vendor contact ({{p|akane|title-last|Mizushima-san}}) is out till the 12th
09:32 <{{p|jiwoo|handle|jiwoo.seo}}> ugh. does {{p|haruto|lower-first|haruto}} have her backup's number
09:40 <{{p|haruto|handle|hkobayashi}}> backup is {{p|+Tanigawa|title-last|Ms. Tanigawa}}, {{ph|+81 6-0000-0177}}. dont call before 10 JST
09:41 <{{p|meilin|handle|mei}}> thx {{p|haruto|title-first|Haruto-kun}}
09:58 <{{p|jiwoo|handle|jiwoo.seo}}> ok so plan stands. {{p|jiwoo|misspelled|Ji-wo}} out (typing my own name wrong, great)
10:02 <{{p|kaho|handle|felix.tsang}}> lol. also pls stop putting the db pw in here, i rotated it. new one is in the vault under migration/{{n|identifier|felix-tmp}}
10:03 <{{p|meilin|handle|mei}}> it was {{s|password|Lantern-Festival-0214}} for anyone grepping later, now dead`)),

	business("board-minutes", "minutes,titles", usr(`Format as formal minutes. Use surnames with titles throughout.

Minutes of the Board of Trustees, {{o|Marlowe College|company}}, held {{d|2026-02-19|Thursday the nineteenth of February, 2026}}.

Present: {{p|khalid|title-full|Dr. Khalid Al-Tamimi}} (Chair), {{p|violet|title-full|Prof. Violet Okonkwo-Hale}}, {{p|basil|title-full|Mr. Basil Thorncroft}}, {{p|carol|title-full|Mrs. Carol Ann Whitehorse}}, {{p|kobby|title-full|Rev. Kwabena Asante-Boateng}}. Apologies: {{p|sage|title-full|Mx. Sage Delacroix}}. In attendance: {{p|blessing}} (Registrar), minute taker.

1. The Chair welcomed {{p|kobby|title-last|Rev. Asante-Boateng}} to his first meeting.
2. Minutes of {{d|2025-11-20|20 November 2025}} approved, proposed {{p|violet|title-last|Prof. Okonkwo-Hale}}, seconded {{p|basil|title-last|Mr. Thorncroft}}.
3. Bursar's report. {{p|carol|title-last|Mrs. Whitehorse}} asked why the {{o|Thorncroft Bequest|fund-eponym}} had underperformed. {{p|basil|title-last|Mr. Thorncroft}} declared an interest (the bequest was his late aunt's) and left the room.
4. The Registrar reported one student complaint, from {{p|+Emeka Okpara}}, a second-year; {{p|blessing|title-last|Ms. Okpara}} noted he is no relation.
5. Next meeting: {{d|2026-05-21|21 May 2026}}.`)),

	business("vendor-form", "finance,gov-id,form", usr(`Validate this vendor form: which fields look malformed? Reply as a table.

NEW VENDOR SETUP
Legal name: {{p|angel}} dba {{o|Batista Electric|company-eponym}}
Tax classification: Sole proprietor
EIN or SSN: {{g|ssn|987-65-4329}}
Address: {{a|Calle Loíza 1402, San Juan, PR 00911}}
Remit-to contact: {{p|angel|first}} / {{e|angel@batista-electric.example}} / {{ph|787-555-0164}}
Bank name: {{o|First Example Credit Union|company}}
ABA routing: {{f|routing|110000000}}
Account: {{f|account|000444555666}}
W-9 signed: {{d|2026-02-28|2/28/26}}
Requested by: {{p|hunter}} ({{p|hunter|handle|hashby}}), Facilities
Approved by: {{p|april|initial-last|A. Nakashima}}, {{d|2026-03-02|3/2/26}}`)),

	business("termination-fr", "hr,french,address", usr(`Relis cette lettre et dis-moi si la procédure est respectée. Corrige les fautes.

{{o|Atelier Nord SAS|company}}
{{a|14 rue des Tanneurs, 59000 Lille}}

Lettre recommandée avec AR
{{p|elodie|title-full|Madame Élodie de La Rochebrune}}
{{a|8 allée des Tilleuls
59800 Lille}}

Lille, le {{d|2026-03-06|6 mars 2026}}

Objet : notification de licenciement

Madame,

À la suite de notre entretien du {{d|2026-02-24|mardi 24 février}}, au cours duquel vous étiez assistée de M. {{p|jb}}, délégué du personnel, nous vous informons de notre décision de vous licencier pour motif économique. Votre préavis de deux mois débutera à la première présentation de cette lettre et prendra fin le {{d|2026-05-08|8 mai 2026}}.

Votre solde de tout compte sera versé sur le compte {{f|iban|FR76 3000 6000 0112 3456 7890 189}}. Pour toute question, contactez Mme {{p|marieeve|last}} au {{ph|01 99 00 41 42}}.

Veuillez agréer, Madame, l'expression de nos salutations distinguées.

{{p|+Gaspard Vanderbeken}}
Directeur général`)),

	business("claim-note", "insurance,gov-id", usr(`Rewrite the adjuster's notes as a clean claim summary with a liability assessment.

CLAIM 26-VA-0091773   Policy {{f|policy|HO-4410-229871}}   DOL {{d|2026-02-08|02/08/2026}}
Insured: {{p|mason}}, DOB {{d|1990-03-22|03/22/1990}}, DL {{g|drivers-license|T000-1422-9900}} (VA)
Claimant: {{p|+Lucille Dubois}} (insured's mother, passenger), {{p|+Lucille Dubois|first|Lucille}} is 71

{{d|2026-02-09|2/9}} spoke w/ insd by phone {{ph|(757) 555-0138}}. Says other veh ran the light at Granby & 21st. Other driver: {{p|wade}}, plate unknown, gave a number that does not work ({{ph|757-555-01}}, incomplete).
{{d|2026-02-11|2/11}} recorded stmt from {{p|+Lucille Dubois|title-last|Mrs. Dubois}}. Neck pain, seen at urgent care {{d|2026-02-08|2/8}}.
{{d|2026-02-17|2/17}} located OV driver via police report. {{p|wade|last-first|HARCOURT, WADE T}}, {{a|3318 Colley Avenue, Norfolk, VA 23508}}. Carrier unknown. He says "the {{o|Mason Creek|place}} bridge light was out" which is a different intersection entirely.
{{d|2026-02-20|2/20}} rental extended thru {{d|2026-03-01|3/1}}. Email estimates to {{e|mason.dubois@example.com}}.`)),

	business("lease-summary", "legal,address", usr(`Pull the key terms from this lease into a one-page summary: parties, premises, term, rent, deposit, notice.

RESIDENTIAL TENANCY AGREEMENT dated {{d|2026-02-27|27 February 2026}}

LANDLORD: {{p|wiremu}} and {{p|+Mereana Te Whaiti}}, of {{a|14 Kowhai Crescent, Havelock North 4130}}
TENANT: {{p|aisyah}} (passport {{g|passport|A00014277}}, Malaysia)
PREMISES: {{a|Flat 2, 88 Tory Street, Te Aro, Wellington 6011}}

The tenancy begins on {{d|2026-03-14|14 March 2026}} for a fixed term ending {{d|2027-03-13|13 March 2027}}. Rent is $720 per week, payable in advance to account {{f|account|00-0000-0000188-00}} in the name of {{p|wiremu|initials-last|W & M Te Whaiti}}. Bond of $2,880 is to be lodged with Tenancy Services.

The Landlord's agent is {{o|Paraone Property Management|company-eponym}} ({{ph|+64 4 555 0142}}, {{e|rentals@paraone-property.example}}). Notices to the Tenant may be sent to {{e|nurul.aisyah@example.org}}.

SIGNED {{p|wiremu|initial-last|W. Te Whaiti}}    SIGNED {{p|aisyah|formal|N. A. binti Abdullah}}`)),

	business("press-release", "pr,quotes", usr(`Tighten this press release to 150 words and write three headline options.

FOR IMMEDIATE RELEASE

{{o|Steppeline|company}} appoints {{p|aigerim}} as Chief Technology Officer

{{l|Almaty}}, {{d|2026-03-10|10 March 2026}}. {{o|Steppeline|company}}, the logistics platform, today announced that {{p|aigerim}} will join as CTO on {{d|2026-04-01|1 April}}. {{p|aigerim|title-last|Ms. Zhaksybekova}} was previously VP Engineering at {{o|Tidepool Labs|company}} and holds a doctorate from {{o|ETH Zürich|company}}.

"{{p|aigerim|first}} built the team that everyone in the region tries to hire from," said founder and CEO {{p|enkhtuya}}. "We have wanted to work with her since we met in {{l|Ulaanbaatar}} ten years ago."

"{{p|enkhtuya|nickname}} and I have argued about routing algorithms for a decade," said {{p|aigerim|last}}. "Now we get to do it in the same building."

Media contact: {{p|nadine}}, {{e|press@steppeline.example}}, {{ph|+7 727 000 01 42}}`)),

	business("scheduling", "email,dates,conference", usr(`Propose three slots that work for everyone and draft the invite text.

{{p|thuy|ascii-first|Thuy}} ({{l|Hanoi}}): free {{d|2026-03-16|16/3}} and {{d|2026-03-17|17/3}} after 14:00 local, not {{d|2026-03-18|18/3}}.
{{p|joao|nickname}} ({{l|Recife}}): "qualquer dia menos {{d|2026-03-17|terça, 17/03}}; prefiro de manhã."
{{p|seun|nickname}} ({{l|Lagos}}): available Mon 16th to Thu 19th, 09:00 to 17:00 WAT.
{{p|tomasz|nickname}} ({{l|Gdańsk}}): {{d|2026-03-18|18.03.}} or {{d|2026-03-19|19.03.}}, {{bt}}po 13:00{{bt}}.
{{p|ngozi|first}} ({{l|London}}): back from leave on {{d|2026-03-18|Wednesday, March 18th}}.

Bridge: https://meet.harborline.example/j/88213004417, passcode {{s|pin|447193}}. Dial-in {{ph|+44 20 7946 0555}}, host PIN {{s|pin|90214#}} (host is {{p|ngozi}}).`)),

	business("qbr-pt", "notes,portuguese", usr(`Resuma estas notas da QBR em cinco tópicos e traduza para inglês.

QBR {{o|Carvalho & Cavalcanti Advogados|company-eponym}}, {{d|2026-03-03|3 de março de 2026}}
Presentes: Dra. {{p|ines}} (sócia), {{p|rafa|nickname}} (TI), do nosso lado eu e o {{p|joao|first}}.

- A Dra. {{p|ines|first}} reclamou do tempo de resposta do suporte em fevereiro. O chamado do dia {{d|2026-02-12|12/02}} ficou 3 dias parado.
- {{p|rafa|first}} quer SSO até {{d|2026-04-30|30 de abril}}. Disse que o {{p|+Thiago Nascimento|title-full|Dr. Thiago Nascimento}} (sócio fundador) não usa o sistema e pede tudo por telefone: {{ph|+55 81 0000-0142}}.
- Renovação: contrato vence em {{d|2026-06-30|30/06/2026}}. CNPJ para a proposta: {{g|cnpj|00.000.000/0001-91}}.
- Próxima reunião: primeira terça de junho.
- Enviar ata para {{e|ines@carvalhocavalcanti.example}}.`)),

	business("dunning-nl", "finance,dutch", usr(`Maak hier een vriendelijke maar duidelijke tweede herinnering van.

Betreft: factuur 2026-0091, vervallen op {{d|2026-02-14|14 februari 2026}}

Geachte heer {{p|maarten|last|Van den Heuvel}},

Op {{d|2026-01-15|15-01-2026}} stuurden wij u factuur 2026-0091 ten bedrage van € 1.482,25. Volgens onze administratie is deze nog niet voldaan. Wij verzoeken u het bedrag vóór {{d|2026-03-20|20 maart a.s.}} over te maken op {{f|iban|NL91 ABNA 0417 1643 00}} t.n.v. {{o|Achterberg Installatietechniek B.V.|company-eponym}} onder vermelding van het factuurnummer.

Heeft u al betaald, dan kunt u deze herinnering als niet verzonden beschouwen. Vragen? Bel {{p|robin}} op {{ph|+31 20 555 0142}} of mail naar {{e|debiteuren@achterberg-installatie.example}}.

Met vriendelijke groet,
{{p|robin|initial-last|R. Achterberg}}`)),

	business("translate-ja", "email,honorifics,romanized", usr(`Translate to natural English for my manager, keep the names as written.

Kobayashi desu. Itsumo osewa ni natte orimasu.

{{d|2026-03-12|3-gatsu 12-nichi}} no kaigi no ken desu ga, {{p|akane|title-last|Mizushima-buchou}} ga shucchou no tame, {{d|2026-03-19|3-gatsu 19-nichi (moku)}} ni henkou sasete itadakemasen deshou ka. {{p|+Tanigawa|title-last|Tanigawa-san}} to {{p|meilin|title-last|Ouyang-sama}} ni wa watashi kara renraku shimasu.

Nao, seikyuusho wa {{p|christian|title-last|Lindgren-Ochoa-sama}} ate ni {{d|2026-03-02|3/2}} ni yuusou shimashita. Todoite inai baai wa {{ph|03-0000-0155}} made go-renraku kudasai.

Yoroshiku onegai itashimasu.
{{p|haruto}}
{{e|h.kobayashi@example.com}}`)),

	business("voicemail", "transcript,spoken-digits", usr(`Clean up this voicemail transcription and extract a callback task.

"Hi, uh, this message is for {{p|paige|first}}, {{p|paige|spelled|Paige with an I, Lindqvist}}. This is {{p|lance}} over at {{o|Corbin Roofing|company-eponym}}. I'm calling about the estimate we sent on the, uh, twenty-seventh. My guy {{p|rowan|first}} says he can start the week of {{d|2026-03-23|March twenty-third}} if we get the deposit. You can reach me at {{ph|four one five, five five five, zero one three two|spoken}}, again that's {{ph|415-555-0132}}, or my cell which is {{ph|six two eight, five five five, oh one nine nine|spoken}}. Email's {{e|lance@corbinroofing.example|plain}}, that's lance at corbin roofing dot example. Thanks, bye."

Received {{d|2026-03-03|Mar 3}} 4:51 PM, duration 0:47.`)),

	business("eponymous-companies", "negative,companies", usr(`For a brand strategy deck: which of these founder-named companies still benefit from the founder association and which have outgrown it? One line each, then a 2x2.

{{o|Siemens|company-eponym}}, {{o|Bosch|company-eponym}}, {{o|Philips|company-eponym}}, {{o|Ford|company-eponym}}, {{o|Honda|company-eponym}}, {{o|Tata|company-eponym}}, {{o|Dell|company-eponym}}, {{o|Hewlett-Packard|company-eponym}}, {{o|Goldman Sachs|company-eponym}}, {{o|Morgan Stanley|company-eponym}}, {{o|Wells Fargo|company-eponym}}, {{o|Marks & Spencer|company-eponym}}, {{o|Procter & Gamble|company-eponym}}, {{o|Johnson & Johnson|company-eponym}}, {{o|Ben & Jerry's|company-eponym}}, {{o|Levi Strauss|company-eponym}}, {{o|Harley-Davidson|company-eponym}}, {{o|Rolls-Royce|company-eponym}}, {{o|Mercedes-Benz|company-eponym}}, {{o|Tiffany & Co.|company-eponym}}, {{o|Wendy's|company-eponym}}, {{o|Dr. Martens|company-eponym}}, {{o|Abercrombie & Fitch|company-eponym}}.

Treat {{o|Chanel|company-eponym}} and {{o|Hermès|company-eponym}} as a separate luxury group. Our own client is a family firm whose name we will add later, so leave a placeholder row.`)),

	business("policy-template", "negative,placeholders", usr(`Turn this into a reusable template with consistent placeholders and a short how-to-fill note.

REMOTE WORK AGREEMENT

This agreement is between {{n|placeholder|[EMPLOYEE NAME]}} ("Employee") and {{n|placeholder|[COMPANY LEGAL NAME]}} ("Company"), effective {{n|placeholder|[START DATE]}}.

1. Location. Employee will work from {{n|placeholder|[HOME ADDRESS]}}. Changes need written approval from {{n|placeholder|<manager name>}}.
2. Hours. Core hours are 10:00 to 15:00 in the Employee's time zone, Monday to Friday.
3. Equipment. Company provides a laptop (asset tag {{n|placeholder|{asset_tag}}}). Stipend of {{n|placeholder|$[AMOUNT]}} per month.
4. Contact. Employee can be reached at {{n|placeholder|(XXX) XXX-XXXX}} and {{n|placeholder|firstname.lastname@company.tld}}.
5. Review. This arrangement is reviewed every {{n|placeholder|[N]}} months, first on {{n|placeholder|YYYY-MM-DD}}.

Signature: ____________________   Date: {{n|placeholder|__/__/____}}`)),

	business("place-names", "negative,places", usr(`Rank these for our next three regional sales offices and explain the trade-offs: {{l|Victoria}} and {{l|Hamilton}} in Canada; {{l|Adelaide}}, {{l|Darwin}} and {{l|Alice Springs}} in Australia; {{l|Wellington}} and {{l|Nelson}} in New Zealand; {{l|Santiago}}, {{l|Concepción}} and {{l|Valparaíso}} in Chile; {{l|San José}} in Costa Rica; {{l|Lincoln}}, {{l|Madison}}, {{l|Eugene}} and {{l|Orlando}} in the US; {{l|Alexandria}} in Egypt; {{l|Sofia}} in Bulgaria; {{l|Kingston}} in Jamaica; {{l|Salvador}} and {{l|Vitória}} in Brazil.

We already have people in {{l|Geneva}}, {{l|Phoenix}} and {{l|Mercedes}} (the one in Uruguay). Head office wants at least one office named after a saint, which I assume is a joke.`)),
}

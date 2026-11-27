package main

var itemsWorkA = []def{
	work("meeting-notes-initials", "meeting,initials",
		usr(`Clean these up into minutes with an action table (owner, action, due). Expand the initials to full names in the table.

OPS WEEKLY, {{o|Corvane Freight}}, {{d|2026-02-03|Tue 3 Feb 2026}}, {{l|Bristol}} + call
Present: {{p|marguerite}} ({{p|marguerite|initials|MDB}}, chair), {{p|obinna}} ({{p|obinna|initials|OC}}), {{p|siobhan}} ({{p|siobhan|initials|SNB}}), {{p|tevita}} ({{p|tevita|initials|TFL}}), {{p|ingrid}} ({{p|ingrid|initials|IAT}}, from {{l|Bergen}}), {{p|rashida}} ({{p|rashida|initials|RB}}, notes)
Apologies: {{p|jurgen}} (on site in {{l|Duisburg}}), {{p|penelope|nickname}}

1. Customs backlog
{{p|obinna|initials|OC}}: 41 consignments stuck at {{l|Felixstowe}}, mostly paperwork from the {{o|Halden & Pryce|company-eponym}} account. {{p|siobhan|initials|SNB}} thinks half are ours to fix. {{p|marguerite|initials|MDB}} wants a number by Thursday.
- {{p|obinna|initials|OC}} to split the list ours/theirs by {{d|2026-02-05|5 Feb}}
- {{p|siobhan|initials|SNB}} to call {{p|halvard|title-last|Brekke}} at {{o|Halden & Pryce|company-eponym}} (their counsel, {{ph|+47 22 55 50 14}}) re the power of attorney

2. Depot rota
{{p|tevita|nickname}} proposed moving the Sunday shift to a 4-on-4-off pattern. {{p|ingrid}} said {{l|Bergen}} tried it and lost two drivers. {{p|rashida|initials|RB}} to pull the absence numbers. {{p|marguerite|nickname}} is not convinced. Decision deferred to {{d|2026-02-17|17 Feb}}.
- {{p|rashida|initials|RB}}: absence report, both depots, by {{d|2026-02-10|10/02}}
- {{p|tevita|initials|TFL}} + {{p|ingrid|initials|IAT}}: one-page comparison

3. AOB
{{p|jurgen|first}} dialled in from the van for two minutes: the {{l|Duisburg}} client signed. {{p|penelope|possessive}}'s leaving do is Friday, {{p|siobhan|nickname}} is collecting. Next meeting {{d|2026-02-10|Tue 10 Feb}}, {{p|obinna|nickname}} to chair as {{p|marguerite|initials|MDB}} is out.`)).on("2026-02-05", "obinna", "2026-02-10", "rashida"),

	work("email-thread-quoted", "email,quoted-replies,signatures",
		usr(`What did we actually agree with the customer here, and is there anything I still owe them? I was added halfway through.

From: {{p|lucinda}} <{{e|lucinda.featherstonehaugh@corvanefreight.example}}>
To: {{p|zoltan}} <{{e|zoli@dunaspedicio.example}}>
Cc: {{p|reed}} <{{e|reed.calloway@corvanefreight.example}}>; {{p|mateus}} <{{e|mateus.anjos@corvanefreight.example}}>
Date: {{d|2026-04-16|Thu, 16 Apr 2026 09:12:40 +0100}}
Subject: RE: RE: Damaged pallets, consignment {{n|identifier|CVF-90417-HU}}

Dear {{p|zoltan|first}},

Agreed. We will credit {{n|money|EUR 2,340.00}} against your May invoice and {{p|mateus|first}} will arrange collection of the two pallets from {{a|Váci út 152, 1138 Budapest}} on {{d|2026-04-21|Tuesday 21 April}}. I have copied {{p|reed|first}}, who takes over the account from me on {{d|2026-05-01|1 May}}.

Kind regards,
{{p|lucinda|nickname}}

{{p|lucinda}}
Senior Account Manager, {{o|Corvane Freight Ltd}}
{{a|Unit 14, Tidewater Business Park, 220 Quayside Road, Bristol BS1 9ZZ}}
T {{ph|0117 496 0182}} | M {{ph|07700 900417}}

> On {{d|2026-04-15|15 Apr 2026}}, at 17:48, {{p|zoltan|family-first|Kerekes Zoltán}} <{{e|zoli@dunaspedicio.example}}> wrote:
>
> Dear Ms {{p|lucinda|title-last|Featherstonehaugh}},
>
> Thank you. EUR 2,340 is acceptable if the pallets are collected this month. My warehouse manager, {{p|zsofia}}, will be there, her mobile is {{ph|+36 30 555 0192}}. Please do not send the driver before 8.
>
> Üdvözlettel,
> {{p|zoltan|family-first|Kerekes Zoltán}}
> ügyvezető, {{o|Duna Spedíció Kft.}}
>
>> On {{d|2026-04-14|14 Apr 2026}}, at 11:05, {{p|lucinda}} wrote:
>>
>> Dear Mr {{p|zoltan|title-last|Kerekes}},
>>
>> I am sorry about the damage. Our insurer will cover up to EUR 2,000; I can add EUR 340 as goodwill. Would that close the matter? Our claims reference is {{n|identifier|CLM-2026-0388}}.
>>
>> {{p|lucinda|nickname}}
>>
>>> On {{d|2026-04-10|10 Apr 2026}}, {{p|zoltan|nickname}} wrote:
>>> Two of six pallets arrived crushed. Photos attached. This is the second time, {{p|lucinda|first}}. I expect a proposal by Tuesday.`)).on("2026-04-21", "mateus", "2026-05-01", "reed"),

	work("slack-transcript", "chat,speaker-labels",
		usr(`summarise this thread for someone who was on holiday. who decided what

#sales-emea  {{d|2026-03-12|Thursday, March 12th}}

{{p|chad}}  9:02 AM
morning. the {{o|Baltyk Cargo}} renewal is stuck in legal again

{{p|jordan}}  9:04 AM
on whose side? {{p|katarzyna|nickname}} told me tuesday they were ready to sign

{{p|chad}}  9:05 AM
ours. {{p|oyelaran|first}} flagged the liability cap. @{{p|oyelaran|handle|morenike.ow}} can you say more

{{p|oyelaran|display|Morenike O-W}}  9:11 AM
Clause 14.2. They struck the cap entirely. I can live with 2x fees, not unlimited. I told {{p|katarzyna|lower-first}} this on the phone and followed up in writing to {{e|k.wojciechowska@baltyk-cargo.example}}.

{{p|jordan}}  9:12 AM
ok. @{{p|chad|handle|chadw}} your call, you own the number

{{p|chad}}  9:15 AM
2x is fine. if {{p|katarzyna|title-last|Wojciechowska}} pushes back offer 3x and loop in {{p|marguerite|nickname}}. i'm in {{l|Chad|country}} next week btw (yes really, {{l|N'Djamena}}, family thing) so {{p|jordan|lower-first}} has the pen from the 16th

{{p|ngozi}}  9:17 AM
{{p|chad|first}} in {{l|Chad|country}}. finally.

{{p|jordan}}  9:18 AM
😐 noted. i'll take it. {{p|ngozi|first}} can you update the forecast, close date moves to {{d|2026-03-27|3/27}}

{{p|ngozi}}  9:21 AM
done. also their AP contact changed, it's now {{p|agnieszka|ascii-fold|Agnieszka Szczepanska}} ({{p|agnieszka|nickname}}), {{e|a.szczepanska@baltyk-cargo.example}}, {{ph|+48 58 555 0127}}

{{p|chad}}  9:22 AM
👍`)),

	work("hr-accommodation-note", "hr,confidential",
		usr(`Rewrite this as a formal file note. Neutral tone, no speculation, keep all the facts and dates.

HR note, private. Written by {{p|rashida}} {{d|2026-01-22|22/01/26}}.

Met with {{p|tevita}} (depot supervisor, {{l|Avonmouth}}, employee no. {{n|identifier|E-20417}}) at his request. Also present: his union rep {{p|hunter}}.

{{p|tevita|nickname}} disclosed that he was diagnosed with type 2 diabetes in November and has had two hypos on night shift. His GP, Dr {{p|gwenllian|title-last|Prys}} at {{o|Severnside Medical Practice}}, has written a fit note dated {{d|2026-01-15|15 January}} recommending no lone night working for three months. He handed me the note; copy on file.

He asked that this not be shared with his line manager, {{p|obinna}}, beyond "medical restriction, days only". I agreed. {{p|hunter|first}} asked about pay protection. I said I would check with {{p|ingrid}} but that shift premium is normally protected for 12 weeks under the policy.

{{p|tevita|first}} mentioned that his wife, {{p|leilani|first}}, also works for us (customer support) and he would prefer she is not approached about any of this.

Actions:
- me: confirm pay protection with {{p|ingrid|initials|IAT}} by {{d|2026-01-26|Mon 26th Jan}}
- me: occupational health referral, {{p|tevita|possessive}}'s consent form signed today
- {{p|hunter|initial-last|H. Maldonado}}: send the union's template risk assessment
- review meeting {{d|2026-04-20|20 April}}

For the anonymised case log he is to be described only as {{q|the depot supervisor on medical restriction who started in March 2021}}. Contact for {{p|tevita|initials|TFL}} while off rota: {{ph|07700 900388}}, {{e|tev.fifita@personal-mail.example}}. Home address on file is {{a|27 Pill Road, Shirehampton, Bristol BS11 9ZZ}}.`)).on("2026-01-15", "gwenllian", "2026-01-26", "rashida"),

	work("recruiting-debrief", "recruiting,candidate",
		usr(`Write the hiring committee summary from these debrief notes. Hire or no hire, strengths, concerns, and comp recommendation.

Debrief: Senior Customs Analyst, req {{n|identifier|R-2026-014}}
Candidate: {{p|akosua}} ("{{p|akosua|first}}", pronounced a-KOH-sia). Currently at {{o|Tema Port Services}}, {{l|Accra}}. Relocating to {{l|Rotterdam}} in August.
Onsite {{d|2026-05-19|19 May 2026}}. Panel: {{p|obinna}}, {{p|siobhan}}, {{p|maianh}}, {{p|dale}} (hiring manager).

{{p|obinna|first}} (customs case study): strong hire. She found the tariff misclassification in under ten minutes. "Better than I would have done."
{{p|siobhan|nickname}} (stakeholder interview): hire. Calm, direct. One concern: she has only worked under Ghanaian and ECOWAS rules, the UCC will be new.
{{p|maianh|first}} (data exercise): lean hire. SQL was fine, spreadsheet modelling slow. {{p|maianh|first}} noted {{p|akosua|title-last|Agyeman-Prempeh}} asked good questions about our data quality.
{{p|dale|first}}: hire. Wants her to start before peak.

References: {{p|oluwaseun}} at {{o|Lagos Freightline}} ({{ph|+234 1 555 0167}}) called her "the person you want when the port authority changes the rules overnight". Second referee, a Mr {{p|victor|title-last|Osei-Tutu}}, has not replied; {{p|dale|first}} emailed {{e|v.oseitutu@temaport.example}} twice.

Comp: she is on GHS 31,000 a month. Asked for EUR 78k. Band tops out at 74k. {{p|dale}} suggests 74k plus relocation of 6k. Visa: needs a Dutch highly skilled migrant permit, {{p|ingrid}} to confirm sponsorship timing. Her passport ({{l|Ghana}}) number for the IND form is {{g|gh-passport|G2841937}}, expires {{d|2031-02-08|08 FEB 2031}}.

Candidate contact: {{e|akosua.agyeman@fastmail.example}}, {{ph|+233 24 555 0183}}. Offer deadline she gave us: {{d|2026-06-05|5 June}}, she has another process running with {{o|Maersk|company}}.`),
		ast(`Recommendation: hire. Four of four interviewers support an offer for {{p|akosua}}, with one lean hire from the data exercise.

Strengths: fast and accurate on the customs case study; calm and direct with stakeholders; asks good questions about data quality; a strong reference from {{p|oluwaseun}}.
Concerns: no experience yet with the Union Customs Code; slower on spreadsheet modelling; the second referee, Mr {{p|victor|title-last|Osei-Tutu}}, has not responded.
Compensation: EUR 74,000, the top of the band, plus EUR 6,000 relocation, as {{p|dale}} proposes. She asked for 78,000.
Timing: she needs an answer by {{d|2026-06-05|5 June}} and a permit before starting.

I have left her passport number and her current salary out of the summary, since the committee does not need them.`),
		usr(`Good call on the passport. Two edits: the committee chair, {{p|marguerite}}, wants the interviewers named with their verdicts, so put {{p|obinna|first}}, {{p|siobhan|first}}, {{p|maianh|first}} and {{p|dale|first}} back in. And refer to the candidate as Ms {{p|akosua|title-last|Agyeman-Prempeh}} throughout, not by first name. For anything that leaves HR, describe her only as {{q|the Ghanaian customs analyst relocating to Rotterdam in August}}.`)).on("2026-05-19", "akosua", "2026-06-05", "akosua"),

	work("support-ticket", "support,ticket",
		usr(`Suggest a reply and an internal note for this ticket. The customer is right, we did double charge.

Ticket {{n|identifier|SUP-55871}}  |  Priority: High  |  Channel: email
Requester: {{p|aino}} <{{e|aino.makiranta@pohjolanrahti.example}}>
Organisation: {{o|Pohjolan Rahti Oy}}
Assignee: {{p|leilani}}
Created: {{d|2026-05-04|4.5.2026}} 08:41 EEST

Hei,

We were charged twice for April. Both payments came off the company card ending 4444 on {{d|2026-05-01|1.5.2026}}: EUR 1,250.00 and again EUR 1,250.00. Invoice {{n|identifier|INV-2026-04-0731}}.

I tried to call your support line and waited 25 minutes. Please refund one of the charges this week, our accountant {{p|linnea}} closes the books on the 8th.

If you need the full card number for the refund it is {{f|card-mc|5555 5555 5555 4444}}, exp 09/28. Our VAT number is FI28417390.

Terveisin,
{{p|aino}}
talouspäällikkö
{{o|Pohjolan Rahti Oy}}, {{a|Satamakatu 14 B, 33200 Tampere}}
puh. {{ph|+358 40 555 0149}}

--- internal ---
{{p|leilani|nickname}} {{d|2026-05-04|4 May}} 09:10: confirmed double capture in the processor. Asked {{p|chad}} to approve refund. Told {{p|aino|first}} never to send card numbers by email, redacted in the ticket body (not in the original mail, cannot edit that).
{{p|chad|first}} {{d|2026-05-04|4 May}} 09:32: approved. refund ref {{n|identifier|RF-88213}}.`),
		ast(`Suggested reply:

"Hei {{p|aino|first}}, you are right, and I am sorry. April was charged twice on {{d|2026-05-01|1 May}}. We have refunded one payment of EUR 1,250.00 today, reference RF-88213; it should reach the card within three to five working days, in time for your accountant to close the books. Please do not send card numbers by email in future; we never need the full number for a refund. I have removed it from our ticket. Kind regards, {{p|leilani}}"

Internal note: double capture confirmed and refunded with approval from {{p|chad}}. The original email still contains a full card number and cannot be edited; recommend asking the customer to treat the card as exposed.`),
		usr(`Send that, but make the greeting more formal, she signs as {{p|aino}} and we have never met. And copy their accountant, {{p|linnea}}, I think the address is {{e|linnea.akerblom@pohjolanrahti.example}}. Should I tell {{p|aino|first}} to cancel the card or is that overstepping?`)),

	work("crm-export", "crm,export",
		usr(`Dedupe this CRM export and tell me which records are the same person. Output one row per real person with the best phone and email.

contact_id|name|company|title|email|phone|owner|last_touch|notes
C-10021|{{p|katarzyna}}|{{o|Baltyk Cargo}}|Head of Procurement|{{e|k.wojciechowska@baltyk-cargo.example}}|{{ph|+48 58 555 0102}}|{{p|jordan|handle|jmwangi-hale}}|{{d|2026-03-10|2026-03-10}}|renewal, legal
C-10022|{{p|katarzyna|nickname}} {{p|katarzyna|last}}|Baltyk|procurement|{{e|kasia.w@baltyk-cargo.example}}|{{ph|0048 58 555 01 02}}|{{p|chad|handle|chadw}}|{{d|2025-11-30|2025-11-30}}|met at {{o|Transport Logistic}} fair
C-10023|{{p|katarzyna|caps-last}} {{p|katarzyna|caps-first}}|{{o|BALTYK CARGO SP. Z O.O.}}||{{e|K.WOJCIECHOWSKA@BALTYK-CARGO.EXAMPLE}}|||{{d|2024-06-02|2024-06-02}}|import from badge scan
C-10057|{{p|stavros}}|{{o|Aegean Reefer Lines}}|Fleet Manager|{{e|s.papadimitrakopoulos@aegeanreefer.example}}|{{ph|+30 21 0555 0136}}|{{p|jordan|handle|jmwangi-hale}}|{{d|2026-01-18|2026-01-18}}|prefers calls after 15:00 {{l|Athens}} time
C-10058|{{p|stavros|initial-last|S. Papadimitrakopoulos}}|Aegean Reefer|fleet|{{e|stavros@aegeanreefer.example}}|{{ph|+30 694 555 0136}}|{{p|ngozi|handle|naderibigbe}}|{{d|2025-09-09|2025-09-09}}|mobile. Name is too long for our badge printer
C-10090|{{p|oguz}}|{{o|Marmara Lojistik A.Ş.}}|Owner|{{e|oguzhan@marmaralojistik.example}}|{{ph|+90 212 555 01 58}}|{{p|chad|handle|chadw}}|{{d|2026-02-25|2026-02-25}}|
C-10091|{{p|oguz|ascii-fold|Oguzhan Celikkaya}}|Marmara Lojistik|owner|{{e|oguzhan@marmaralojistik.example}}|{{ph|+90 532 555 01 58}}|{{p|chad|handle|chadw}}|{{d|2026-02-26|2026-02-26}}|dup? spelled without Turkish letters
C-10092|{{p|gulsen}}|Marmara Lojistik|Finance|{{e|gulsen.karabulut@marmaralojistik.example}}|{{ph|+90 212 555 01 59}}|{{p|chad|handle|chadw}}|{{d|2026-02-26|2026-02-26}}|{{p|oguz|nickname}} Bey's sister-in-law, handles invoices`)),

	work("calendar-invite", "calendar,ics",
		usr(`This invite shows the wrong time for half the attendees. Can you see why from the raw ICS? Also list who has not responded.

BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Corvane Freight//Calendar {{n|version|11.4}}//EN
BEGIN:VEVENT
UID:{{n|uuid|3f2504e0-4f89-41d3-9a0c-0305e82c3301}}@corvanefreight.example
DTSTAMP:{{d|2026-06-02|20260602T081500Z}}
DTSTART;TZID=Europe/London:{{d|2026-06-09|20260609T140000}}
DTEND;TZID=Europe/London:{{d|2026-06-09|20260609T150000}}
SUMMARY:Q3 lane review ({{p|marguerite|nickname}} / {{p|valentina|nickname}} / {{l|Santiago}} team)
ORGANIZER;CN={{p|marguerite}}:mailto:{{e|m.delacroix-benoit@corvanefreight.example}}
ATTENDEE;CN="{{p|valentina|last-first|Sepúlveda Rojas, Valentina}}";PARTSTAT=ACCEPTED:mailto:{{e|vsepulveda@corvanefreight.example}}
ATTENDEE;CN={{p|cristobal}};PARTSTAT=NEEDS-ACTION:mailto:{{e|cristobal.aravena@corvanefreight.example}}
ATTENDEE;CN={{p|hemi}};PARTSTAT=TENTATIVE:mailto:{{e|hemi.tuhoro@corvanefreight.example}}
ATTENDEE;CN={{p|baraka|caps}};PARTSTAT=NEEDS-ACTION:mailto:{{e|baraka.sankale@corvanefreight.example}}
ATTENDEE;CN=Boardroom {{n|eponym|Brunel}};CUTYPE=ROOM:mailto:{{e|room-brunel@corvanefreight.example}}
LOCATION:Boardroom {{n|eponym|Brunel}}\, {{a|220 Quayside Road}}\, {{l|Bristol}} BS1 9ZZ / Teams
DESCRIPTION:Dial-in {{ph|+44 20 7946 0321}} PIN {{s|pin|774 219#}}\nAgenda from {{p|cristobal|nickname}} to follow.\n {{p|hemi|first}}: it is 01:00 for you\, feel free to skip and read the notes.
END:VEVENT
END:VCALENDAR`)),

	work("performance-review", "hr,review",
		usr(`Tighten this review. It is too long and the feedback for improvement is vague. Keep every concrete example.

Annual review FY25: {{p|penelope}} (Customer Operations Lead)
Reviewer: {{p|marguerite}}. Period {{d|2025-04-01|1 April 2025}} to {{d|2026-03-31|31 March 2026}}. Draft 2.

Summary
{{p|penelope|nickname}} had a strong year. She took over the {{l|Avonmouth}} desk when {{p|lucinda|first}} moved to key accounts and cut first-response time from 9 hours to under 3. She is leaving us in February to join her partner in {{l|Auckland}}, so this review doubles as a handover reference.

What went well
- The {{o|Pohjolan Rahti}} escalation in October. {{p|aino}} had threatened to leave; {{p|penelope|first}} flew to {{l|Tampere}}, sat with their dispatchers for two days and came back with a list of eleven fixes. Nine shipped. {{p|aino|first}} renewed.
- Mentoring. {{p|leilani}} and {{p|xiadani|first}} both name {{p|penelope|nickname}} as the reason they stayed. {{p|leilani|nickname}} was promoted in January.
- She wrote the out-of-hours playbook that {{p|tevita|possessive}}'s team now uses. {{x|common-word|Pen}}-and-paper fallback included, which saved us on {{d|2025-12-03|3 December}} when the label service went down.

What could be better
- Delegation. In the {{o|Baltyk Cargo}} dispute she handled every call herself. {{p|jordan}} offered to take the commercial side and was politely refused.
- Written updates. {{p|ingrid}} and I both had to chase for the monthly report three times. Ms {{p|penelope|title-last|Arkwright-Sodhi}} agrees this is fair.
- Conflict with {{p|chad|first}} over refund authority in Q2 was left to simmer. It resolved, but only after {{p|rashida}} from HR mediated.

Rating: Exceeds. Bonus: 12% (approved by {{p|marguerite|initials|MDB}}, {{d|2026-01-30|30 Jan 2026}}).
Forwarding contact after she leaves: {{e|pen.arkwright@kiwimail.example}}.`)),
}

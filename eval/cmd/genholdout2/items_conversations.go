package main

var itemsConversations = []def{
	long("agent-git", "tool-calls,agent,git",
		sys(`You are a coding agent with shell access. Run commands with the bash tool and explain what you find.`),
		usr(`The nightly export started producing empty files on Wednesday. Find the commit that did it.`),
		ast(`I will look at recent history for the exporter.

<tool_use name="bash">{"command":"git log --since='5 days ago' --format='%h %an <%ae> %ad %s' --date=short -- internal/export"}</tool_use>`),
		usr(`<tool_result name="bash">
a41c9e2 {{p|ngozi}} <{{e|ngozi@harborline.example}}> {{d|2026-03-05}} export: write header even when there are no rows
77b0d13 {{p|seun}} <{{e|seun.babatunde@harborline.example}}> {{d|2026-03-04}} export: switch to streaming writer
c19e5aa {{p|seun}} <{{e|seun.babatunde@harborline.example}}> {{d|2026-03-04}} export: drop the temp file, write straight to the bucket
0be7f41 {{p|bjorn|ascii-fold|Bjorn Akesson}} <{{e|bjorn.akesson@harborline.example}}> {{d|2026-03-02}} deps: bump {{o|Parquet|library}} writer
</tool_result>`),
		ast(`Two commits by {{p|seun}} landed on {{d|2026-03-04}}, the day the problem started. I will diff the streaming writer change.

<tool_use name="bash">{"command":"git show 77b0d13 --stat && git show 77b0d13 -- internal/export/writer.go | head -40"}</tool_use>`),
		usr(`<tool_result name="bash">
commit 77b0d13c5e2a4f6b8d0c1e3a5b7d9f1c3e5a7b9d
Author: {{p|seun}} <{{e|seun.babatunde@harborline.example}}>
Date:   {{d|2026-03-04|Wed Mar 4 17:20:31 2026}} +0100

    export: switch to streaming writer

    Pairing with {{p|thuy|ascii-first|Thuy}}. Flush is now the caller's job.

 internal/export/writer.go | 38 ++++++++++----------
 1 file changed, 19 insertions(+), 19 deletions(-)
-	defer w.Flush()
+	// caller flushes
</tool_result>

That looks like it. Before you fix it, who should review? {{p|seun|nickname}} is out this week.`)),

	long("agent-grep", "tool-calls,agent,handles",
		usr(`List every TODO in the repo that is assigned to someone who has left. People who left: {{p|tomasz|handle|tkb}}, {{p|kaho|handle|felix.tsang}}, and whoever "{{p|meilin|handle|mei}}" is.`),
		ast(`<tool_use name="grep">{"pattern":"TODO\\(","path":"src","output":"lines"}</tool_use>`),
		usr(`<tool_result name="grep">
src/auth/session.ts:41:  // TODO({{p|tomasz|handle|tkb}}): remove after the cookie migration
src/auth/session.ts:97:  // TODO({{p|kaho|handle|felix.tsang}}): rate limit by tenant, not by ip
src/billing/tax.ts:12:   // TODO({{p|meilin|handle|mei}}): VAT for the Canary Islands, ask {{p|inaki|first}}
src/billing/tax.ts:140:  // TODO({{p|caglar|handle|cyildirim}}): rounding, see PLAT-4471
src/ui/theme.ts:8:       // TODO(design): {{n|common-word|violet}} and {{n|common-word|sage}} palettes need AA contrast
src/ui/avatar.ts:33:     // TODO({{p|kaho|handle|felix.tsang}}): initials for mononyms, e.g. "{{p|wulan}}" should render as W
src/util/dates.ts:5:     // TODO: stop assuming {{n|common-word|may}} is a month name in the fuzzy parser
</tool_result>

"{{p|meilin|handle|mei}}" is {{p|meilin|western-order|Meilin Ouyang}}, she moved to the data team, so she has not left. Take her off the list.`)),

	long("agent-db", "tool-calls,agent,sql",
		sys(`You are a support engineering assistant. You may run read-only SQL with the query tool. Never include full card numbers in your answers.`),
		usr(`A customer called {{p|fatima|nickname}} says she was charged after cancelling. Her surname is {{p|fatima|last}} I think. Can you find her account and her last three charges?`),
		ast(`<tool_use name="query">{"sql":"select id, full_name, email, phone, status, cancelled_at from customers where full_name ilike '%idrissi%' limit 5"}</tool_use>`),
		usr(`<tool_result name="query">
 id    | full_name               | email                    | phone             | status    | cancelled_at
-------+-------------------------+--------------------------+-------------------+-----------+--------------
 55190 | {{p|fatima}} | {{e|f.elidrissi@example.org}} | {{ph|+33 6 39 98 01 27}} | cancelled | {{d|2026-01-28}}
 61877 | {{p|+Yasmine El Idrissi}}     | {{e|yasmine.eli@example.org}} | {{ph|+212 600 000 142}}  | active    |
(2 rows)
</tool_result>`),
		ast(`There are two matches. {{p|fatima}} (id 55190) cancelled on {{d|2026-01-28}}, which fits. Checking her charges.

<tool_use name="query">{"sql":"select id, amount, currency, card_last4, created_at from charges where customer_id = 55190 order by created_at desc limit 3"}</tool_use>`),
		usr(`<tool_result name="query">
 id      | amount | currency | card_last4 | created_at
---------+--------+----------+------------+---------------------
 9914207 |  19.00 | EUR      | 3155       | {{d|2026-03-01}} 04:00:12
 9871133 |  19.00 | EUR      | 3155       | {{d|2026-02-01}} 04:00:09
 9820045 |  19.00 | EUR      | 3155       | {{d|2026-01-01}} 04:00:11
</tool_result>

ok so two charges after cancellation. draft the apology email to {{p|fatima|nickname}}, and cc her daughter since the notes say {{p|+Yasmine El Idrissi|first|Yasmine}} handles her mail.`)),

	long("crm-assistant", "tool-calls,assistant,json",
		sys(`You are a sales assistant connected to the CRM. Tools: crm.search, crm.update. Today is {{d|2026-03-09|Monday, March 9, 2026}}.`),
		usr(`Who owns the Achterberg account and when did we last talk to them?`),
		ast(`{"tool_calls":[{"id":"call_01","type":"function","function":{"name":"crm.search","arguments":"{\"query\":\"Achterberg\"}"}}]}`),
		usr(`{"tool_call_id":"call_01","role":"tool","content":{"accounts":[{"id":"acc_2217","name":"{{o|Achterberg Installatietechniek B.V.|company-eponym}}","owner":"{{p|annelize}}","primary_contact":{"name":"{{p|robin}}","email":"{{e|r.achterberg@achterberg-installatie.example}}","mobile":"{{ph|+31 6 5550 0177}}"},"last_activity":{"type":"call","date":"{{d|2026-02-12}}","by":"{{p|annelize|initial-last|A. van der Merwe}}","note":"{{p|robin|first}} wants a quote for 12 seats. Call back after {{d|2026-03-09|9 maart}}."}}]}}

Today is {{d|2026-03-09|Monday, March 9, 2026}}. Remind me what I promised, and set the next activity for Wednesday.`)),

	long("calendar-assistant", "tool-calls,assistant,calendar",
		usr(`What do I have on Thursday, and can you move anything with {{p|enzo|nickname}} to Friday?`),
		ast(`Thought: I need the calendar for Thursday.
Action: calendar.list
Action Input: {"date": "{{d|2026-03-12}}"}`),
		usr(`Observation: [
  {"id": "ev1", "start": "{{d|2026-03-12|2026-03-12T09:00}}", "title": "1:1 {{p|enzo|nickname}} / {{p|ria|nickname}}", "attendees": ["{{e|lorenzo.dibenedetto@atelier-nord.example}}", "{{e|eleftheria.p@tallowfield.example}}"]},
  {"id": "ev2", "start": "{{d|2026-03-12|2026-03-12T11:30}}", "title": "Dentist, Dr. {{p|montse|last-part|Puigdevall}}", "location": "{{a|Carrer de Mallorca 214, 08008 Barcelona}}"},
  {"id": "ev3", "start": "{{d|2026-03-12|2026-03-12T15:00}}", "title": "Budget review ({{p|enzo|initials|LDB}}, {{p|inaki|initials|IEG}}, {{p|jb|nickname}})", "attendees": ["{{e|lorenzo.dibenedetto@atelier-nord.example}}", "{{e|inaki@etxeberria.example}}", "{{e|jb.lefevre@atelier-nord.example}}"]},
  {"id": "ev4", "start": "{{d|2026-03-12|2026-03-12T19:00}}", "title": "{{p|+Nikos|possessive}}'s name day dinner"}
]`),
		ast(`On Thursday {{d|2026-03-12|12 March}} you have a 1:1 with {{p|enzo|nickname}} at 09:00, the dentist at 11:30, the budget review with {{p|enzo|nickname}}, {{p|inaki|first}} and {{p|jb|nickname}} at 15:00, and {{p|+Nikos|possessive}}'s dinner at 19:00. Shall I move both meetings with {{p|enzo|nickname}}, or only the 1:1?`),
		usr(`only the 1:1. friday the 13th at 10. and text {{p|+Nikos|first}} that i'll be late, his number is {{ph|+30 690 000 0142}}`)),

	long("inbox-assistant", "tool-calls,assistant,email",
		sys(`You help the user manage email. Tools: mail.search(query), mail.draft(to, subject, body). Never send without confirmation.`),
		usr(`Find the last message from my landlord about the deposit and draft a reply asking for the itemized deductions.`),
		ast(`<function_call name="mail.search">{"query":"deposit from:landlord OR subject:deposit","limit":2}</function_call>`),
		usr(`<function_result>
[1] From: {{p|wiremu}} <{{e|wiremu@tewhaiti.example}}>   Date: {{d|2026-03-02|2 Mar 2026}}
    Subject: Bond refund, 88 Tory St
    Kia ora {{p|aisyah|first}}, the inspection was fine apart from the carpet in the hall. I will release the bond less $350 for cleaning. {{p|+Mereana Te Whaiti|first|Mereana}} will lodge the form this week. Ngā mihi, {{p|wiremu|nickname}}
[2] From: {{o|Paraone Property Management|company-eponym}} <{{e|rentals@paraone-property.example}}>   Date: {{d|2026-02-20|20 Feb 2026}}
    Subject: Final inspection booked
    Dear {{p|aisyah|formal|Ms binti Abdullah}}, your final inspection is booked for {{d|2026-02-27|Friday 27 February}} at 10 am.
</function_result>`),
		ast(`I found it. {{p|wiremu}} wrote on {{d|2026-03-02|2 March}} that he will keep $350 for carpet cleaning. Here is a draft:

To: {{e|wiremu@tewhaiti.example}}
Subject: Re: Bond refund, 88 Tory St

Kia ora {{p|wiremu|first}},

Thanks for the update. Before the form is lodged, could you send me the itemized invoice for the $350? The carpet was professionally cleaned on {{d|2026-02-25|25 February}} and I have the receipt.

Ngā mihi,
{{p|aisyah|first}}

Shall I save it?`),
		usr(`Yes, but sign it with my full name, {{p|aisyah}}, and copy the agency.`)),

	long("support-bot", "system-prompt,support",
		sys(`You are the virtual assistant for {{o|Lakeshore Mutual Health|company}}. The authenticated member is {{p|nastya|western|Anastasia Volkonskaya}}, member ID {{f|member-id|LMH-881420073}}, date of birth {{d|1988-12-14|12/14/1988}}, phone on file {{ph|847-555-0119}}. Verify identity by asking for the date of birth before discussing claims. Do not reveal the member ID unless the member states it first.`),
		usr(`hi, i need to know why my claim was denied`),
		ast(`I can help with that. To verify your identity, could you confirm your date of birth?`),
		usr(`{{d|1988-12-14|12/14/1988}}. this is {{p|nastya|western|Anastasia Volkonskaya}}, some of my mail says {{p|nastya|misspelled|Anastasia Volkonsky}} which is my husband's version of the name, and the member id is {{f|member-id|LMH-881420073}}`),
		ast(`Thank you, {{p|nastya|first}}. I see claim 26-0041-7781 was denied because the provider was listed as out of network. Would you like me to open an appeal?`),
		usr(`yes. and update my phone, the {{ph|847-555-0119}} one is old. new one is {{ph|(224) 555-0161}}. everyone calls me {{p|nastya|nickname}} by the way`)),

	long("wedding-toast", "personal,writing",
		usr(`I'm best man at my friend's wedding in June. Help me write a 3 minute toast. He's {{p|dilshan}}, she's {{p|selam}}. We met at uni in {{l|Colombo}}.`),
		ast(`Happy to help. A few questions so it sounds like you: How did {{p|dilshan|first}} and {{p|selam|first}} meet? Is there a story that shows who he is without embarrassing him in front of his grandmother? And what do you call each other?`),
		usr(`They met in a queue at the {{l|Addis Ababa}} airport, he was 4 kg over on baggage and she took his books in her carry on. I call him {{p|dilshan|nickname|Dilly}}, everyone at uni did. Her family calls her {{p|selam|nickname}}. His ammi (mrs {{p|dilshan|last}}) will be in the front row so nothing about the scooter. Oh and mention {{p|+Kasun}}, our friend who passed in 2021, briefly.`),
		ast(`Here is a first draft.

"Good evening. For those who do not know me, I have known {{p|dilshan|nickname|Dilly}} since our first week at university, which means I have known him longer than he has known how to pack a suitcase..."

Would you like it warmer or funnier? I kept the line about {{p|+Kasun}} short and near the end.`),
		usr(`funnier. and the wedding date moved, it's {{d|2026-06-20|the 20th of June}} now not the 13th, in case you reference it. also it's {{p|selam|ascii-variant|Selamawit G/Mariam}} on the invitation, that's how her family abbreviates it`)),

	long("caregiver-meds", "personal,medical",
		usr(`My mum is on a lot of medication and I want a simple chart. She's 79. Morning: amlodipine 5 mg, metformin 500. Evening: metformin 500, atorvastatin 20. And a patch she changes on Sundays.`),
		ast(`Here is a simple chart with morning and evening columns. Do you want me to add space for her to tick each dose, and should it show the prescriber or pharmacy number?`),
		usr(`yes ticks. her GP is Dr {{p|angharad|last}} at {{o|Llywelyn & Rees Surgery|company-eponym}}, {{ph|029 2018 0142}}. pharmacy is the one on Cowbridge Road. put her name at the top, {{p|+Gwenllian Harcourt}}, and her NHS number {{g|nhs|999 000 0142}} because the district nurse always asks`),
		ast(`Done. The header reads "{{p|+Gwenllian Harcourt}}, NHS no. {{g|nhs|999 000 0142}}", with Dr {{p|angharad|last}}'s number underneath. Anything else?`),
		usr(`she goes by {{p|+Gwenllian Harcourt|nickname|Gwen}}, add that in brackets. and add my brother {{p|wade|first}} as emergency contact, {{ph|07700 900233}}. her next review is {{d|2026-04-09|9th April}}`)),

	long("trip-booking", "tool-calls,travel",
		sys(`You are a travel assistant with access to flights.search and bookings.get.`),
		usr(`Pull up my booking K7QM2X and tell me if I can add a bag for the kid.`),
		ast(`<tool_use name="bookings.get">{"reference":"K7QM2X"}</tool_use>`),
		usr(`<tool_result name="bookings.get">{"reference":"K7QM2X","passengers":[{"name":"{{p|tomasz|airline|KOWALCZYKBRANDT/TOMASZ MR}}","type":"ADT","frequent_flyer":"QX 00014201"},{"name":"{{p|gosia|airline|WISNIEWSKA/MALGORZATA MS}}","type":"ADT"},{"name":"{{p|+Zofia Kowalczyk-Brandt|airline|KOWALCZYKBRANDT/ZOFIA MISS}}","type":"CHD","dob":"{{d|2018-09-02}}"}],"segments":[{"flight":"QX 264","from":"WAW","to":"DOH","date":"{{d|2026-03-27}}"},{"flight":"QX 908","from":"DOH","to":"DPS","date":"{{d|2026-03-27}}"}],"contact":{"email":"{{e|tomek.kb@example.com}}","phone":"{{ph|+48 22 555 01 99}}"},"baggage":{"ADT":"30kg","CHD":"30kg"}}</tool_result>`),
		ast(`{{p|+Zofia Kowalczyk-Brandt|first|Zofia}} already has a 30 kg allowance on both segments on {{d|2026-03-27|27 March}}, the same as the adults, so there is nothing to add. Would you like seats together?`),
		usr(`yes please. window for {{p|+Zofia Kowalczyk-Brandt|nickname|Zosia}}, and {{p|gosia|nickname}} wants the aisle. my name is spelled wrong in the frequent flyer profile by the way, it says {{p|tomasz|misspelled|Tomas Kowalczyk-Brant}}`)),

	long("hr-bot", "system-prompt,hr",
		sys(`You are the HR helpdesk assistant for {{o|Brightquay|company}}. Employee record for this session: name {{p|jun}}, preferred name {{p|jun|nickname}}, employee ID 00417, manager {{p|bobby}}, hire date {{d|2017-09-05|September 5, 2017}}, PTO balance 11.5 days. Answer policy questions from the handbook only.`),
		usr(`hey it's {{p|jun|nickname}}. how much PTO do I have and does it roll over?`),
		ast(`Hi {{p|jun|nickname}}. You have 11.5 days. Up to 5 days roll over into the next year; anything above that expires on the last day of the year.`),
		usr(`ok. I want to take {{d|2026-04-06|April 6}} through {{d|2026-04-17|April 17}}. does {{p|bobby|nickname}} need to approve (the system shows him as {{p|bobby}}) or can {{p|bobby|title-last|Mr. Tannenbaum}}'s delegate do it? he's on medical leave. my hire date was {{d|2017-09-05|September 5, 2017}} if the tenure rule matters, and for the record my legal name on the form should be {{p|jun}}`)),

	long("config-debug", "coding,secrets",
		usr(`My app can't connect to the database after I moved it to a new server. Error: "FATAL: password authentication failed for user app_rw"`),
		ast(`That error means the server received the connection and rejected the credentials. Can you share the connection settings your app uses (redact the password) and the relevant line of pg_hba.conf?`),
		usr(`i dont know which part is the password honestly. here is all of it

[database]
url = postgresql://app_rw:{{s|password|N3ptune%40Orbit-77}}@db2.internal.example:5432/app?sslmode=verify-full
pool = 20

[smtp]
host = smtp.internal.example
user = {{e|no-reply@savannahpay.example}}
pass = {{s|password|mailer-9Qx2-LpZ7}}

[admin]
bootstrap_user = {{e|sibusiso.mthembu@savannahpay.example}}
bootstrap_token = {{s|token|bt_live_Yb6Nd3Vh5Sj0Ga8Tb3XkQ7mP2vL8sR4w}}

# pg_hba.conf
host  app  app_rw  10.0.4.0/24  scram-sha-256

{{p|sbu|nickname}} set this up before he went on leave and i can't reach him till the 16th`)),

	long("name-correction", "assistant-names,typo",
		usr(`Summarize: "Present were Ms. {{p|aino|ascii-fold|Aino Kaariainen}}, Mr. {{p|stefan|ascii-fold|Stefan Constantinescu}} and Dr. {{p|zsofia|western-order|Zsófia Szabó}}. {{p|aino|last-ascii|Kaariainen}} proposed moving the audit to Q3; {{p|stefan|last}} objected; {{p|zsofia|last}} abstained."`),
		ast(`Summary: At the meeting, {{p|aino|ascii-fold|Aino Kaariainen}} proposed moving the audit to Q3. {{p|stefan|ascii-fold|Stefan Constantinescu}} objected and {{p|zsofia|western-order|Zsófia Szabó}} abstained, so no decision was recorded.`),
		usr(`Good, but fix the spellings: it's {{p|aino}} with the dots and {{p|stefan}} with the comma below the S. And {{p|zsofia|title-last|Dr. Szabó}} writes her name family name first, {{p|zsofia}}. Then add that the next meeting is {{d|2026-03-24|24.3.2026}} in {{l|Tampere}}.`)),

	long("legal-intake", "legal,personal",
		sys(`You are an intake assistant for a tenant rights clinic. Collect facts; do not give legal advice.`),
		usr(`i got an eviction notice yesterday and i dont know what to do`),
		ast(`I am sorry, that is stressful. I can take down the details so an advisor can call you. What is your full name, the address of the rental, and the date on the notice?`),
		usr(`{{p|precious}}. {{a|Flat 3, 41 Abbeville Road, London SW4 9ZZ}}. notice is dated {{d|2026-03-03|3rd March}} and says i have to be out by {{d|2026-05-05|5 May}}. landlord is a company, {{o|Thorncroft Estates Ltd|company-eponym}}, but the letter is signed by a {{p|basil|initial-last|B. Thorncroft}}`),
		ast(`Thank you, {{p|precious|title-last|Ms Ndlovu}}. Is your deposit protected in a scheme, and what is the best number and time to reach you?`),
		usr(`no idea about the deposit. {{ph|07700 900552}} after 6pm. if i dont pick up try my partner {{p|+Themba}}, {{ph|07700 900553}}. please dont email, {{p|basil|first}} thorncroft's son works at my office and i dont want it on my work screen`)),

	long("agent-shell", "tool-calls,agent,shell,path",
		usr(`ssh to the build box keeps asking for a password. figure out why. you can run commands on my laptop.`),
		ast("```bash\nwhoami && ls -la ~/.ssh && cat ~/.ssh/config\n```"),
		usr(`stdout:
{{p|christian|handle|clindgren}}
total 32
drwx------   6 {{p|christian|handle|clindgren}}  staff   192 {{d|2026-02-11|Feb 11}} 09:14 .
drwxr-x---+ 41 {{p|christian|handle|clindgren}}  staff  1312 {{d|2026-03-08|Mar  8}} 22:40 ..
-rw-r--r--   1 {{p|christian|handle|clindgren}}  staff   211 {{d|2026-02-11|Feb 11}} 09:14 config
-rw-------   1 {{p|christian|handle|clindgren}}  staff   419 {{d|2024-06-03|Jun  3  2024}} id_ed25519
-rw-r--r--   1 {{p|christian|handle|clindgren}}  staff   104 {{d|2024-06-03|Jun  3  2024}} id_ed25519.pub
-rw-r--r--   1 {{p|christian|handle|clindgren}}  staff  2210 {{d|2026-03-01|Mar  1}} 17:02 known_hosts
Host build
  HostName build-01.internal.example
  User {{p|christian|handle|christian.lindgren-ochoa}}
  IdentityFile /Users/{{p|christian|path|clindgren}}/.ssh/id_rsa
  # old box: ssh {{p|august|handle|awennerstrom}}@build-00.internal.example, pw was {{s|password|Midsommar!2019}}`),
		ast("The config points at `id_rsa`, which does not exist; your key is `id_ed25519`. Checking what the server offers:\n\n```bash\nssh -v build 2>&1 | grep -E 'Offering|Authentications|identity file' | head\ncat ~/.ssh/id_ed25519.pub\n```"),
		usr(`stdout:
debug1: identity file /Users/{{p|christian|path|clindgren}}/.ssh/id_rsa type -1
debug1: Authentications that can continue: publickey,password
ssh-ed25519 {{n|public-key|AAAAC3NzaC1lZDI1NTE5AAAAIK2mQ9xLp4Zt7Wc1Yb6Nd3Vh5Sj0Ga8Tb3XkQ7mP2vL8}} {{e|christian@lindgren-ochoa.example}}

ok fix the config. also {{p|august|first}} left in 2024 so drop that comment`)),

	long("agent-tests", "negative,tool-calls,agent",
		sys(`You are a coding agent. Tools: bash, edit.`),
		usr(`Tests are flaky on main. Run the suite twice and tell me which ones flip.`),
		ast(`<tool_use name="bash">{"command":"for i in 1 2; do go test ./... 2>&1 | grep -E '^(FAIL|ok|---)' ; done"}</tool_use>`),
		usr(`<tool_result name="bash">
ok  	example.org/tidewatch/internal/chart	0.412s
--- FAIL: {{n|identifier|TestGraceWindow}} (0.03s)
FAIL	example.org/tidewatch/internal/sched	0.288s
ok  	example.org/tidewatch/internal/store	1.204s
ok  	example.org/tidewatch/internal/chart	0.398s
ok  	example.org/tidewatch/internal/sched	0.301s
--- FAIL: {{n|identifier|TestMarkStale}} (1.12s)
FAIL	example.org/tidewatch/internal/store	1.377s
</tool_result>`),
		ast(`Two tests flip: {{n|identifier|TestGraceWindow}} in sched and {{n|identifier|TestMarkStale}} in store. Both look time dependent. I will read the first one.

<tool_use name="bash">{"command":"grep -n 'time.Now\\|Sleep' internal/sched/*_test.go | head"}</tool_use>`),
		usr(`<tool_result name="bash">
internal/sched/window_test.go:41:	start := time.Now()
internal/sched/window_test.go:58:	time.Sleep(25 * time.Millisecond)
internal/sched/window_test.go:77:	if time.Since(start) > {{n|identifier|gracePeriod}} {
</tool_result>

Makes sense. Inject a clock instead of sleeping. We are on go {{n|version|1.22.1}} and {{o|Testify|library}} {{n|version|v1.9.0}}, no other deps please.`)),

	long("general-qa", "negative,general",
		usr(`What is the difference between a will and a living trust? Keep it simple, US context.`),
		ast(`A {{n|common-word|will}} takes effect when you die and goes through probate. A living trust takes effect while you are alive, holds assets in the trust's name, and usually avoids probate. Many people have both.`),
		usr(`If I {{n|common-word|grant}} my sister power of attorney, can she change my {{n|common-word|will}}? And does a {{n|common-word|bill}} of sale count as proof that I gave her the car? Also what is a {{n|common-word|frank}} discussion with a lawyer going to cost me, roughly, in a mid-size city. No names needed, this is hypothetical.`)),

	long("research-tools", "negative,tool-calls,public-figure",
		usr(`I'm writing a short piece on programming language history. Who is each of these named after: {{o|Pascal|product}}, {{o|Ada|product}}, {{o|Haskell|product}}, {{o|Erlang|product}}?`),
		ast(`<tool_use name="web.search">{"query":"programming languages named after people Pascal Ada Haskell Erlang"}</tool_use>`),
		usr(`<tool_result name="web.search">
1. "{{o|Pascal|product}} is named for the mathematician {{n|public-figure|Blaise Pascal}}; {{o|Ada|product}} honors {{n|public-figure|Ada Lovelace}}."
2. "{{o|Haskell|product}} takes its name from the logician {{n|public-figure|Haskell Curry}}, whose surname also gives us currying."
3. "{{o|Erlang|product}} is named after the Danish mathematician {{n|public-figure|Agner Krarup Erlang}}, and conveniently also abbreviates {{o|Ericsson|company-eponym}} Language."
4. "The {{o|Turing|product}} language, the {{o|Gödel|product}} language and the {{o|Occam|product}} language follow the same tradition."
</tool_result>

Good. Add one sentence each on {{n|public-figure|Alan Turing}}, {{n|public-figure|Kurt Gödel}} and {{n|public-figure|William of Ockham}}, and note that {{o|Kotlin|product}} and {{o|Java|product}} are islands, not people.`)),
}

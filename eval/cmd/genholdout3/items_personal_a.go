package main

var itemsPersonalA = []def{
	personal("family-group-chat", "chat,family,logistics",
		usr(`can you make a simple schedule out of this for the fridge? who drives who, when

[{{d|2026-09-13|9/13/26}}, 7:02 PM] {{p|itzel}}: ok week plan. {{p|desmond|nickname}} you have {{p|paloma|first}} monday + weds, swim at 4:15 at the Y
[{{d|2026-09-13|9/13/26}}, 7:03 PM] {{p|desmond|display|Des 🛠}}: monday yes. weds i have the dentist at 3:30, dr {{p|anjali|lower-last}} moved me
[{{d|2026-09-13|9/13/26}}, 7:03 PM] {{p|itzel}}: {{p|anjali|last}} is the PEDIATRICIAN {{p|desmond|nickname|des}}. your dentist is the other one
[{{d|2026-09-13|9/13/26}}, 7:04 PM] {{p|desmond|display|Des 🛠}}: right. dr whoever. still 3:30
[{{d|2026-09-13|9/13/26}}, 7:05 PM] {{p|itzel}}: fine i'll ask {{p|rosario|nickname|chayo}}. mamá can you take {{p|paloma|nickname}} wednesday? swim 4:15
[{{d|2026-09-13|9/13/26}}, 7:09 PM] {{p|rosario|display|Abuela Chayo}}: Sí mija. Yo llevo a la {{p|paloma|first}}. Y el {{p|thaddeus|nickname}}?
[{{d|2026-09-13|9/13/26}}, 7:10 PM] {{p|itzel}}: {{p|thaddeus|first}} has robotics tues/thurs til 5:30, {{p|herb|first}} next door drives them, his grandson is on the team
[{{d|2026-09-13|9/13/26}}, 7:10 PM] {{p|itzel}}: mr {{p|herb|lower-last}} wants gas money btw. i told him to stop it
[{{d|2026-09-13|9/13/26}}, 7:12 PM] {{p|desmond|display|Des 🛠}}: {{p|herb|lower-first}} is 81 should he be driving
[{{d|2026-09-13|9/13/26}}, 7:12 PM] {{p|itzel}}: he drives better than you
[{{d|2026-09-13|9/13/26}}, 7:14 PM] {{p|rosario|display|Abuela Chayo}}: Necesito la dirección de la alberca
[{{d|2026-09-13|9/13/26}}, 7:15 PM] {{p|itzel}}: {{a|1840 Mesquite Canyon Rd, Tucson, AZ 85745}}. front desk {{ph|(520) 555-0116}}
[{{d|2026-09-13|9/13/26}}, 7:16 PM] {{p|itzel}}: also friday is {{p|paloma|possessive}}'s parent conference with ms {{p|violet|lower-last}}, 2:40, room 14. {{p|desmond|nickname|des}} that's you, i have court
[{{d|2026-09-13|9/13/26}}, 7:17 PM] {{p|desmond|display|Des 🛠}}: 👍 what do i say
[{{d|2026-09-13|9/13/26}}, 7:17 PM] {{p|itzel}}: that she reads at night with a flashlight and we're not sorry
[{{d|2026-09-13|9/13/26}}, 7:20 PM] {{p|thaddeus|display|tad}}: can i be removed from this chat`)),

	personal("medical-referral-letter", "medical,letter,uk",
		usr(`My GP sent this and I do not understand what happens next or whether I should be worried. Can you explain it in plain English?

{{o|Severnside Medical Practice}}
{{a|12 Harbour Parade, Portishead, Bristol BS20 9ZZ}}
Tel {{ph|01275 496 0133}}

{{d|2026-03-02|2 March 2026}}

Dr {{p|sibusiso|initial-last|S. Ndlovu-Khumalo}}
Consultant Cardiologist
{{o|Bristol Heart Institute}}

Dear Dr {{p|sibusiso|title-last|Ndlovu-Khumalo}},

Re: Mrs {{p|folasade}}, DOB {{d|1984-07-29|29/07/1984}}, NHS No. {{g|gb-nhs|485 277 1960}}
{{a|8 Nore Road, Portishead BS20 9ZZ}}

Thank you for seeing this 41-year-old solicitor who presented on {{d|2026-02-24|24 February}} with a six-week history of palpitations, worse on exertion, with one episode of near-syncope while running. There is a family history: her father, {{p|+Babatunde Adeyemi|full}}, died suddenly aged 52.

Examination was unremarkable. BP {{n|measurement|118/74}}, pulse 68 regular. Resting ECG shows a QTc of {{n|measurement|472 ms}}. Bloods including TFTs normal ({{d|2026-02-26|26/02}}).

I would be grateful for your assessment, including a 7-day monitor and echocardiogram. She is not on regular medication. No known drug allergies. She is anxious, understandably, and her husband {{p|rupert|first}} attended with her.

Mrs {{p|folasade|title-last|Adeyemi-Clarke}} prefers to be contacted on her mobile, {{ph|07700 900261}}, and is happy for messages to be left. She has asked that letters are not sent to her work address.

Yours sincerely,

Dr {{p|gwenllian}}
GMC {{x|professional-registration|6142873}}

cc patient`),
		ast(`In plain English: your GP, Dr {{p|gwenllian|title-last|Prys}}, is asking a heart specialist, Dr {{p|sibusiso|title-last|Ndlovu-Khumalo}}, to see you because of palpitations, one near-faint while running, a heart tracing with a slightly long interval, and your father's sudden death at 52. None of that is a diagnosis. It is a careful referral: the specialist will probably fit a small monitor for a week and do an ultrasound scan of the heart. What happens next is that the hospital writes or phones with an appointment. Until then the usual advice is to avoid hard exertion and to seek urgent help for fainting or chest pain. Would you like questions to take to the appointment?`),
		usr(`Yes please, five questions at most. And can you tell me what QTc means without frightening me? My husband {{p|rupert|nickname}} has already looked it up and is no help. I am {{q|the 41-year-old solicitor}} in the letter, by the way, {{p|folasade|nickname}}; nobody calls me {{p|folasade|first}} except the NHS.`)).on("2026-02-24", "folasade", "2026-03-02", "gwenllian"),

	personal("insurance-claim-letter", "insurance,letter,us",
		usr(`They denied part of my claim. Help me write an appeal. Here is their letter.

{{o|Saguaro Mutual Insurance Company}}
Claims Department, {{a|P.O. Box 4410, Phoenix, AZ 85001}}

{{d|2026-10-06|October 6, 2026}}

{{p|desmond|caps}}
{{a|2215 W Calle Paraiso
Tucson, AZ 85745}}

Policy number: {{f|policy|HO-3-77204918-02}}
Claim number: {{n|identifier|CLM-AZ-2026-118843}}
Date of loss: {{d|2026-08-22|08/22/2026}}
Adjuster: {{p|hunter}}, ext. 4417

Dear Mr. {{p|desmond|title-last|Whitfield}}:

We have completed our review of your claim for water damage at the insured property. Based on the inspection performed by {{o|Ocotillo Restoration LLC}} on {{d|2026-09-03|September 3, 2026}}, we are approving payment of {{n|money|$6,412.80}} for the kitchen ceiling and cabinets, less your $1,000 deductible.

We are unable to cover the replacement of the roof underlayment ($9,850.00). Our adjuster, Mr. {{p|hunter|title-last|Maldonado}}, determined that the damage resulted from long-term deterioration rather than the monsoon storm of {{d|2026-08-22|August 22}}. See policy section I, exclusion 2(c).

Payment will be issued jointly to you and your mortgage servicer, {{o|First Sonoran Bank}}, loan number ending 3381. If you prefer direct deposit, the account we have on file is routing {{f|us-routing|091004727}}, account {{f|us-account|4417 0038 2291}}.

You may appeal within 60 days of this letter. Send written appeals to the address above or to {{e|appeals@saguaromutual.example}}. You may also call {{ph|1-800-555-0199}}.

Sincerely,
{{p|sue|alias|Susan Harrowgate}}
Claims Supervisor

What they are ignoring: the roofer, {{p|ernesto}} of {{o|Zavaleta Roofing|company-eponym}}, wrote that the underlayment was fine in his {{d|2026-05-15|May 15}} inspection. I have that report. {{p|itzel|first}} thinks we should also mention that {{p|hunter|last}} spent eleven minutes on the roof.`)).on("2026-08-22", "desmond", "2026-09-03", "hunter"),

	personal("flight-booking", "travel,passport,booking",
		usr(`Check these against each other: the airline says the name on the booking does not match the passport and wants a fee to fix it. Are they right?

YOUR BOOKING IS CONFIRMED
Booking reference: 7QXK2M
Issued {{d|2026-11-02|02NOV26}} by {{o|Kestrel Pacific Airways}}

Passenger 1: {{p|minori|booking|KOBAYAKAWA/MINORI MS}}
Passenger 2: {{p|lachlan|booking|MCALLISTERNG/LACHLAN MR}}
Passenger 3: {{p|sora|booking|MCALLISTERNG/SORA MSTR}} (child, DOB {{d|2019-04-11|11APR19}})

KP 412  {{l|Sydney}} SYD 21:40  to  {{l|Tokyo}} HND 05:55+1   {{d|2026-12-19|19DEC26}}   seats 41A 41B 41C
KP 411  {{l|Tokyo}} HND 22:15  to  {{l|Sydney}} SYD 09:50+1   {{d|2027-01-08|08JAN27}}   seats 38D 38E 38F

Ticket numbers: {{n|identifier|9812400417733}}, {{n|identifier|9812400417734}}, {{n|identifier|9812400417735}}
Contact: {{e|minori.k@harbourmail.example}} / {{ph|+61 491 570 156}}
Paid with Visa ending 1881. Frequent flyer {{n|identifier|KP88204417}} ({{p|lachlan|initial-last|L. McAllister-Ng}}).

And the passports:

{{p|lachlan|mrz|P<AUSMCALLISTER<NG<<LACHLAN<JAMES}}<<<<<<<<<<<<<<<
Passport no. {{g|au-passport|PA4829173}}, {{p|lachlan|last-first|McALLISTER-NG, Lachlan James}}, born {{d|1986-02-03|03 FEB 1986}}, expires {{d|2031-06-14|14 JUN 2031}}

Japanese passport: {{p|minori|caps-last}} {{p|minori|caps-first}} / {{p|minori|native-script|小早川 みのり}}, no. {{g|jp-passport|TK4829173}}, born {{d|1988-09-30|30 SEP 1988}}, expires {{d|2029-03-21|21 MAR 2029}}

{{p|sora|first}} travels on his Australian passport {{g|au-passport|PB7730194}}. My name is spelled right, it is {{p|lachlan|nickname}}'s that lost its hyphen.`),
		ast(`The airline is not right to charge for this. Airline reservation systems cannot store hyphens, so the surname is stored without one, and that matches the machine-readable line of the passport, where the hyphen is also replaced. The three bookings match the three passports. I would reply quoting the machine-readable zone and ask them to confirm no change is needed.`),
		usr(`Great. Draft that reply for me, from {{p|minori}}, and mention that {{p|sora|possessive}}'s date of birth on the booking is correct too.`)).on("2026-12-19", "minori", "2027-01-08", "minori"),

	personal("tax-question", "tax,us,bank",
		usr(`First year filing jointly and I am confused about which numbers go where. Our details:

Me: {{p|itzel}}, SSN {{g|us-ssn|912-34-5078}}, born {{d|1987-12-04|12/04/1987}}
Spouse: {{p|desmond|middle|Desmond R. Whitfield}}, SSN {{g|us-ssn|918-27-4410}}
Dependents: {{p|paloma}} (9, SSN {{g|us-ssn|900-41-2267}}) and {{p|thaddeus}} (13, SSN {{g|us-ssn|900-41-3095}})
Address: {{a|2215 W Calle Paraiso, Tucson, AZ 85745}}

My W-2 from {{o|Pima County Superior Court}}: box 1 $88,410.22, box 2 $11,903.00. Employer EIN {{x|employer-tax-id|86-0000417}}.
His is self-employment, {{o|Whitfield Carpentry LLC|company-eponym}}, EIN {{x|employer-tax-id|86-0044172}}, net about $52,000, and he paid estimated tax on {{d|2026-04-15|4/15}}, {{d|2026-06-15|6/15}} and {{d|2026-09-15|9/15}} but missed January.
We also paid my mother {{p|rosario}} $400 a month for childcare. She lives with us and has an ITIN, {{n|placeholder|9XX-70-XXXX}}, I will fill in the real one later.

Refund should go to our joint checking: routing {{f|us-routing|312077585}}, account {{f|us-account|000648213907}}.

Questions: 1. Can we claim the child and dependent care credit when the carer is my mother? 2. Does {{p|desmond|nickname}} owe a penalty for the missed estimated payment? 3. Our preparer last year, {{p|pearl}} at {{o|H&R Block|company}}, said something about a "safe harbor" and I nodded without understanding.`),
		ast(`Short answers, then detail. 1. Payments to a parent can qualify for the child and dependent care credit if she is not your dependent and the care lets you both work; {{p|rosario}} would need to report the income, and you need her taxpayer number on the form. 2. A missed fourth-quarter estimated payment usually leads to an underpayment penalty for {{p|desmond|first}}, unless a safe harbor applies. 3. The safe harbor {{p|pearl|first}} mentioned means no penalty if your payments covered either 90 percent of this year's tax or 100 percent of last year's. Do you know last year's total tax?`),
		usr(`Last year's total was $14,220. And a correction: {{p|desmond|nickname}} did pay in January after all, he found the confirmation dated {{d|2027-01-15|1/15}}, $2,600, it just came out of the business account instead of ours. Does that change your answer to 2? Also {{p|rosario|nickname}} says she would rather not "be on any form", how do I explain to my mother that she has to be?`)),

	personal("card-dispute", "banking,dispute",
		usr(`Write a dispute letter to my bank. Firm but polite. Facts below.

i am {{p|chiara}}. my card was charged three times by a hotel in {{l|Lisbon}} that i never stayed at. i cancelled the booking in writing on {{d|2026-06-28|28 giugno}} and have the email.

card: {{f|card-visa|4716 3829 0154 7728}} (the bank's own debit card), exp 11/27. account IBAN {{f|it-iban|IT64 K030 6909 6061 0000 0048 215}}.
charges: {{d|2026-07-03|03/07/2026}} EUR 412.00, {{d|2026-07-03|03/07/2026}} EUR 412.00, {{d|2026-07-05|05/07/2026}} EUR 60.00 "minibar", all {{o|Hotel Miradouro da Graça}}.
i called the bank on {{d|2026-07-07|7 luglio}} and spoke to a signor {{p|gianmarco|title-last|Dall'Orto}} who opened case {{n|identifier|PRT-2026-448120}} and then nothing happened. his direct line was {{ph|02 5550 0148}}.
my codice fiscale if they need it: {{g|it-cf|DNGCHR88E57H501T}}
address: {{a|Via dei Serpenti 41, 00184 Roma}}
phone {{ph|+39 347 555 0162}}, email {{e|chiara.deangelis@postacerta.example}}

the booking was made by my grandmother's carer, {{p|ludovica}}, using my card with permission, for my nonna {{p|giuseppina}} ({{p|giuseppina|honorific|nonna Pina}}) who then broke her hip and could not travel. so the name on the booking is {{p|giuseppina|caps|GIUSEPPINA LO PRESTI}}, not mine. does that matter?`),
		ast(`It matters a little, and it helps you. The cardholder authorised the booking, the guest was {{p|giuseppina}}, and the booking was cancelled in writing before the stay, so the two room charges are for a service not provided and the minibar charge is impossible. In the letter I would state plainly that the guest named on the reservation is your grandmother, that {{p|ludovica}} made the booking with your permission, and attach the cancellation email of {{d|2026-06-28|28 June}}. Shall I write it in Italian, addressed to the bank's complaints office, citing case {{n|identifier|PRT-2026-448120}}?`),
		usr(`yes in italian please. sign it {{p|chiara}}, and say that signor {{p|gianmarco|title-last|Dall'Orto}} promised a call back within five working days on {{d|2026-07-07|7 luglio}} and never called. do NOT put the full card number in the letter, only the last four.`)),

	personal("legal-letter-deposit", "legal,letter,tenancy",
		usr(`Is this letter from my landlord's solicitor as scary as it sounds? What are my options? I think the dates are wrong.

{{o|Brekke Oyelaran LLP|company-eponym}}
Solicitors
{{a|3rd Floor, 19 Corn Street, Bristol BS1 9ZZ}}
Our ref: {{n|identifier|MOW/DRA/00417-2}}    Your ref: none
{{d|2026-04-27|27 April 2026}}

BY FIRST CLASS POST AND EMAIL ({{e|rupert.ca@harbourmail.example}})

Mr {{p|rupert|initial-last|R. Clarke-Ashby}} and Mrs {{p|folasade|initial-last|F. Adeyemi-Clarke}}
{{a|8 Nore Road, Portishead BS20 9ZZ}}

Dear Sir and Madam,

OUR CLIENT: MR {{p|constantin|caps}}
PROPERTY: {{a|FLAT 2, 114 COTHAM HILL, BRISTOL BS6 9ZZ}} ("the Property")

We act for Mr {{p|constantin|title-last|Dragomir}}, your former landlord. You vacated the Property on {{d|2026-01-31|31 January 2026}}. Our client holds a deposit of £1,650 protected with the {{o|Deposit Protection Service}} under certificate {{n|identifier|DPS-41772093}}.

Our client's inventory clerk, Ms {{p|matilda}}, attended on {{d|2026-02-02|2 February 2026}} and recorded damage to the kitchen worktop, a missing wardrobe door and rent arrears of £412.50 for the period {{d|2026-01-01|1 January}} to {{d|2026-01-15|15 January 2026}}. Our client therefore proposes to retain £1,290.00.

Unless we receive your written agreement by 4 pm on {{d|2026-05-11|11 May 2026}} our client will refer the matter to adjudication and reserves the right to issue proceedings in the County Court without further notice.

Yours faithfully,

{{p|oyelaran}}
Partner
Direct dial {{ph|0117 496 0171}}
{{e|m.oyelaran-whitcombe@brekkeoyelaran.example}}

My notes: rent was paid to {{p|constantin|nickname}} by standing order on the 1st of every month including January, sort code {{f|gb-sort-code|30-96-17}} account {{f|gb-account|45820361}}, I have the statements. {{p|folasade|nickname}} says, as a solicitor, that "without further notice" is bluster. The wardrobe door was missing when we moved in on {{d|2023-08-12|12/08/2023}}, it is on the check-in inventory signed by {{p|matilda|nickname}} herself.`)).on("2026-01-31", "rupert", "2026-02-02", "matilda", "2026-05-11", "constantin"),

	personal("school-email", "school,email",
		usr(`How should I reply to this? I do not want to sound defensive but I also do not think {{p|temilola|nickname}} did anything wrong.

From: {{p|bronislawa|initial-last|B. Kowalczyk}} <{{e|bkowalczyk@rowanbrook-primary.example}}>
To: {{p|folasade}} <{{e|sade.ac@harbourmail.example}}>; {{p|rupert}} <{{e|rupert.ca@harbourmail.example}}>
Cc: {{e|office@rowanbrook-primary.example}}
Sent: {{d|2026-11-18|Wednesday, 18 November 2026 16:22}}
Subject: {{p|temilola|first}} in Year 4, a quick word

Dear Mr and Mrs {{p|folasade|misspelled|Adeyemi-Clark}},

I hope you are both well. I wanted to let you know about something that happened at lunchtime today before {{p|temilola|first}} tells you her version, which I am sure will be vivid.

{{p|temilola|nickname}} and another child, whom I will call P, disagreed about the rules of a game. {{p|temilola|first}} told P that her mummy is a lawyer and would "sue the whole of Year 4". P was upset. Mrs {{p|aroha|title-last|Ngata-Williams}}, our lunchtime supervisor, spoke to both girls and they have made up.

I am not concerned, but I would appreciate it if you could have a gentle word at home about using {{p|temilola|possessive}}'s considerable vocabulary kindly. She is a delight to teach and her reading age is now assessed at 11 years 4 months.

Parents' evening is on {{d|2026-12-03|Thursday 3rd December}}; I have you down for 5.40 pm. If that does not suit, please ring the office on {{ph|01275 496 0108}}.

With best wishes,
{{p|bronislawa|first}} {{p|bronislawa|last}} (Mrs)
Class teacher, 4K
{{o|Rowanbrook Primary School}}, {{a|Rowanbrook Lane, Portishead BS20 9ZZ}}`)),
}

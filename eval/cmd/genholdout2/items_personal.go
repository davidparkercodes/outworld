package main

var itemsPersonal = []def{
	personal("family-chat", "chat,lowercase,kinship", usr(`can you turn this into a simple schedule for the week? who picks up who and when

[{{d|2026-03-08|08/03/2026}}, 18:02:11] Mamá: {{p|ximena|nickname}} can you get {{p|+Mateo}} from futbol on tuesday? i have the dentist
[{{d|2026-03-08|08/03/2026}}, 18:02:40] {{p|ximena|first}}: sí. what time
[{{d|2026-03-08|08/03/2026}}, 18:03:05] Mamá: 5:30 at the field behind {{o|Colegio Benito Juárez|school}}. coach is {{p|+Profe Octavio|title-first}}, his number is {{ph|+52 55 5550 0111}} if they run late
[{{d|2026-03-08|08/03/2026}}, 18:04:19] {{p|chema|nickname}}: i can do thursday. also abuela's appointment moved to viernes 13, 9am, dr {{p|+Villaseñor|title-last}} again
[{{d|2026-03-08|08/03/2026}}, 18:04:51] {{p|ximena|first}}: is that the cardiologist or the eye one
[{{d|2026-03-08|08/03/2026}}, 18:05:30] {{p|chema|nickname}}: eye. tía {{p|+Lupita|kinship-first}} is driving her. {{p|+Mateo|lower|mateo}} don't forget your cleats this time
[{{d|2026-03-08|08/03/2026}}, 18:06:02] {{p|+Mateo}}: ok
[{{d|2026-03-08|08/03/2026}}, 18:09:44] Mamá: and somebody call your tía back she left me 3 voicemails. {{ph|55 5550 0193}}`)),

	personal("school-email", "school,child", usr(`Help me reply to this. I want to agree to the meeting but explain that the reading issue started after the classroom change, without sounding defensive.

From: {{p|deirdre|title-last|Ms. O'Callaghan}} <{{e|d.ocallaghan@stbrendans-ns.example}}>
Subject: {{p|+Oisín Achterberg|first|Oisín}}, a quick word
Date: {{d|2026-03-04|Wed, 4 Mar 2026}} 15:48

Dear {{p|robin|title-last|Mr. Achterberg}},

I wanted to get in touch about {{p|+Oisín Achterberg|first|Oisín}}. He is a lovely boy and well liked in Third Class, but since the mid-term break he has been reluctant to read aloud, and on Tuesday he was upset after the spelling test. Our learning support teacher, {{p|siobhan|title-last|Ms. Ní Bhraonáin}}, would like to do a short assessment with your permission.

Could you or {{p|+Oisín Achterberg|possessive|Oisín}}'s mum come in on Wednesday the 11th at 2.45? If that does not suit, ring the office on {{ph|020 915 5540}} and {{p|+Bernie|first}} will find another time.

Kind regards,
{{p|deirdre}} ({{p|deirdre|nickname}})
3rd Class, St. Brendan's NS`)),

	personal("patient-message", "medical,gov-id", usr(`Make this message to my doctor shorter and clearer. Keep every number.

To: Dr. {{p|arjun|last}}, Cardiology
From: {{p|earl}}, DOB {{d|1939-03-08|3/8/1939}}, MRN 00413017

Dr. {{p|arjun|last}}, this is {{p|earl|first}}. Since you changed the metoprolol on {{d|2026-02-17|Feb. 17th}} I have been dizzy most mornings. My daughter {{p|carol|first}} took my pressure on {{d|2026-03-01|3/1}} (98 over 61) and {{d|2026-03-03|3/3}} (104 over 58). I fell in the kitchen on Monday, did not hit my head. The pharmacy says the insurance will not cover the new one. My Medicare number is {{g|medicare|1EG4-TE5-MK73}} and the supplement is {{f|policy|SUP-00917733-02}}. Please call {{p|carol|possessive|Carol Ann}}'s phone not mine, I cannot hear it: {{ph|406-555-0131}}. Next visit is not until {{d|2026-04-21|April 21}}.`)),

	personal("flight-booking", "travel,caps,gov-id", usr(`Put this into a packing-list style itinerary with local times, and tell me if the connection in Doha is too tight.

BOOKING CONFIRMED  Ref: K7QM2X
Issued: {{d|2026-02-20|20FEB26}}

PASSENGERS
1. {{p|tomasz|airline|KOWALCZYKBRANDT/TOMASZ MR}}    ETKT 000-0000014201   PASSPORT {{g|passport|ZZ0001420}} POL  DOB {{d|1982-05-17|17MAY82}}
2. {{p|gosia|airline|WISNIEWSKA/MALGORZATA MS}}      ETKT 000-0000014202   PASSPORT {{g|passport|ZZ0001421}} POL  DOB {{d|1993-01-30|30JAN93}}
3. {{p|+Zofia Kowalczyk-Brandt|airline|KOWALCZYKBRANDT/ZOFIA MISS}} (CHD)   DOB {{d|2018-09-02|02SEP18}}

FLIGHTS
QX 264  WAW Warsaw 15:05   DOH Doha 22:40      {{d|2026-03-27|FRI 27MAR}}
QX 908  DOH Doha 23:35     DPS Denpasar 14:30  {{d|2026-03-28|SAT 28MAR}} (arrival)
QX 909  DPS Denpasar 18:10 DOH Doha 23:05      {{d|2026-04-10|FRI 10APR}}
QX 263  DOH Doha 01:55     WAW Warsaw 06:20    {{d|2026-04-11|SAT 11APR}}

CONTACT {{e|tomek.kb@example.com}}  {{ph|+48 22 555 01 99}}
PAYMENT VISA ending 3155 ({{f|card-visa|4000 0025 0000 3155}}), billed to {{p|tomasz|initial-last|T. Kowalczyk-Brandt}}
Driver on arrival: {{p|gede|first-only|Pak Gede}}, WhatsApp {{ph|+62 812 5550 177}}`)),

	personal("rental-booking", "travel,address,secrets", usr(`Write a short friendly message to the host asking for early check-in and whether the cot is still available.

Your reservation is confirmed!
{{o|Casa da Ribeira|property}} · hosted by {{p|ines|first}}
{{a|Rua das Flores 14, 2.º Esq., 4050-262 Porto, Portugal}}
Check-in {{d|2026-04-03|sex., 3 de abr. de 2026}} after 15:00 · Checkout {{d|2026-04-07|ter., 7 de abr. de 2026}} by 11:00
Guests: {{p|marieeve}}, {{p|jb}}, 1 infant

Getting in: the key safe is to the left of the green door, code {{s|door-code|2580#}}. Wifi "RibeiraGuest", password {{s|wifi-password|sardinha-azul-77}}.
Host phone: {{ph|+351 22 000 0142}}. If {{p|ines|first}} does not answer, her brother {{p|+Duarte}} lives downstairs.
Paid: 412,00 € with card ending 4444. Confirmation code HMX2QW9T.`)),

	personal("babysitter-note", "family,secrets,address", usr(`Can you format this as a one-page sheet I can print and stick on the fridge for the sitter?

sitter is {{p|jade|first}} ({{p|jade}}), Saturday 5pm to midnight
kids: {{p|+Kwame}} (7) and {{p|+Abena}} (3). {{p|+Abena|first}} is allergic to peanuts and eggs, epipen in the blue bag. {{p|+Kwame|first}} will tell you he is allowed to stay up till 10, he is not
we are at {{p|kobby|possessive|Kobby}}'s work thing at {{o|The Violet Room|venue}}, {{a|220 Queen Street West, Toronto, ON M5V 1Z4}}
me ({{p|+Akosua Asante-Boateng|first|Akosua}}): {{ph|416-555-0147}}. {{p|kobby|nickname}}: {{ph|647-555-0182}}
neighbour if you cant reach us: {{p|carol|title-last|Mrs. Whitehorse}} at number 16, {{ph|416-555-0103}}
pediatrician after hours line {{ph|1-888-555-0190}}, ask for dr {{p|priya|last}}
alarm code {{s|alarm-code|4471}} then OFF. wifi is "asante5g" pw {{s|wifi-password|Jollof&Waakye2019}}
our address in case you need to give it to 911: {{a|14 Marchmount Road, Toronto, ON M6G 2A9}}`)),

	personal("seating-plan", "event,names", usr(`Seat these 22 people at three round tables so that nobody sits next to their ex and each table has at least one person who speaks English and Portuguese. Constraints are in the notes.

Table hosts: {{p|joao}} and {{p|hunter}} (the couple)
Family: {{p|+Dona Aparecida|title-first}} (grandmother, Portuguese only), {{p|+Tio Beto|kinship-first}}, {{p|rafa}}, {{p|ines|first}} and her wife {{p|nadine|first}}, {{p|+Grandpa Walt|kinship-first}}, {{p|cliff|first}} and {{p|daisy|first}} {{p|cliff|last}} (née {{p|daisy|last}})
Friends: {{p|wulan}}, {{p|zarghona}}, {{p|kaho|western-order|Felix Tsang}}, {{p|yael}}, {{p|mido|nickname}}, {{p|sasha|nickname}}, {{p|reed}}, {{p|paige}}, {{p|sage}}, {{p|jiwoo|first}}, {{p|kainoa}}, {{p|shandiin}}

Notes: {{p|reed|first}} and {{p|paige|first}} split up in January. {{p|sage|first}} is vegan. {{p|mido|nickname}} speaks Portuguese. {{p|kaho|nickname}} must be near {{p|jiwoo|first}}. {{p|+Grandpa Walt|first|Walt}} needs a seat near the door. {{p|wulan|nickname}} and {{p|zarghona|first}} do not know anyone else.`)),

	personal("landlord-texts", "chat,address", usr(`I need to send a formal written notice now. Draft it from these texts, with dates.

{{d|2026-02-09|Feb 9}}, 8:14 AM  me: Hi {{p|lance|title-last|Mr Corbin}}, the heating in 3B is out again. It is 12 degrees in here.
{{d|2026-02-09|Feb 9}}, 11:52 AM  {{p|lance|first}} (landlord): will send {{p|rusty|nickname}} tomorrow
{{d|2026-02-11|Feb 11}}, 6:30 PM  me: Nobody came. My daughter has asthma.
{{d|2026-02-12|Feb 12}}, 9:03 AM  {{p|lance|first}} (landlord): {{p|rusty|lower-nickname|rusty}} says he knocked nobody home
{{d|2026-02-12|Feb 12}}, 9:05 AM  me: I work from home. Nobody knocked. Have him call me {{ph|628-555-0114}}
{{d|2026-02-19|Feb 19}}, 7:41 AM  me: Day 10. I am buying a space heater and deducting it.
{{d|2026-02-19|Feb 19}}, 7:58 AM  {{p|lance|first}} (landlord): do not deduct anything talk to my wife {{p|+Marlene Corbin|first|Marlene}} she does the books {{e|marlene@corbinroofing.example}}

My name for the letter: {{p|tahmina}}. Address: {{a|1420 Fulton Street, Apt 3B, San Francisco, CA 94117}}. Lease started {{d|2024-08-01|August 1, 2024}}.`)),

	personal("relationship-advice", "advice,common-word-name,lowercase", usr(`i need an outside opinion. my boyfriend {{p|reed|lower-first|reed}} and i have been together two years. last weekend at his sister {{p|+Hazel|lower|hazel}}'s birthday he told his mom, in front of everyone, that we are "not in a hurry" to move in together. we signed a lease in january. when i asked him about it he said i should not read into it, and then his friend {{p|hunter|lower-first|hunter}} texted me (!!) saying "{{p|reed|first}} is just scared, give him grace". i dont even know how {{p|hunter|first}} has my number.

my sister {{p|+Autumn}} says leave. my therapist (dr. {{p|shirin|last-part|Tavakoli}}) says ask him one direct question and listen. what is the one question?

for context i'm 31, he's 34, and yes i know i should not read his texts but his phone was on the reed diffuser shelf right next to me and it lit up.`)),

	personal("obituary", "writing,dates,places", usr(`Please polish this obituary for the newspaper. Keep it under 180 words. The paper wants dates written out.

{{p|+Wincentyna Kowalczyk|full|Wincentyna "Wicia" Kowalczyk}} (née {{p|+Wincentyna Kowalczyk|maiden|Brandt}}), {{d|1931-11-02|2.11.1931}} to {{d|2026-03-01|1.03.2026}}

Our mother and grandmother {{p|+Wincentyna Kowalczyk|nickname|Wicia}} died peacefully at home in {{l|Gdańsk}} on {{d|2026-03-01|the first of March}}, aged 94. Born in {{l|Lwów}}, she came to the coast in 1946 and worked for thirty years as a seamstress at the shipyard. She is survived by her son {{p|tomasz}}, daughter-in-law {{p|gosia|first}}, granddaughter {{p|+Zofia Kowalczyk-Brandt|first|Zosia}}, and her sister {{p|+Bronisława}}. She was predeceased by her husband {{p|+Henryk Kowalczyk|first|Henryk}} in 2009.

Funeral mass at {{o|St. Bridget's Basilica|place}} on {{d|2026-03-06|Friday 6 March}} at 11:00. No flowers; donations to the hospice. Contact for the family: {{p|tomasz|nickname}}, {{ph|+48 58 555 01 42}}.`)),

	personal("resume", "cv,address", usr(`Rewrite my CV summary and tighten the bullets. Target: senior product designer roles in Amsterdam.

{{p|selam|caps}}
{{a|Keizersgracht 14-II, 1015 CX Amsterdam}} · {{ph|+31 6 5550 0142}} · {{e|selam.gebremariam@example.com}} · portfolio.example.com/{{p|selam|url-slug|selamg}}
Born {{d|1992-01-07|7 January 1992}}, Addis Ababa · Dutch residence permit (type II) no. {{g|residence-permit|V000142AB}}

EXPERIENCE
Product Designer, {{o|Tidepool Labs|company}} (2021 to now). Led checkout redesign with {{p|ines}} (PM); conversion up 11%.
UX Designer, {{o|Savannah Pay|company}}, {{l|Nairobi}} (2017 to 2021). First design hire. Reported to {{p|wanjiru}}.

EDUCATION
BA Graphic Design, {{o|Addis Ababa University|company}}, 2014

REFERENCES
{{p|ines}}, {{e|ines.albuquerque@tidepool-labs.example}}, {{ph|+351 22 000 0177}}
{{p|wanjiru}} ("{{p|wanjiru|nickname}}"), {{e|wanjiru.mwangi@savannahpay.example}}`)),

	personal("tax-question", "finance,gov-id", usr(`I got two forms that disagree. Which number do I report, and do I need to ask for a corrected one?

Form 1099-NEC (2025)
PAYER: {{o|Brightquay Media LLC|company}}, {{a|500 Example Plaza, Suite 1400, Austin, TX 78701}}, TIN {{g|ein|00-0142017}}
RECIPIENT: {{p|april}}, {{a|77 Kapiolani Blvd, Apt 14F, Honolulu, HI 96813}}, TIN {{g|ssn|900-55-0142}}
Box 1 Nonemployee compensation: 18,450.00

Their year-end statement emailed {{d|2026-01-28|1/28/2026}} says 16,950.00 across 9 invoices, last one paid {{d|2025-12-19|12/19/25}}. The difference is one invoice for 1,500 that they paid on {{d|2026-01-05|Jan 5}}. Their bookkeeper is {{p|june|first}} ({{e|ap@brightquay.example}}, {{ph|512-555-0166}}). My refund goes to routing {{f|routing|110000000}} account {{f|account|000111222333}} if that matters. Filing deadline is {{d|2026-04-15|April 15}}.`)),

	personal("carpool", "family,phones,table", usr(`Make a fair 6-week rotation from this roster. Each family drives once every three weeks; siblings count as one family.

U11 Thunder, spring roster ({{d|2026-03-21|first practice Sat 21 March}})
Player            Parent                 Phone           Notes
{{p|+Lani Kahananui}}    {{p|kainoa|first}}                 {{ph|(808) 555-0173}}  has a 7-seater
{{p|+Noa Kahananui}}     {{p|kainoa|first}}                 {{ph|(808) 555-0173}}  Lani's brother
{{p|+Theo Nakashima}}    {{p|april}}        {{ph|808-555-0120}}    cannot drive Wednesdays
{{p|+Mele Leota}}        {{p|tavita}} {{ph|808-555-0155}}    text only
{{p|+Pua Okafor}}        {{p|ray|first}} or {{p|+Leilani Okafor|first|Leilani}}         {{ph|808.555.0188}}    {{p|ray|first}} works nights, call {{p|+Leilani Okafor|first|Leilani}}
{{p|+Kai Batbayar}}      {{p|enkhtuya|nickname}}                 {{ph|+1 808 555 0109}} new this season
Coach: {{p|+Coach Makoa|title-first}}, {{e|coach.makoa@example.org}}
Field: {{o|Ray Kroc Park|place}} field 2 (not named after Pua's dad, he checked)`)),

	personal("insurance-appeal", "medical,letter,financial", usr(`Strengthen this appeal letter. Cite that the surgeon was in network on the date of service.

{{p|nastya|western|Anastasia Volkonskaya}}
{{a|2140 Sheridan Road, Unit 4, Evanston, IL 60201}}
{{d|2026-03-05|March 5, 2026}}

Appeals Department, {{o|Lakeshore Mutual Health|company}}
Re: Member ID {{f|member-id|LMH-881420073}}, Claim 26-0041-7781, patient {{p|nastya|formal|VOLKONSKAYA, ANASTASIA S}}, DOB {{d|1988-12-14|12/14/1988}}

To whom it may concern:

I am appealing the denial dated {{d|2026-02-20|February 20, 2026}} for my surgery on {{d|2026-01-13|January 13, 2026}} with Dr. {{p|harpreet}}. Your representative "{{p|destiny|first}}" told me by phone on {{d|2026-02-24|2/24}} that Dr. {{p|harpreet|last}} left the network on {{d|2026-02-01|February 1}}, which is after my date of service. My husband, {{p|+Mikhail Volkonsky}}, is the policyholder; his employee ID is 20417 and his SSN on file ends in 0142 ({{g|ssn|900-14-0142}}).

Please reprocess. You can reach me at {{ph|847-555-0119}} or {{e|nastya.v@example.net}}.`)),

	personal("genealogy", "family-history,dates,names", usr(`Does this timeline make sense? I think two different men are being merged. Point out the contradictions.

- {{p|+Padraig Ó Dubhghaill|full|Pádraig Ó Dubhghaill}}, b. {{d|1871-03-17|17 Mar 1871}}, {{l|Clifden}}; emigrated {{d|1889-05-04|4 May 1889}} on the SS Polynesian
- {{p|+Padraig Ó Dubhghaill|anglicized|Patrick Doyle}}, laborer, naturalized {{l|Boston}} {{d|1896-10-12|Oct. 12, 1896}}; wife {{p|+Brigid Doyle|maiden|Bridget Conneely}}
- {{p|+Padraig Ó Dubhghaill|anglicized|Patk. Doyle}}, 1900 census, age 31, born "Mar 1869"
- {{p|+Padraig Ó Dubhghaill|anglicized|P. J. Doyle}}, d. {{d|1931-01-29|29th January 1931}}, {{l|Dorchester}}, age "58"
- daughter {{p|+Nora Doyle|full|Honora (Nora) Doyle}}, b. {{d|1898-07-02|July 2, 1898}}, m. {{p|+Aloysius Featherstonehaugh}} 1921
- their grandson is my father-in-law, {{p|jack}}, b. {{d|1957-08-30|8/30/1957}}

The family story is that {{p|+Padraig Ó Dubhghaill|first|Pádraig}} changed the name at the dock. {{p|jack|possessive}} cousin {{p|deirdre|nickname}} has the naturalization paper.`)),

	personal("visa-form", "travel,gov-id,form", usr(`Check my answers for the e-visa form before I submit. Anything inconsistent?

Surname: {{p|caglar|caps-last|YILDIRIM}}
Given names: {{p|caglar|caps-ascii-first|CAGLAR}}
Name in native script / as in passport: {{p|caglar}}
Date of birth (DD/MM/YYYY): {{d|1989-06-21|21/06/1989}}
Place of birth: {{l|İzmir}}
Passport number: {{g|passport|U00014277}}   Issued: {{d|2021-09-14|14/09/2021}}   Expires: {{d|2031-09-13|13/09/2031}}
National ID (T.C. Kimlik No): {{g|tc-kimlik|10000000146}}
Home address: {{a|Atatürk Caddesi No: 14 D: 7, Alsancak, 35220 Konak/İzmir}}
Phone: {{ph|+90 232 555 01 42}}    Email: {{e|caglar.yildirim@example.com}}
Arrival: {{d|2026-05-02|02/05/2026}}   Departure: {{d|2026-05-16|16/05/2026}}
Host in country: {{p|sbu}}, {{a|14 Florida Road, Morningside, Durban 4001}}, {{ph|+27 31 555 0142}}
Mother's maiden name: {{p|+Öztürk|maiden}}`)),

	personal("spanglish-family", "chat,spanish,code-switch", usr(`translate this to english for my husband, keep the tone

{{p|+Tía Chayo|kinship-first}}: mija ya llegó el paquete?? el que te mandó tu prima {{p|montse|nickname}} desde {{l|Girona}}
yo: no tía, todavía no. tracking says el jueves 12
{{p|+Tía Chayo|kinship-first}}: ay dios. y {{p|angel|first}} cómo sigue de la rodilla
yo: better, lo operan el {{d|2026-03-24|24 de marzo}} en el {{o|Hospital San Lucas|place}}. el doctor {{p|+Batista|title-last}} no, otro, el dr. {{p|agus|last}}
{{p|+Tía Chayo|kinship-first}}: dile a {{p|angel|nickname|Angelito}} que le mando un caldo. sigues en la misma casa? {{a|1402 Calle Loíza}}?
yo: sí tía la misma
{{p|+Tía Chayo|kinship-first}}: y contesta el teléfono!! te marqué al {{ph|787 555 0164}} como diez veces`)),

	personal("vet-message", "pets,pet-name", usr(`Summarize what I need to do before the appointment.

Hello {{p|daisy|title-last|Ms. Mukherjee}},

This is a reminder that {{n|pet-name|Basil}} (canine, 6 y, {{p|daisy|last}}) is booked for a dental cleaning with Dr. {{p|pearl|last-part|Abernathy}} on {{d|2026-03-19|Thursday, March 19}} at 8:15 AM. No food after 10 PM the night before; water is fine. Please bring {{n|pet-name|Basil}}'s current medications.

Our records also show that {{n|pet-name|Sage}} (feline, 11 y) is overdue for a rabies booster. You can book both online or call {{ph|(503) 555-0127}}.

Estimate sent to {{e|daisy.mukherjee@example.com}}. Card on file ending 1117.

{{o|Rowan & Ash Veterinary|company}}, {{a|4410 SE Hawthorne Blvd, Portland, OR 97215}}`)),

	personal("class-parents", "chat,possessive,kinship", usr(`who has agreed to bring what? make a list, and flag who has not answered

Class 4B parents 🎒
{{p|akane|first}} ({{p|+Ren Mizushima|possessive|Ren}}'s mum): bake sale is fri 20th!! need 6 trays. i'll do matcha cookies
{{p|+Kofi|possessive}}'s dad: i can do 2 trays of meat pies. is {{p|+Ren Mizushima|first|Ren}} still nut free?
{{p|akane|first}} ({{p|+Ren Mizushima|possessive|Ren}}'s mum): yes thanks {{p|kobby|first}}!
{{p|fatima|nickname}}: msemen from me. {{p|+Yasmine El Idrissi|first|Yasmine}} says {{p|+Miss Oyelaran|title-last}} wants labels with ingredients
{{p|ria|nickname}} 🇬🇷: koulourakia. also who is collecting the money, {{p|fatima|lower-first|fatima zahra}} did it last time
{{p|+Chidi|possessive}}'s mum: I can collect. Bank transfer to {{f|sort-code|20-00-00}} {{f|account|55779911}} ref BAKESALE or cash on the day
{{p|akane|first}} ({{p|+Ren Mizushima|possessive|Ren}}'s mum): @{{p|hamish|first}} @{{p|+Dmitri}} ??`)),

	personal("personal-logins", "secrets,notes", usr(`I keep this in a notes app and I know that is bad. Convert it to a CSV I can import into a password manager (columns: title, username, password, url, notes).

wifi home: {{s|wifi-password|tangerine-otter-4410}}
router admin: admin / {{s|password|Hx7#pL2mQ9!z}}
bank ({{o|First Example Credit Union|company}}): user {{p|bobby|handle|bobbyt61}}, pw {{s|password|Tannenbaum$1961}}, phone PIN {{s|pin|8841}}
  security q: mothers maiden name = {{p|+Lindqvist|maiden}}; first pet = {{n|pet-name|Major}}
email: {{e|robert.tannenbaum3@example.com}} / {{s|password|correct-horse-battery-staple}}
  backup codes: {{s|recovery-code|4417 2290}}, {{s|recovery-code|8830 1175}}, {{s|recovery-code|6021 9943}}
streaming: shared with {{p|+Skyler|first}}, login {{e|skyler.t@example.com}} pw {{s|password|popcorn4ever!}}
crypto wallet seed (do NOT lose): {{s|seed-phrase|ribbon cactus drift oxygen lamp velvet tunnel moral echo flight harbor quiz}}
garage keypad {{s|door-code|1961*}}
SSA account: {{g|ssn|666-45-6789}}, created {{d|2019-04-02|4/2/2019}}`)),

	personal("bill-split", "chat,finance,handles", usr(`work out who owes whom from this, minimum number of transfers

flat 4 bills, february
rent 2,400 (paid by {{p|sage|first}}), electric 186.40 ({{p|basil|first}}), internet 55 ({{p|jiri|ascii-first|Jiri}}), groceries: {{p|basil|first}} 212.15, {{p|jiri|first}} 97.80, {{p|sage|first}} 0 ("i was away till the 16th")
{{p|august|first}} moved in {{d|2026-02-15|15 Feb}} so he pays half of everything except groceries, he bought his own
pay {{p|sage|first}} at @{{p|sage|handle|sage-delacroix}} or e-transfer {{e|sage.delacroix@example.com}}
{{p|basil|first}} takes bank transfer only: {{f|sort-code|20-00-00}} / {{f|account|10293847}}
{{p|jiri|first}}: IBAN {{f|iban|CZ65 0800 0000 1920 0014 5399}}
{{p|august|first}} ({{p|august|handle|august.w}}) says he will pay everyone back "in august", which, no. by {{d|2026-03-15|march 15}} please`)),

	personal("recipe-scaling", "negative,common-words,measurements", usr(`Scale this to feed 14 and convert to metric. Also, can I swap the herbs?

Roast squash with {{n|common-word|basil}} and {{n|common-word|sage}} (serves 4-6)
- 1 large squash, about 2-3 lb
- 3/4 cup {{n|common-word|olive}} oil
- 1/2 bunch fresh {{n|common-word|basil}}, 6-8 {{n|common-word|sage}} leaves, 2 bay leaves
- 1 tbsp {{n|common-word|rosemary}}, a pinch of {{n|common-word|ginger}}, cracked {{n|common-word|pepper}}
- a handful of {{n|common-word|hazel}} nuts, 1 {{n|common-word|clementine}}, zest only
- {{n|common-word|pearl}} barley to serve, 1 1/2 cups

Roast at 425 F for 35-40 min. My oven runs hot, so I usually {{n|common-word|mark}} it at 400 and {{n|common-word|will}} check at 30. The original says to rest it under foil "for {{n|common-word|grace}}", which I think is a typo for "for good measure". Ratio of oil to squash should stay 1:4.`)),

	personal("trip-ideas", "negative,places,months", usr(`We have two weeks in {{n|month|May}} or {{n|month|June}}, maybe late {{n|month|August}}. Two adults, one toddler, no car. Compare: {{l|Lisbon}} and {{l|Porto}} by train; {{l|Kyoto}} and {{l|Nara}}; the {{l|Amalfi}} coast from {{l|Salerno}}; {{l|Victoria}} and the {{l|Gulf Islands}}; {{l|St. John's}} and the {{l|Bonavista}} peninsula; {{l|Santa Fe}} and {{l|Taos}}.

Things that matter: flights under 12 hours from {{l|Chicago}}, a washing machine, somewhere to {{n|common-word|wade}} that is not a {{n|common-word|cliff}} edge, and no heat above 30 C. We did {{l|Florence}} last year and it was too crowded. {{n|common-word|April}} is out because of work. Will a {{n|common-word|rowan}}-lined campsite in {{l|Skye}} be midge season by then?`)),

	personal("homework-history", "negative,public-figure,fictional", usr(`My daughter has to write 300 words comparing how two rulers held power. She picked {{n|public-figure|Cleopatra}} and {{n|public-figure|Genghis Khan}}. Give her an outline, not the essay.

Her second assignment is for English: is {{n|fictional|Hamlet}} indecisive or careful? She wants to contrast him with {{n|fictional|Elizabeth Bennet}} and, somehow, {{n|fictional|Frodo Baggins}}. Her teacher suggested {{n|fictional|Odysseus}} instead and mentioned that {{n|public-figure|Jane Austen}} never married while {{n|public-figure|William Shakespeare}} left his wife his second-best bed, which my daughter now finds more interesting than either book.

For maths she needs one fact each about {{n|public-figure|Euclid}}, {{n|public-figure|Isaac Newton}}, {{n|public-figure|Emmy Noether}} and {{n|public-figure|Srinivasa Ramanujan}}. Keep it short.`)),
}

package main

var itemsPersonalB = []def{
	personal("rental-application", "rental,application,references",
		usr(`The agent wants this application "tidied up" before it goes to the landlord. Fix the wording, keep every fact, and tell me if I am oversharing.

RENTAL APPLICATION: {{a|14B Kelburn Parade, Kelburn, Wellington 6012}}
Move-in requested: {{d|2026-08-01|1 August 2026}}, 12 months

APPLICANT 1
Name: {{p|aroha}}
Date of birth: {{d|1992-03-04|4 March 1992}}
Phone: {{ph|021 555 0137}}   Email: {{e|aroha.nw@kiwimail.example}}
Current address: {{a|3/88 Aro Street, Aro Valley, Wellington 6021}} (since {{d|2023-02-10|10/02/2023}})
Driver licence: {{g|nz-driver-licence|DK482917}} version 041
IRD number (for the bond form): {{g|nz-ird|084-271-934}}
Employer: {{o|Te Whatu Ora}}, registered nurse, $84,000. Manager: {{p|matilda}}, {{ph|04 555 0172}}

APPLICANT 2
Name: {{p|hemi}} (partner)
Date of birth: {{d|1990-11-22|22 November 1990}}
Phone: {{ph|+64 21 555 0119}}
Employer: {{o|Corvane Freight}} (NZ branch), lane manager, $97,500
Passport: New Zealand, {{g|nz-passport|LM482917}}

REFERENCES
Current landlord: {{p|victor}}, {{ph|027 555 0188}}, {{e|vic.oseitutu@kiwimail.example}}. Rent $640/wk, never late.
Personal: {{p|penelope}} (former colleague of {{p|hemi|possessive}}'s, recently moved to {{l|Auckland}}), {{ph|+64 22 555 0141}}

OTHER
Pets: one cat, {{n|pet-name|Biscuit}}, desexed, 6 years. Reference from {{p|victor|nickname}} available.
Bank for rent payments: {{o|Kiwibank}} {{f|nz-account|38-9017-0482917-00}}, account name {{p|aroha|initial-last|A Ngata-Williams}}
Emergency contact: my mother, {{p|+Mere Ngata|full}}, {{l|Gisborne}}, {{ph|06 555 0114}}
Anything else: {{p|hemi|first}} works some nights. We are quiet. I have a restricted nut allergy and would ask that the previous tenant's almond tree is not counted as my responsibility.`)),

	personal("wedding-guest-list", "event,planning",
		usr(`Seat these people at three tables of six. Constraints are in the notes column. Then draft the text for the two place cards I flagged.

Guest list v7 (final-final), wedding of {{p|thais}} and {{p|joaopedro}}, {{d|2026-10-10|10/10/2026}}, {{l|Petrópolis}}

Nome | Lado | Notas
{{p|josefa}} | noiva | {{p|josefa|honorific|Vó Zefa}}, 88, cadeira de rodas, perto da saída
{{p|beatriz}} | noiva | madrinha. não sentar com o ex ({{p|guilherme|nickname}})
{{p|guilherme}} | noivo | padrinho. ver acima
{{p|mateus}} | noivo | primo do {{p|joaopedro|nickname}}, vem de {{l|Bristol}}
{{p|chiara}} | noiva | amiga de {{l|Roma}}, vegetariana, não fala português
{{p|ludovica}} | noiva | acompanhante da {{p|chiara|first}}
{{p|valentina}} | noivo | colega do {{p|joaopedro|first}}, traz o marido {{p|cristobal|first}}
{{p|cristobal}} | noivo |
{{p|florencia}} | noiva | fala alto. longe da {{p|josefa|nickname}}
{{p|andres}} | noivo | "{{p|andres|nickname}}", alérgico a camarão
{{p|ximena}} | noiva | fotógrafa nas horas vagas, quer mesa com boa luz
{{p|fernando}} | noivo | chefe do {{p|joaopedro|nickname}}. sentar com gente simpática
{{p|xiadani}} | noivo | acompanhante do {{p|fernando|nickname}}
{{p|wayra}} | noiva |
{{p|genevieve}} | noiva | amiga de {{l|Montréal}}, place card em francês: "{{p|genevieve|title-last|Tremblay-Ouellet}}, Mme"
{{p|solene}} | noiva | idem, place card: "Mlle {{p|solene|first}} {{p|solene|caps-last}}"
{{p|inaki}} | noivo | não bebe
{{p|maricarmen}} | noivo | mãe do {{p|inaki|first}}, chama de {{p|maricarmen|nickname}}, nunca "{{p|maricarmen|name-part|María}}"

RSVP pendente: {{p|andres|name-part|Andrés}} (mandei msg no {{ph|+57 310 555 0164}}). Buffet fecha a lista dia {{d|2026-09-25|25/09}}. Contato do buffet: dona {{p|+Marlene|first}}, {{ph|(24) 5550-0127}}, {{e|eventos@serradourada.example}}.`)),

	personal("estate-question", "legal,family,estate",
		usr(`My grandfather died without a will and my aunt says the house is "obviously hers". I need to understand where we stand before the family meeting. The facts as I know them:

Opa, {{p|willem}} ({{p|willem|honorific|Opa Wim}}), died on {{d|2026-02-19|19 februari 2026}} in {{l|Zwolle}}. He was 91. Oma died in 2014.
He had two children: my mother, who died in 2020, and my aunt {{p|anouk}}.
My mother had two children: me ({{p|femke}}) and my brother {{p|maarten|first}}. Yes, I know {{p|maarten|first}} carries his father's surname, {{p|maarten|last}}, and I carry my mother's, long story.
The house is at {{a|Assendorperstraat 212, 8012 CE Zwolle}}. No mortgage. Tante {{p|anouk|first}} has lived there with Opa since 2019 as his carer and says he "promised" it to her.
There is a savings account at {{o|ING|company}}, about 61,000 euro, IBAN {{f|nl-iban|NL07 INGB 0005 2931 84}}, and a small pension from {{o|Philips|company-eponym}}.
The notaris handling it is mr. {{p|joris|title-last|de Bruijn-Ackermans}} in {{l|Zwolle}}, {{ph|038 555 0142}}. He asked for everyone's BSN. Mine is {{g|nl-bsn|304561824}}.
{{p|joris|first}} {{p|joris|last}} is also, awkwardly, my ex.

Questions: under Dutch intestacy, what share do {{p|maarten|first}} and I get together versus {{p|anouk|first}}? Does her caring for {{p|willem|nickname}} for seven years count for anything legally? The family meeting is on {{d|2026-03-21|zaterdag 21 maart}} and I want to walk in knowing the law, not arguing about what Opa would have wanted.`)).on("2026-02-19", "willem"),

	personal("rail-hotel-itinerary", "travel,itinerary",
		usr(`Put this trip into a day-by-day plan and tell me where the connections are too tight. Confirmations pasted as received.

=== {{o|Deutsche Bahn|company}} ===
Auftrag {{n|order-number|912 483 027 715}}, gebucht am {{d|2026-07-02|02.07.2026}}
Reisende: {{p|emine}}, {{p|matthias}}, {{p|elif}} (12 J.), BahnCard 25 Nr. {{x|loyalty-card|7081 4100 2948 3175}}
{{d|2026-08-03|Mo, 03.08.2026}}  {{l|Hamburg}} Hbf ab 07:28, ICE 587, {{l|München}} Hbf an 13:41, Wagen 9, Plätze 61 62 64
{{d|2026-08-03|Mo, 03.08.2026}}  {{l|München}} Hbf ab 14:17, RJX 65, {{l|Wien}} Hbf an 18:30

=== {{o|Hotel Zum Goldenen Hirschen}} ===
Sehr geehrte Frau {{p|emine|title-last|Yıldırım-Hoffmann}},
wir bestätigen Ihre Reservierung Nr. {{n|identifier|GH-26-08-0417}}: Familienzimmer, Anreise {{d|2026-08-03|3. August}}, Abreise {{d|2026-08-07|7. August 2026}}, 3 Personen.
Adresse: {{a|Schönbrunner Straße 118, 1050 Wien}}. Late check-in bitte telefonisch: {{ph|+43 1 555 0163}}.
Die Kreditkarte endend auf 0005 wurde als Garantie hinterlegt.
Mit freundlichen Grüßen, {{p|gesine|initial-last|G. Pfannenstiel}}, Rezeption

=== {{o|ÖBB|company}} Nightjet ===
Booking code {{n|identifier|NJ-8842-KX}}
{{d|2026-08-07|07 Aug 2026}} {{l|Wien}} Hbf 20:13 to {{l|Venezia}} S. Lucia 08:24 (+1), sleeper, 3 berths
Passenger names as on ID: {{p|emine|ascii-fold|EMINE YILDIRIM-HOFFMANN}}, {{p|matthias|caps|MATTHIAS HOFFMANN-REUTER}}, {{p|elif|ascii-fold|ELIF YILDIRIM-HOFFMANN}}
ID numbers for the border: {{g|de-id-card|L01X00T47}} ({{p|emine|initials|EYH}}), {{g|de-id-card|T22000129}} ({{p|matthias|initials|MHR}}), {{p|elif|first}} travels on a Kinderreisepass {{g|de-passport|C01X00T47}}

=== airbnb, {{l|Venezia}} ===
Host {{p|ludovica|nickname}}: "Ciao {{p|matthias|nickname}}! Check-in dalle 15, le chiavi sono dal bar sotto casa, chiedi di {{p|gianmarco|nickname}}. Indirizzo: {{a|Fondamenta della Misericordia 2536, 30121 Venezia}}. Il mio numero: {{ph|+39 041 555 0185}}."
{{d|2026-08-08|8 Aug}} to {{d|2026-08-12|12 Aug}}. Then home by plane, not booked yet. Oma {{p|hannelore|nickname}} picks us up, she needs the flight number by the {{d|2026-07-20|20th of July}}.`)).on("2026-08-03", "emine", "2026-08-07", "emine", "2026-08-08", "matthias"),

	personal("pharmacy-message", "medical,prescription",
		usr(`The pharmacy app sent me this and the dose looks different from what the doctor said. What should I ask them? I am not asking for medical advice, just the right questions.

{{o|Apotheke am Fleet}}, {{l|Hamburg}}
Ihre Vorbestellung ist abholbereit.

Patientin: {{p|hannelore}}, geb. {{d|1941-05-17|17.05.1941}}
Versichertennummer: {{g|de-kvnr|H482917305}} ({{o|AOK Nord}})
Rezept vom {{d|2026-06-11|11.06.2026}}, ausgestellt von Dr. med. {{p|friedhelm}}, {{a|Eppendorfer Landstraße 77, 20249 Hamburg}}

1x Metoprololsuccinat {{n|measurement|47,5 mg}} Retardtabletten, 100 St., 1-0-0
1x Apixaban {{n|measurement|2,5 mg}}, 60 St., 1-0-1
1x Pantoprazol {{n|measurement|20 mg}}, 30 St., bei Bedarf

Zuzahlung: 15,00 EUR. Abholung bis {{d|2026-06-19|19.06.2026}}, Abholcode {{s|pickup-code|4471-KX}}.
Abholung durch Dritte nur mit Vollmacht. Bevollmächtigt laut Kundenkonto: {{p|matthias}} (Sohn), Tel. {{ph|0171 39200 47}}.

What the doctor told my mother on the phone, according to her: "the heart tablet goes up to 95". Dr. {{p|friedhelm|title-last|Oberländer}} is on holiday until {{d|2026-06-29|29. Juni}} and the practice nurse, Schwester {{p|+Annegret|first}}, said to ask the pharmacy. Mutti is {{p|hannelore|nickname}} to everyone, in case they ask who called.`)),

	personal("visa-form-help", "immigration,form",
		usr(`I am filling in the online visitor visa form for my mother-in-law and the system keeps rejecting the name fields. How should I enter her name? Details from her documents:

Passport (Republic of Senegal): Surname {{p|+Ndèye Fatou Diallo|caps-last|DIALLO}}, Given names {{p|+Ndèye Fatou Diallo|caps-first|NDEYE FATOU}}. No. {{g|sn-passport|A02841937}}, issued {{d|2023-05-09|09 MAI 2023}} in {{l|Dakar}}, expires {{d|2028-05-08|08 MAI 2028}}.
Birth certificate: {{p|+Ndèye Fatou Diallo|full}}, née le {{d|1958-01-01|01/01/1958}} à {{l|Saint-Louis}} (the day and month are administrative, she does not know her real birthday).
National ID: {{g|sn-cni|1 751 1958 00417}}
Everyone calls her {{p|+Ndèye Fatou Diallo|honorific|Yaay Fatou}}.

She is visiting us, {{p|aminata}} and {{p|etienne}}, at {{a|27 rue des Capucins, 69001 Lyon}}, from {{d|2026-12-15|15 décembre 2026}} to {{d|2027-02-28|28 février 2027}}. Her son {{p|mamadou}} ({{p|mamadou|nickname}}), my husband's brother-in-law, will travel with her; his French residence card is {{g|fr-titre-sejour|9917204481}}.

The form has: "Nom de famille", "Prénom(s)", "Nom de naissance", and the validation message says "caractères non autorisés" when I type the è. It also refuses a single-word given name for {{p|mamadou|first}} if I leave "deuxième prénom" empty.

Sponsor details they want: {{p|etienne|caps-last|LEFÈVRE-MARCHAND}} {{p|etienne|first}}, numéro de sécurité sociale {{g|fr-nir|1 82 11 69 123 217 38}}, employer {{o|Hospices Civils de Lyon}}, phone {{ph|06 39 98 41 27}}.`)),

	personal("vet-appointment", "pets,appointment",
		usr(`Summarise what the vet wants me to do and by when. Two animals, do not mix them up.

{{o|Kelburn Veterinary Clinic}}
{{a|102 Upland Road, Kelburn, Wellington 6012}} · {{ph|04 555 0161}}

Visit summary, {{d|2026-05-06|6 May 2026}}
Owner: {{p|aroha}} (client no. {{n|identifier|40417}})
Vet: Dr {{p|rajesh|initial-last|R. Subramaniam}}

Patient 1: {{n|pet-name|Biscuit}}, domestic shorthair, F(S), 6 y, {{n|measurement|4.8 kg}}
Presenting: vomiting x3 since Sunday. Bloods normal. Likely dietary (owner reports {{n|pet-name|Biscuit}} ate part of a houseplant, possibly a {{n|common-word|lily}}; if it was a true lily this is urgent, please check the plant tonight and ring us).
Plan: bland diet 5 days, recheck if vomiting continues past Friday.

Patient 2: {{n|pet-name|Jasper}}, Labrador cross, M(N), 11 y, {{n|measurement|31.2 kg}}
Presenting: stiffness, worse after the walk up {{l|Mount Victoria|place-eponym}}. Hips radiographed.
Plan: start meloxicam {{n|measurement|1.5 mg/ml}}, dose by weight on the label, with food. Weight target {{n|measurement|29 kg}}. Recheck {{d|2026-06-03|3 June}}.
Note: {{n|pet-name|Jasper}} is registered to {{p|hemi}}, who is listed as second contact ({{ph|+64 21 555 0119}}). {{p|hemi|first}} authorised treatment by phone.

Invoice {{n|identifier|KV-118203}}: $412.60, paid by EFTPOS.
Next vaccinations due: {{n|pet-name|Biscuit}} {{n|month|August}}, {{n|pet-name|Jasper}} {{n|month|November}}.
Dr {{p|rajesh|first}} is away from {{d|2026-05-18|18 May}}; Dr {{p|+Tavita Seiuli|title-last|Seiuli}} will cover.`)),

	personal("roommate-advice", "advice,lowercase,typos",
		usr(`ok i need an outside opinion bc im too angry to think

so my flatmate {{p|jiyeon|lower-first}} (we call her {{p|jiyeon|nickname|jiji}}) has been subletting her room on weekends without telling me or the landlord. i found out bc a guy called {{p|+Dong-min|lower-first}} showed up saturday w a suitcase ({{q|tall guy, red suitcase, works at the bubble tea place downstairs}}) and a code for our door. OUR DOOR CODE. its {{s|door-code|4471#}} and now a stranger has it

i texted her and she said "its fine {{p|donghyun|lower|baek dong-hyun}} is my cousins friend" which, first of all, the guy said his name was {{p|+Dong-min|lower-first}} not {{p|donghyun|lower-first}} so who is {{p|donghyun|lower-first}}, and second its not fine

the lease is in both our names. landlord is mr {{p|kaho|lower-last}} ({{p|kaho|lower|cheung ka-ho}}), hes nice but hes strict about this, its literally clause 9. his managing agent is {{p|weiling|lower|tan wei ling}} at {{ph|6555 0142}} / {{e|weiling.tan@lionrockproperty.example}}

things i know:
- she did it at least 4 times, i found the listing. shes charging 90 a night
- she owes me 380 for utilities since {{n|month|march}}, i have the paynow screenshots, her number is {{ph|+65 8555 0137}}
- my mum ({{p|yating|lower|lin ya-ting}}, shes coming to stay {{d|2026-07-11|11 july}}) will lose her mind if theres a random man in the flat
- {{p|jiyeon|nickname|jiji}} is otherwise a good flatmate and i dont want to blow up the frienship

do i tell mr {{p|kaho|lower-last}}? do i give her a deadline? pls write me a msg to send her thats firm but not psycho. my name is {{p|+Hui Min|lower-first}} if u need to sign it`),
		ast(`Here is a draft that is firm without being hostile:

"{{p|jiyeon|nickname}}, I need to talk about the weekend guests. A stranger arriving with our door code is not okay with me, and subletting breaks clause 9 of the lease we both signed. Please stop the listings today and change nothing on the door until we have agreed a new code together. I also still need the 380 for utilities. Can we sort both by Sunday? I would rather fix this between us than involve Mr {{p|kaho|title-last|Cheung}}. {{p|+Hui Min|first}}"

I left out the detail about the two different names. You may want to ask about that in person.`),
		usr(`thats good but add a line that i already changed the code (i did, not telling u the new one lol) and that my mum arrives on the {{d|2026-07-11|11th of july}} so it has to be sorted before then. and dont call him mr {{p|kaho|lower-last}} in the msg, she calls him {{p|kaho|honorific|uncle ka-ho}}`)),
}

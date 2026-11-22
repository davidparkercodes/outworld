package main

var itemsChat = []def{
	it("chat-slack", "handle,lowercase-name,possessive,product-name,common-word-name,secret-in-chat",
		sys(`You summarize incident channels. List who did what, and every follow-up with its owner.`),
		usr(`#payments-oncall export, {{d|2026-10-06}}

[09:12] {{p|priya|handle|priya.r}}: morning. {{p|lukasz|lower|lukasz}} are you still seeing the 502s from the settlement job?
[09:13] {{p|lukasz|handle|lwisniewski}}: yep, started right after {{p|rob|possessive|Rob}}'s deploy. rolling back now
[09:15] {{p|priya|handle|priya.r}}: ok. can someone page {{p|tanaka|last|Tanaka}}-san, the {{o|Kafka|product}} consumer lag is past 40k
[09:16] {{p|rob|lower|rob}}: on it. fwiw the deploy was just the {{o|Jackson|product}} bump, nothing in the consumer
[09:21] {{p|tanaka|handle|haruto.t}}: here. lag is from partition 7. {{p|lukasz|ascii-fold|Lukasz}} can you restart settle-worker-3? creds are in the vault, or use the break-glass one: {{s|password|Tr4wler!Osprey-22}}
[09:22] {{p|lukasz|handle|lwisniewski}}: please don't paste that here {{p|tanaka|first|Haruto}} :( rotating it after this
[09:40] {{p|priya|handle|priya.r}}: resolved. {{p|priya|nickname|Priya}} + {{p|lukasz|first|Łukasz}} to write the postmortem by Thursday. let's not let the retry storm rob us of another weekend`)),

	it("chat-slack", "nickname,title-last,common-word-name,person-like-org",
		usr(`Can you turn this into a two line handover note?

{{p|chidinma|full|Chidinma Okwuosa-Bello}}  10:02 AM
@{{p|seun|nickname|Seun}} can you take the vendor call with {{o|Okafor & Daughters Logistics|person-like-org}} at 2? I'm stuck in the budget review

{{p|seun|full|Oluwaseun Adebayo}}  10:04 AM
sure. is it still Mr. {{p|skip|title-last|Hensley}} on their side or did he retire

{{p|chidinma|full|Chidinma Okwuosa-Bello}}  10:05 AM
{{p|skip|first|Skip}} retired in June, it's {{p|dawn|first|Dawn}} now. {{e|dawn.mwangi@okafor-daughters.example}}, desk line {{ph|404-555-0117}}

{{p|seun|full|Oluwaseun Adebayo}}  10:06 AM
got it. I'll skip the pricing slides then, {{p|dawn|first|Dawn}} has seen them. are we still doing the dawn slot for the warehouse walk-through or did that move

{{p|chidinma|full|Chidinma Okwuosa-Bello}}  10:09 AM
moved to 9. thanks {{p|seun|nickname|Seun}}. oh and {{p|chidinma|nickname|Chidi}} is fine, nobody types my whole name :)`)),

	it("chat-teams", "last-first,title-last,diacritics,particle-name",
		sys(`You are a workplace assistant. Answer from the pasted chat only.`),
		usr(`[{{d|2026-11-18|11/18/2026}} 3:41 PM] {{p|astrid|last-first|Lindqvist, Astrid}}: Has anyone heard from {{p|jurgen|first|Jürgen}} about the {{l|Hamburg}} site visit?
[{{d|2026-11-18|11/18/2026}} 3:43 PM] {{p|annika|last-first|Schönfeld, Annika}}: He's out until Monday. {{p|jurgen|possessive|Weißmüller}}'s OOO says to contact {{p|annelise|first|Annelise}}.
[{{d|2026-11-18|11/18/2026}} 3:44 PM] {{p|astrid|last-first|Lindqvist, Astrid}}: {{p|annelise|last|von Trautmann}}? She moved to procurement in August.
[{{d|2026-11-18|11/18/2026}} 3:47 PM] {{p|annika|last-first|Schönfeld, Annika}}: Then try his mobile, {{ph|+49 30 23125 417}}. Or mail {{e|juergen.weissmueller@nordhafen-logistik.example}}
[{{d|2026-11-18|11/18/2026}} 3:52 PM] {{p|astrid|last-first|Lindqvist, Astrid}}: Thanks {{p|annika|first|Annika}}. I'll loop in {{p|jurgen|initial-last|J. Weißmüller}} on the thread once he's back.

Who should I contact about the site visit and how?`)),

	it("chat-whatsapp", "first-name,lowercase-name,caps,uk-phone",
		usr(`need a nice message for the card based on this, something warm from all of us

{{d|2026-11-14|14/11/2026}}, 18:03 - Mum: Has anyone got {{p|eilidh|possessive|Eilidh}}'s new number? The old one just rings out
{{d|2026-11-14|14/11/2026}}, 18:05 - {{p|hamish|first|Hamish}}: {{ph|07700 900143}}. she changed it after the move to {{l|Inverness}}
{{d|2026-11-14|14/11/2026}}, 18:05 - Mum: Thanks love. And the address for the card?
{{d|2026-11-14|14/11/2026}}, 18:09 - {{p|hamish|first|Hamish}}: {{a|Flat 2, 18 Crannog Street, Inverness IV3 5QT}}
{{d|2026-11-14|14/11/2026}}, 18:10 - Mum: Lovely. Your auntie {{p|moira|first|Moira}} says hello by the way
{{d|2026-11-14|14/11/2026}}, 18:14 - {{p|eilidh|full|Eilidh MacLeod}}: I'M RIGHT HERE you know. it's {{e|eilidh.macleod@example.org}} if you want to send the photos mum
{{d|2026-11-14|14/11/2026}}, 18:15 - {{p|hamish|first|Hamish}}: sorry {{p|eilidh|lower|eilidh}} x`)),

	it("chat-imessage", "common-word-name,lowercase-name,address",
		usr(`{{d|2026-11-07|Sat, Nov 7}} at 9:41 AM
{{p|sandy|first|Sandy}}: are you guys still coming to the lake? beach is super sandy after the storm fyi
Me: yes!! leaving at 11. is {{p|rich|lower|rich}} bringing the kayak
{{p|sandy|first|Sandy}}: {{p|rich|first|Rich}} says yes but he needs the roof rack back from {{p|olive|first|Olive}}
Me: lol classic. tell him the olive oil is on me if he also brings the grill
{{p|sandy|first|Sandy}}: ok. gate code is {{s|gate-code|4471}}# and the cabin is {{a|212 Loon Hollow Rd, Ely, MN 55731}}
Me: perfect. whats {{p|lane|lower|lanes}} number, he's riding with us
{{p|sandy|first|Sandy}}: {{p|lane|first|Lane}} is {{ph|218-555-0164}}. stay in the right lane after the bridge, the left one is closed

^ can you pull out everything I need to bring and the address into a checklist`)),

	it("chat-irc", "family-name-first,handle,lowercase-name,honorific",
		usr(`<{{p|kenji|handle|kenji_m}}> anyone know who owns the billing-export cron now that {{p|zhang|lower|zhang}} left?
<{{p|bogdan|lower|bogdan}}> {{p|zhang|lower|zhang weiqi}}? he's still here, just moved teams. ping @{{p|zhang|handle|weiqi.zhang}}
<{{p|kenji|handle|kenji_m}}> oh. I thought {{p|zhang|first|Weiqi}} went to the {{l|Shenzhen}} office
<{{p|bogdan|lower|bogdan}}> that was {{p|+Li Xiuying|last|Li}}. different person, same desk
<{{p|kenji|handle|kenji_m}}> lol ok. the cron runs as user {{p|bogdan|handle|bionescu}} btw, is that you
<{{p|bogdan|lower|bogdan}}> ...yes. don't tell {{p|kenji|last|Morishita}}-san's boss
<{{p|kenji|handle|kenji_m}}> I am {{p|kenji|last|Morishita}}-san

Summarize who owns the cron and what needs to change.`)),

	it("voicemail", "lowercase-name,disfluency,spoken-phone",
		sys(`You clean up automatic voicemail transcriptions into a short note with caller, reason and callback details.`),
		usr(`voicemail transcript (auto), received {{d|2026-11-19}} 4:52 pm, from {{ph|(312) 555-0188}}

hi um this message is for {{p|fergus|lower|fergus}} uh {{p|fergus|lower|fergus mcallister}} this is {{p|tom|lower|tom}} calling from uh {{o|lakeshore dental}} about your appointment on on {{d|2026-12-03|december third}} we had to move doctor {{p|ayesha|possessive|siddiqui}}'s schedule around so um if you could call us back at {{ph|three one two five five five oh one eight eight|spoken}} that'd be great again that's {{p|tom|lower|tom}} at {{o|lakeshore dental}} okay thanks bye`)),

	it("voicemail", "lowercase-name,disfluency,spoken-phone,common-word-name",
		usr(`my phone transcribed this from my wife, what exactly do I need to do and by when?

"hey it's me uh so i just got off the phone with the school and they said {{p|bao|ascii-fold|bao}} can't do the field trip unless the form's in by friday so can you sign it tonight um also missus {{p|siobhan|lower|o'riordan}} wants a call back her number is uh {{ph|five five five zero one four two|spoken}} no area code i guess it's the same as ours and and don't forget {{p|amber|lower|amber}} is picking him up on thursday not me she'll wait by the amber light at the crossing okay love you bye"`)),

	it("voicemail", "lowercase-name,disfluency",
		usr(`is this a scam? transcript of a voicemail I just got:

this is a message for uh miss {{p|zoe|lower|hartigan wells}} my name is {{p|khalid|lower|khalid}} i'm calling from the fraud team at um {{o|first meridian bank}} regarding the card ending in four four four four we've seen a a charge of two hundred and twelve dollars at a merchant in in {{l|lisbon}} on the fourteenth please call us back on the number on the back of your card do not uh do not share your pin with anyone thank you`)),

	it("school", "title-last,initial-last,uk-phone,child-name",
		usr(`Can you help me write back to this?

"Dear Mr. and Mrs. {{p|tanvir|title-last|Chowdhury}}, {{p|+Ayaan Chowdhury|first|Ayaan}} was absent on {{d|2026-11-09|Monday 9 November}} and {{d|2026-11-10|Tuesday 10 November}} without a note. Please contact the attendance office on {{ph|020 7946 0155}} or {{e|attendance@stjudes-primary.example}}. Regards, Ms {{p|hannah|initial-last|H. Price}}, Year 4"

{{p|+Ayaan Chowdhury|first|Ayaan}} was at the dentist both days (Dr {{p|ayesha|title-last|Siddiqui}}, same as last time). My husband {{p|tanvir|first|Tanvir}} already emailed {{p|hannah|full|Hannah Price}} about it last week so I'm a bit annoyed. Keep it polite though.`)),

	it("family-logistics", "first-name,diacritics,place-as-name,common-word-name",
		usr(`Carpool for robotics club, week of {{d|2026-11-30|Nov 30}}:
Mon: {{p|noa|first|Noa}} drives ({{p|noa|first|Noa}}, {{p|jasper|first|Jasper}}, {{p|zoe|first|Zoë}}, {{p|bao|first|Bảo}})
Tue: no practice
Wed: {{p|meiling|first|Mei-Ling}} drives. pickup from {{a|41 Alder Court}} at 3:15
Thu: {{p|skip|first|Skip}}?? (he said maybe)
Fri: me

{{p|jasper|possessive|Jasper}}'s mum is {{p|folake|first|Folake}}, {{ph|0491 570 156}} (they're on {{l|Sydney}} time until the 2nd, long story). {{p|zoe|first|Zoë}} has a nut allergy so no trail mix. Can you turn this into a tidy message for the parents' group and remind {{p|skip|first|Skip}} he still hasn't confirmed, without making it sound like I'm asking him to skip it?`)),

	it("family-logistics", "nickname,diacritics,ascii-fold,form",
		usr(`Fill out the after-school registration from this info and tell me what's missing:

Child: {{p|gudrun|full|Guðrún Sigurðardóttir}}, goes by {{p|gudrun|nickname|Gunna}}, DOB {{d|2018-03-04|04.03.2018}}
Parent/guardian 1: {{p|marit|full|Marit Solbakken}}, {{ph|612-555-0109}}, {{e|marit@solbakken-family.example}}
Parent/guardian 2: {{p|eino|full|Eino Väisänen}} (stepfather), {{ph|612.555.0171}}
Authorized pickup: grandmother, {{p|astrid|first|Astrid}} (no last name on file)
Address: {{a|77 Tamarack Lane, Unit 3, Duluth, MN 55812}}
Allergies: none.
Notes: {{p|gudrun|nickname|Gunna}} answers to {{p|gudrun|nickname|Gunna}}, not {{p|gudrun|ascii-fold|Gudrun}}, please tell the Friday staff.`)),

	it("non-english-es", "spanish,nickname,two-surnames,common-word-name",
		usr(`Resume este chat en una frase y dime quién hace qué:

[{{d|2026-11-14|14/11/2026}}, 07:52] Mamá: Buenos días. ¿Quién recoge a la abuela el sábado?
[{{d|2026-11-14|14/11/2026}}, 07:55] {{p|vale|nickname|Vale}}: Yo puedo, salgo del trabajo a las dos. ¿Sigue en casa de la tía {{p|mafer|nickname|Mafer}}?
[{{d|2026-11-14|14/11/2026}}, 07:56] Mamá: Sí, {{a|Calle de Alcalá 142, 3º B, 28009 Madrid}}. Llama antes a {{p|mafer|first|María Fernanda}} porque el telefonillo no funciona
[{{d|2026-11-14|14/11/2026}}, 07:58] {{p|diego|first|Diego}}: Yo llevo el coche de {{p|jose|nickname|Pepe}} si hace falta. {{p|jose|first|José Luis}} dice que no lo usa hasta el martes
[{{d|2026-11-14|14/11/2026}}, 08:01] Mamá: Vale, perfecto. Gracias hijo
[{{d|2026-11-14|14/11/2026}}, 08:02] {{p|diego|first|Diego}}: jajaja "vale, {{p|vale|nickname|Vale}}"`)),

	it("non-english-es", "spanish,title-last,two-surnames,gov-id,iban,person-like-org",
		usr(`Tradúceme esto al inglés y dime qué tengo que llevar a la firma.

Estimado Sr. {{p|diego|title-last|Hernández Vargas}}:

Le confirmamos la recepción de su solicitud de alquiler del piso en {{a|Carrer de Mallorca 211, 2n 1a, 08036 Barcelona}}. Para formalizar el contrato necesitamos:
- Copia del DNI (nos consta el número {{g|dni|00000000T}}, por favor confirme)
- Número de cuenta para la domiciliación: {{f|iban|ES91 2100 0418 4502 0005 1332}}
- Teléfono de contacto de su avalista, doña {{p|vale|full|Valentina Gutiérrez Paredes}}

La firma sería el {{d|2026-12-03|3 de diciembre de 2026}} en nuestras oficinas.

Un saludo,
{{o|Ramírez y Asociados|person-like-org}}, Administración de Fincas`)),

	it("non-english-de", "german,title-last,particle-name,diacritics",
		usr(`Kannst du mir helfen, darauf höflich zu antworten und um einen Termin am Nachmittag zu bitten?

Sehr geehrte Frau {{p|annelise|title-last|von Trautmann}},

vielen Dank für Ihre Bewerbung vom {{d|2026-10-28|28. Oktober 2026}}. Wir möchten Sie gerne zu einem Gespräch am Donnerstag, den {{d|2026-11-12|12.11.2026}} um 10:30 Uhr einladen. Ihr Ansprechpartner vor Ort ist Herr Dr. {{p|jurgen|title-last|Weißmüller}}; bei Rückfragen erreichen Sie Frau {{p|annika|title-last|Schönfeld}} aus der Personalabteilung unter {{ph|030 23125 268}} oder {{e|annika.schoenfeld@nordhafen-logistik.example}}.

Bitte bringen Sie Ihren Personalausweis mit.

Mit freundlichen Grüßen
i. A. {{p|annika|full|Annika Schönfeld}}
{{o|Nordhafen Logistik GmbH}}, {{a|Am Sandtorkai 48, 20457 Hamburg}}`)),

	it("non-english-fr", "french,mixed-language,title-last,place-as-name,diacritics",
		usr(`Tu peux me traduire ce message du syndic en anglais pour mon coloc {{p|drew|first|Drew}} ? Il ne parle pas un mot de français.

"Bonjour Mme {{p|elodie|title-last|Beaumont-Lefèvre}}, suite à votre appel du {{d|2026-11-09|9 novembre}}, le plombier (M. {{p|thibault|title-last|Garnier}}, {{ph|01 99 00 41 27}}) passera jeudi entre 8h et 12h au {{a|14 rue des Tanneurs, 69005 Lyon}}. Merci de laisser les clés à la gardienne. Cordialement, le syndic"

Also add a line in English that I'll be in {{l|Nancy}} that week (the city, pas une copine) so {{p|drew|first|Drew}} has to be home Thursday morning. Sign it {{p|elodie|first|Élodie}}.`)),

	it("non-english-es", "spanish,mixed-language,title-last,gov-id,diacritics",
		usr(`oye necesito que me ayudes con un mensaje para mi jefa, Mrs. {{p|sandy|title-last|Whitlock}}. le tengo que decir que no puedo ir el lunes porque tengo cita en el consulado (renovación de pasaporte, el mío es el {{g|passport|G20417755}}) y que {{p|joao|first|João}} me cubre el turno. en inglés pero que suene natural, no muy formal. ella siempre firma como {{p|sandy|first|Sandy}} así que supongo que le puedo decir {{p|sandy|first|Sandy}}`)),

	it("non-english-de", "german,first-name,particle-name,password-in-prose",
		usr(`Fass mir bitte zusammen, was ich morgen erledigen muss:

[{{d|2026-11-19|19.11.26}}, 17:02:11] {{p|jurgen|first|Jürgen}}: Kannst du morgen {{p|zoe|first|Zoë}} vom Hort abholen? Ich hab um 16 Uhr den Termin beim Finanzamt
[{{d|2026-11-19|19.11.26}}, 17:03:40] {{p|annika|first|Annika}}: Geht klar. Braucht sie den Turnbeutel?
[{{d|2026-11-19|19.11.26}}, 17:04:02] {{p|jurgen|first|Jürgen}}: Ja, hängt im Flur. Und frag bitte Frau {{p|+Marieke de Groot|title-last|de Groot}}, ob die Rechnung für November schon raus ist
[{{d|2026-11-19|19.11.26}}, 17:06:15] {{p|annika|first|Annika}}: Mach ich. Übrigens hat Opa {{p|willem|first|Willem}} angerufen, er kommt am 2. Advent mit dem Zug aus {{l|Utrecht}}
[{{d|2026-11-19|19.11.26}}, 17:06:58] {{p|jurgen|first|Jürgen}}: Super. WLAN-Passwort für ihn ist immer noch {{s|password|Zimtstern#2019}}, falls er wieder fragt`)),
}

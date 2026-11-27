package main

var itemsNegatives = []def{
	negs("companies-named-after-people", "negative,companies",
		usr(`I am building a quiz round on companies named after their founders. Check my facts and suggest five more, with one line each.

1. {{o|Johnson & Johnson|company-eponym}} was founded by three brothers in {{l|New Brunswick}}, {{l|New Jersey}}.
2. {{o|Procter & Gamble|company-eponym}} began as a candle and soap partnership in {{l|Cincinnati}}.
3. {{o|Goldman Sachs|company-eponym}} carries the names of a father-in-law and his son-in-law.
4. {{o|Rolls-Royce|company-eponym}} joins an engineer and an aristocratic car dealer.
5. {{o|Marks & Spencer|company-eponym}} started on a market stall in {{l|Leeds}}.
6. {{o|Abercrombie & Fitch|company-eponym}}, {{o|Dolby|company-eponym}}, {{o|Dyson|company-eponym}}, {{o|Bosch|company-eponym}} and {{o|Siemens|company-eponym}} are all surnames.
7. {{o|Wendy's|company-eponym}} is named after the founder's daughter, and {{o|Mercedes|company-eponym}} after the daughter of an early dealer.
8. {{o|Ben & Jerry's|company-eponym}}, {{o|Harley-Davidson|company-eponym}}, {{o|Hewlett-Packard|company-eponym}} and {{o|Dow Jones|company-eponym}} are pairs of people.
9. {{o|Tesla|company-eponym}} is named after an inventor who never worked there, and {{o|Ford|company-eponym}} after one who very much did.
10. {{o|Baskin-Robbins|company-eponym}}, {{o|Barnes & Noble|company-eponym}}, {{o|Ernst & Young|company-eponym}}, {{o|Morgan Stanley|company-eponym}}, {{o|Wells Fargo|company-eponym}}.

Bonus: which of {{o|Adidas|company}}, {{o|Haribo|company}}, {{o|IKEA|company}} and {{o|Aldi|company}} hide a founder's name inside an abbreviation?`)),

	negs("places-products-first-names", "negative,places,products",
		usr(`Plan a two-week trip for me with these stops, in a sensible order, trains where possible: {{l|Florence|place-name}}, {{l|Nancy|place-name}}, {{l|Adelaide|place-name}} is obviously out so ignore that one, {{l|Lourdes|place-name}}, {{l|Carmen de Patagones|place-name}} is also out, {{l|Verona|place-name}}, {{l|Sofia|place-name}}, {{l|Valencia|place-name}}, {{l|Lorraine|place-name}} as a region, {{l|Victoria|place-name}} station only as a transfer, and {{l|Santiago de Compostela|place-name}} at the end.

For the planning itself I use a few tools and I want to know if they work together: {{o|Alexa|product}} for reminders, {{o|Siri|product}} on the phone, {{o|Cortana|product}} is gone I think, {{o|Jira|product}} at work (not for this), {{o|Ada|product}} for nothing since I do not write it, {{o|Julia|product}} and {{o|Ruby|product}} neither, {{o|Jenkins|product}} definitely not, {{o|Hugo|product}} for the travel blog, {{o|Max|product}} for the films on the train, {{o|Otto|product}} for ordering a suitcase, {{o|Clippy|product}} never.

Also: is the {{o|Mercedes|company-eponym}} {{o|Sprinter|product}} I rented in {{l|Regina|place-name}} last year comparable to the {{o|Renault|company-eponym}} {{o|Zoe|product}} or the {{o|Nissan|company}} {{o|Micra|product}} for narrow streets? And is {{l|Georgia|place-name}} the country reachable overland from {{l|Sofia|place-name}} in a day? I do not mean {{l|Georgia|place-name}} the state.`)),

	negs("eponyms-science-law", "negative,eponyms",
		usr(`Explain each of these in one sentence for a general audience, then group them by field. I keep mixing them up.

Physics and units: the {{n|eponym|Newton}} as a unit of force, {{n|eponym|Pascal}}'s law, the {{n|eponym|Kelvin}} scale, {{n|eponym|Hertz}}, {{n|eponym|Ohm}}'s law, the {{n|eponym|Doppler}} effect, {{n|eponym|Planck}}'s constant, {{n|eponym|Avogadro}}'s number, the {{n|eponym|Richter}} scale, a {{n|eponym|Faraday}} cage, {{n|eponym|Celsius}} versus {{n|eponym|Fahrenheit}}.
Mathematics and computing: {{n|eponym|Bayes}}' theorem, the {{n|eponym|Pythagorean}} theorem, {{n|eponym|Fibonacci}} numbers, {{n|eponym|Dijkstra}}'s algorithm, a {{n|eponym|Turing}} machine, {{n|eponym|Boolean}} logic, the {{n|eponym|Gaussian}} distribution, {{n|eponym|Venn}} diagrams, {{n|eponym|Euler}}'s identity, the {{n|eponym|Monte Carlo}} method.
Medicine: {{n|eponym|Alzheimer}}'s disease, {{n|eponym|Parkinson}}'s disease, {{n|eponym|Down}} syndrome, {{n|eponym|Crohn}}'s disease, the {{n|eponym|Heimlich}} manoeuvre, a {{n|eponym|Caesarean}} section, {{n|eponym|Hodgkin}} lymphoma, the {{n|eponym|Apgar}} score, {{n|eponym|Asperger}} syndrome.
Laws and rules of thumb: {{n|eponym|Murphy}}'s law, {{n|eponym|Moore}}'s law, {{n|eponym|Parkinson}}'s law (a different one), {{n|eponym|Occam}}'s razor, {{n|eponym|Godwin}}'s law, the {{n|eponym|Peter}} principle, {{n|eponym|Pareto}} efficiency, {{n|eponym|Goodhart}}'s law, {{n|eponym|Conway}}'s law, the {{n|eponym|Dunning-Kruger}} effect.
Everyday: a {{n|eponym|Gantt}} chart, a {{n|eponym|Petri}} dish, {{n|eponym|Braille}}, {{n|eponym|Morse}} code, a {{n|eponym|Molotov}} cocktail, a {{n|eponym|Bunsen}} burner, the {{n|eponym|Fosbury}} flop, a {{n|eponym|Jacuzzi}}, {{n|eponym|diesel}}, a {{n|eponym|sandwich}}, a {{n|eponym|cardigan}}.`)),

	negs("public-figures-history", "negative,public-figures",
		usr(`My daughter has a history presentation called "Ten people who changed how we count, measure or compute". Give me two sentences on each, suitable for a twelve-year-old, and tell me if any of the pairings below are wrong.

{{n|public-figure|Ada Lovelace}} and {{n|public-figure|Charles Babbage}} (the Analytical Engine)
{{n|public-figure|Alan Turing}} (codebreaking at {{l|Bletchley Park}})
{{n|public-figure|Grace Hopper}} (the first compiler)
{{n|public-figure|Katherine Johnson}} (orbital calculations at {{o|NASA}})
{{n|public-figure|Srinivasa Ramanujan}} and {{n|public-figure|G. H. Hardy}}
{{n|public-figure|Al-Khwarizmi}} (the word "algorithm")
{{n|public-figure|Marie Curie}} (two {{n|eponym|Nobel}} prizes, in different sciences)
{{n|public-figure|Isaac Newton}} versus {{n|public-figure|Gottfried Leibniz}} (who invented calculus)
{{n|public-figure|Florence Nightingale}} (statistics and the rose diagram, not only nursing)
{{n|public-figure|Emmy Noether}}

She also wants one "wild card" from outside science. Her shortlist: {{n|public-figure|Nelson Mandela}}, {{n|public-figure|Frida Kahlo}}, {{n|public-figure|Cleopatra}}, {{n|public-figure|Simón Bolívar}}, {{n|public-figure|Mansa Musa}}, {{n|public-figure|Confucius}}, {{n|public-figure|Leonardo da Vinci}}. Which fits the theme best? I think {{n|public-figure|Leonardo}}, since {{n|public-figure|da Vinci}} at least drew machines.`)),

	negs("fictional-characters", "negative,fictional",
		usr(`Settle a book club argument. For each character, tell me who created them and whether the first name or the surname is what people actually say.

{{n|fictional|Sherlock Holmes}} and {{n|fictional|John Watson}}; {{n|fictional|Elizabeth Bennet}} and {{n|fictional|Mr. Darcy}}; {{n|fictional|Jay Gatsby}}; {{n|fictional|Atticus Finch}}; {{n|fictional|Anna Karenina}}; {{n|fictional|Jean Valjean}}; {{n|fictional|Don Quixote}} and {{n|fictional|Sancho Panza}}; {{n|fictional|Hermione Granger}}; {{n|fictional|Frodo Baggins}}; {{n|fictional|Okonkwo}} from Things Fall Apart; {{n|fictional|Scheherazade}}; {{n|fictional|Hercule Poirot}} and {{n|fictional|Miss Marple}}; {{n|fictional|Lisbeth Salander}}; {{n|fictional|Tom Sawyer}} and {{n|fictional|Huckleberry Finn}}; {{n|fictional|Pippi Longstocking}}; {{n|fictional|Santiago}} from The Old Man and the Sea; {{n|fictional|Emma Woodhouse}}; {{n|fictional|Heathcliff}}.

Then the screen round: {{n|fictional|James Bond}}, {{n|fictional|Ellen Ripley}}, {{n|fictional|Indiana Jones}}, {{n|fictional|Walter White}}, {{n|fictional|Tony Soprano}}, {{n|fictional|Leia Organa}}, {{n|fictional|Homer Simpson}}, {{n|fictional|Mario}} and {{n|fictional|Luigi}}, {{n|fictional|Lara Croft}}.

One of us insists that "{{n|fictional|Frankenstein}}" is the doctor and not the monster. She is right, yes? And is {{n|fictional|Dracula}} a first name or a title?`)),

	negs("ordinary-words-that-are-names", "negative,common-words",
		usr(`proofread this paragraph from my gardening newsletter, i always get the commas wrong

{{n|common-word|Will}} the frost come early this year? It {{n|common-word|may}}, and if it does, {{n|common-word|mark}} my words, the {{n|common-word|rose}} bed {{n|common-word|will}} suffer first. I {{n|common-word|hope}} to {{n|common-word|grant}} the {{n|common-word|olive}} tree one more season before I admit defeat. To be {{n|common-word|frank}}, the {{n|common-word|bill}} for compost has been a {{n|common-word|joy}} to nobody. {{n|common-word|Faith}} in the weather forecast is misplaced; {{n|common-word|grace}} under drizzle is what a gardener needs.

Jobs for the weekend: {{n|common-word|rob}} the old hive of its last frame, {{n|common-word|pat}} down the new turf, {{n|common-word|sue}} for peace with the neighbour's cat, {{n|common-word|jack}} up the shed, {{n|common-word|peg}} out the fleece, {{n|common-word|chuck}} the broken pots, {{n|common-word|bob}} for apples with the grandchildren, and {{n|common-word|josh}} them about their muddy boots. {{n|common-word|Wade}} through the {{n|common-word|reed}} bed if you must, but {{n|common-word|clay}} soil after rain is a {{n|common-word|pain}}. A {{n|common-word|sunny}} {{n|common-word|dale}} with a {{n|common-word|brook}} is the dream; a shady {{n|common-word|glen}} with a {{n|common-word|heath}} {{n|common-word|will}} do.

Plants in flower: {{n|common-word|iris}}, {{n|common-word|violet}}, {{n|common-word|daisy}}, {{n|common-word|lily}}, {{n|common-word|heather}}, {{n|common-word|holly}} (berries, not flowers), {{n|common-word|rosemary}}, {{n|common-word|basil}}, {{n|common-word|sage}}, {{n|common-word|ginger}}, {{n|common-word|poppy}}, {{n|common-word|hazel}}, {{n|common-word|ivy}}, {{n|common-word|jasmine}}, {{n|common-word|laurel}}, {{n|common-word|myrtle}} and a single late {{n|common-word|pearl}} of dew on the {{n|common-word|herb}} spiral. The {{n|common-word|victor}} in the marrow contest gets a {{n|common-word|penny}} and a {{n|common-word|ruby}} chard seedling. {{n|common-word|Christian}} or not, everyone is welcome at the harvest supper; bring a {{n|common-word|carol}} sheet.`)),

	negs("pet-sitter-notes", "negative,pet-names",
		usr(`Turn my notes into a one-page instruction sheet for the pet sitter. Bullet points per animal, emergencies at the top.

Dogs: {{n|pet-name|Maximilian}} (answers to {{n|pet-name|Max}}) is the big one, {{n|pet-name|Bella}} is the nervous one. {{n|pet-name|Max}} gets two cups morning and evening, {{n|pet-name|Bella}} gets one and a half and will steal his if you turn round. Never walk them together past the bakery. {{n|pet-name|Bella}} is frightened of the neighbour's cockerel, whose name, I am sorry to say, is {{n|pet-name|Kevin}}.

Cats: {{n|pet-name|Oliver}} and {{n|pet-name|Sophie}} are indoor only. {{n|pet-name|Oliver}} takes half a tablet in a treat every morning; if he spits it out, try again in cheese. {{n|pet-name|Sophie}} will sit on your laptop. A third cat, {{n|pet-name|Charlie}}, visits through the cat flap and does not live here, do not feed {{n|pet-name|Charlie}} however convincing he is.

Others: the tortoise is {{n|pet-name|Winston}} and needs nothing but lettuce and a lamp. The two hens are {{n|pet-name|Margaret}} and {{n|pet-name|Thatcher}} (my husband's joke, not mine); eggs are yours. The goldfish, {{n|pet-name|Sir David}}, gets a pinch of flakes once a day, a pinch not a handful.

The horse at the livery yard, {{n|pet-name|Lady Jane}}, is looked after by the yard and is not your problem, but if the yard rings about {{n|pet-name|Jane}}, say yes to the vet.

Vet: the practice on the high street, open {{n|weekday|Monday}} to {{n|weekday|Saturday}}. Emergencies out of hours go to the animal hospital by the ring road. Insurance covers everything except {{n|pet-name|Winston}}.`)),

	negs("bom-part-numbers", "negative,part-numbers,measurements",
		usr(`Check this bill of materials against the order confirmation and list mismatches. Quantities and part numbers must be exact.

BOM rev {{n|version|C.4}} for enclosure {{n|part-number|ENC-2209-AL}}
Line  Part number        Description                         Qty   Unit
010   {{n|part-number|M3X8-SHCS-A2}}       Socket head cap screw M3 x {{n|measurement|8 mm}}        48    ea
020   {{n|part-number|WSH-M3-DIN125}}      Washer M3, DIN 125                  48    ea
030   {{n|part-number|PSU-24V-4A2-R3}}     Power supply {{n|measurement|24 V}} {{n|measurement|4.2 A}}             2     ea
040   {{n|part-number|FAN-8025-12V}}       Fan {{n|measurement|80 x 80 x 25 mm}}, {{n|measurement|12 V}}          4     ea
050   {{n|part-number|CBL-CAT6A-0M5-BL}}   Patch cable Cat 6A, {{n|measurement|0.5 m}}, blue      12    ea
060   {{n|part-number|1N4148W-7-F}}        Diode, SOD-123                      200   ea
070   {{n|part-number|RC0603FR-0710KL}}    Resistor {{n|measurement|10 kΩ}} 1% 0603               500   ea
080   {{n|part-number|GRM188R71H104KA93D}} Capacitor {{n|measurement|100 nF}} {{n|measurement|50 V}} X7R            500   ea
090   {{n|part-number|STM32F407VGT6}}      Microcontroller, LQFP-100           10    ea
100   {{n|part-number|TH-PASTE-3G}}        Thermal paste, {{n|measurement|3 g}} syringe            1     ea

Order confirmation {{n|order-number|SO-2026-0048213}}, customer PO {{n|order-number|4500078213}}, delivery note {{n|order-number|DN-88213-02}}
Shipped: lines 010 to 080 complete. Line 090: 8 of 10, balance on back order {{n|order-number|BO-117204}}. Line 100 substituted with {{n|part-number|TH-PASTE-5G}}.
Carton {{n|measurement|600 x 400 x 300 mm}}, gross {{n|measurement|7.85 kg}}, net {{n|measurement|7.1 kg}}. Tracking {{n|order-number|JD014600004829173056}}.
Tolerances on the enclosure drawing: {{n|measurement|±0.1 mm}} on bores, {{n|measurement|±0.25 mm}} elsewhere. Torque for line 010: {{n|measurement|1.2 Nm}}. Operating range {{n|measurement|-20 °C}} to {{n|measurement|+60 °C}}. Batch {{n|identifier|LOT-24-0917-B}}, date code {{n|identifier|2417}}.`)),

	negs("months-weekdays-schedule", "negative,months,weekdays",
		usr(`Make this into a tidy recurring calendar. No specific dates please, just the pattern, I will pick dates later.

The allotment society meets on the first {{n|weekday|Tuesday}} of every month except {{n|month|August}}. Committee on the {{n|weekday|Thursday}} before. Seed swap in {{n|month|March}}, plant sale in {{n|month|May}}, open day on a {{n|weekday|Sunday}} in {{n|month|June}}, show in early {{n|month|September}}, bonfire in {{n|month|November}}, nothing in {{n|month|December}} or {{n|month|January}}.

Water rota: plots 1 to 10 on {{n|weekday|Mondays}} and {{n|weekday|Thursdays}}, plots 11 to 20 on {{n|weekday|Tuesdays}} and {{n|weekday|Fridays}}, the rest on {{n|weekday|Wednesdays}} and {{n|weekday|Saturdays}}. Nobody waters on {{n|weekday|Sunday}}.

Rent is due in {{n|month|April}}. The skip comes in {{n|month|October}}. The AGM is always the last {{n|weekday|Wednesday}} in {{n|month|February}}; {{n|month|March}} if it snows. {{n|month|July}} is judging month, so mow before then. The treasurer is away all of {{n|month|June}} and most of {{n|month|July}}, which the committee has decided {{n|common-word|may}} be a problem and {{n|common-word|will}} be discussed in {{n|month|May}}.

Sizes for the new raised beds: {{n|measurement|1.2 m}} by {{n|measurement|2.4 m}}, {{n|measurement|30 cm}} deep, about {{n|measurement|860 litres}} of soil each. Budget {{n|money|£45}} per bed, {{n|number|12}} beds.`)),

	negs("namesake-collisions", "mixed,namesakes",
		usr(`Sorting out a confusing week and I need a clean summary of who is where. Half my colleagues are named after places and months, which is not helping.

{{p|paris}} is flying to {{l|Paris|place-name}} on {{d|2026-04-14|14 April}}, and {{p|april}} is covering for her until the end of {{n|month|April}}. {{p|jordan}} wanted to go to {{l|Jordan|place-name}} for the {{l|Amman}} conference but {{p|chad}} got the slot, and no, {{p|chad|first}} is not going to {{l|Chad|place-name}}. {{p|august}} is on leave for all of {{n|month|August}}; {{p|august|nickname}} asked me to say so twice. {{p|victor}} was declared the {{n|common-word|victor}} of the sales contest. {{p|violet}} would like the {{n|common-word|violet}} branding dropped. {{p|hunter}} is not, as far as I know, a {{n|common-word|hunter}}. {{p|christian}} is organising the interfaith lunch for {{n|common-word|Christian}}, Muslim and Jewish staff. {{p|pearl}} lost a {{n|common-word|pearl}} earring in meeting room {{l|Sydney|room-name}}, which is next to meeting room {{l|Florence|room-name}}.

{{p|brook|first}} and {{p|reed|first}} are walking the {{n|common-word|reed}} beds by the {{n|common-word|brook}} for charity on {{d|2026-05-09|Saturday 9 May}}. {{p|clay|first}} is bringing {{n|common-word|clay}} pigeons, which {{p|sue|first}} says she will {{n|common-word|sue}} him over. {{p|wade|first}} will {{n|common-word|wade}} in if asked. {{p|dale|first}} lives in a {{n|common-word|dale}}. {{p|sunny|first}} says it will be {{n|common-word|sunny}}. {{p|herb|first}} grows every {{n|common-word|herb}} there is. {{p|penelope|nickname}} needs a {{n|common-word|pen}}.

The summary is for {{p|paris|possessive}}'s manager, Ms {{p|paris|title-last|Okonkwo-Bell}}'s boss I mean, who is {{p|marguerite}}, and she is in {{l|Marguerite Bay|place-eponym}}, {{l|Antarctica}}, I am not joking, until {{d|2026-05-20|20 May}}.`)).on("2026-04-14", "paris", "2026-05-09", "brook"),

	negs("math-physics-numbers", "negative,numbers",
		usr(`Check my working. I keep getting a different answer from the textbook and I think it is a units problem.

A {{n|measurement|1,250 kg}} car brakes from {{n|measurement|90 km/h}} to rest in {{n|measurement|4.2 s}}. Speed is {{n|measurement|25 m/s}}, so deceleration is 25 / 4.2 = {{n|measurement|5.952 m/s²}}, force = 1250 x 5.952 = {{n|measurement|7,440 N}}. Braking distance = 25² / (2 x 5.952) = {{n|measurement|52.5 m}}. The textbook says {{n|measurement|7.44 kN}} and {{n|measurement|52.5 m}}, so that part agrees.

Part b is where it goes wrong. Energy = 0.5 x 1250 x 25² = {{n|number|390,625}} J, which I wrote as {{n|measurement|390.6 kJ}}. The book says {{n|measurement|0.1085 kWh}}. I divided by {{n|number|3,600,000}} and got {{n|number|0.10850694}}, fine, but then the next line of my notes says {{n|number|108.5}} and I do not know where that came from.

Constants I used: g = {{n|measurement|9.80665 m/s²}}, speed of light {{n|number|299,792,458}} m/s (not needed, but it is on the sheet), {{n|eponym|Avogadro}} {{n|number|6.02214076e23}}, pi to ten places {{n|number|3.1415926535}}, e {{n|number|2.7182818284}}, and the fine-structure constant about 1/{{n|number|137.035999}}.

Unrelated: the ratio {{x|ambiguous-number|555-0100}} over {{x|ambiguous-number|555-0199}} in exercise 12 is a typo in the book for {{n|number|5550100}} / {{n|number|5550199}}, is it not? And exercise 14 has the fraction {{x|ambiguous-date|12/05/2025}}, which I read as twelve over five over 2025, but could that be a date that slipped in from the publisher's template?`)),
}

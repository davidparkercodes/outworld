package main

var itemsDataA = []def{
	data("csv-customers", "csv",
		usr(`Write a Python snippet that normalises the phone column to E.164 and flags rows where the country does not match the phone prefix. Sample:

customer_id,first_name,last_name,email,phone,country,birth_date,street,city,postcode,vat_or_tax_id
88001,{{p|solene|first}},{{p|solene|last}},{{e|solene.kerbrat@armor-mail.example}},{{ph|02 99 00 41 27}},FR,{{d|1987-03-19|1987-03-19}},{{a|14 rue de Siam}},{{l|Brest}},29200,
88002,{{p|thibault|first}},"{{p|thibault|last}}",{{e|t.delarochefoucault@mbaye-conseil.example}},{{ph|+33 6 39 98 12 04}},FR,{{d|1979-11-02|1979-11-02}},"{{a|3, allée des Tilleuls}}",{{l|Nantes}},44000,
88003,{{p|urs|first}},{{p|urs|last}},{{e|urs.zurbriggen@walliser-post.example}},{{ph|+41 27 555 01 43}},CH,{{d|1965-08-01|1965-08-01}},{{a|Bahnhofstrasse 7}},{{l|Zermatt}},3920,{{g|ch-ahv|756.4821.9037.64}}
88004,{{p|ludovica|first}},{{p|ludovica|last}},{{e|ludo.sanfilippo@postacerta.example}},{{ph|347 555 0109}},IT,{{d|1995-07-21|1995-07-21}},{{a|Via Garibaldi 12}},{{l|Torino}},10122,{{g|it-cf|SNFLVC95L61L219F}}
88005,{{p|jarmila|first}},{{p|jarmila|last}},{{e|jarmila.prochazkova@seznam-posta.example}},{{ph|+420 555 010 177}},CZ,{{d|1978-11-05|1978-11-05}},{{a|Národní 28}},{{l|Praha}},110 00,{{g|cz-rc|786105/2419}}
88006,{{p|torbjorn|first}},{{p|torbjorn|last}},{{e|tobben@fjordnett.example}},{{ph|+47 55 55 50 62}},NO,{{d|1975-08-19|1975-08-19}},{{a|Strandgaten 196}},{{l|Bergen}},5004,{{g|no-fnr|19087534106}}
88007,{{p|mikkel|first}},{{p|mikkel|last}},{{e|mikkel.ostergaard@danpost.example}},{{ph|+45 55 50 12 34}},SE,{{d|1983-05-02|1983-05-02}},{{a|Nørrebrogade 44, 2. th.}},{{l|København N}},2200,{{g|dk-cpr|020583-1428}}
88008,{{p|irina|first}},{{p|irina|last}},{{e|irina.zhuravleva@pochta.example}},{{ph|8 (495) 555-01-26}},RU,{{d|1990-12-30|1990-12-30}},"{{a|ул. Тверская, д. 18, кв. 44}}",{{l|Москва}},125009,{{g|ru-inn|773458219040}}
88009,{{p|irina|ascii-fold|IRINA}},{{p|irina|caps-last|ZHURAVLEVA}},{{e|IRINA.ZHURAVLEVA@POCHTA.EXAMPLE}},{{ph|+7 495 555 01 26}},RU,{{d|1990-12-30|30.12.1990}},,,,`)),

	data("tsv-employees", "tsv",
		usr(`Pivot this by department with headcount and average tenure in years as of today. Paste from the HR system, tab separated.

emp_no	name	preferred	dept	manager	start_date	work_email	mobile	national_id
E-20001	{{p|marguerite|last-first|Delacroix-Benoît, Marguerite}}	{{p|marguerite|nickname}}	Operations	-	{{d|2016-06-13|13/06/2016}}	{{e|m.delacroix-benoit@corvanefreight.example}}	{{ph|07700 900102}}	{{g|gb-nino|QQ 12 34 56 C}}
E-20044	{{p|obinna|last-first}}	{{p|obinna|nickname}}	Customs	{{p|marguerite|initial-last|M. Delacroix-Benoît}}	{{d|2019-02-04|04/02/2019}}	{{e|obinna.c@corvanefreight.example}}	{{ph|07700 900144}}	{{g|gb-nino|QQ 48 29 17 A}}
E-20118	{{p|siobhan|last-first|Ní Bhroin, Siobhán}}	{{p|siobhan|nickname}}	Customs	{{p|obinna|initial-last|O. Chukwuemeka}}	{{d|2020-09-01|01/09/2020}}	{{e|siobhan.nib@corvanefreight.example}}	{{ph|+353 20 915 5012}}	{{g|ie-pps|8273641L}}
E-20417	{{p|tevita|last-first}}	{{p|tevita|nickname}}	Depot	{{p|obinna|initial-last|O. Chukwuemeka}}	{{d|2021-03-15|15/03/2021}}	{{e|tevita.fifita-latu@corvanefreight.example}}	{{ph|07700 900388}}	{{g|gb-nino|QQ 73 05 81 B}}
E-20502	{{p|ingrid|last-first}}		People	-	{{d|2018-01-08|08/01/2018}}	{{e|ingrid.aasgaard-thorsen@corvanefreight.example}}	{{ph|+47 55 55 50 19}}	{{g|no-fnr|19087534297}}
E-20633	{{p|rashida|last-first}}		People	{{p|ingrid|initial-last|I. Aasgaard-Thorsen}}	{{d|2022-11-21|21/11/2022}}	{{e|rashida.begum@corvanefreight.example}}	{{ph|07700 900633}}	{{g|gb-nino|QQ 20 63 30 D}}
E-20701	{{p|khin|full}}	{{p|khin|first}}	Sales (APAC)	{{p|jordan|initial-last|J. Mwangi-Hale}}	{{d|2023-05-02|02/05/2023}}	{{e|khin.thida.soe@corvanefreight.example}}	{{ph|+65 8555 0170}}	{{g|sg-nric|G5827364W}}
E-20702	{{p|wulan|full}}	{{p|wulan|nickname}}	Sales (APAC)	{{p|jordan|initial-last|J. Mwangi-Hale}}	{{d|2023-05-02|02/05/2023}}	{{e|sulistyowati@corvanefreight.example}}	{{ph|+62 812 5550 0147}}	{{g|id-nik|3578104507890002}}
E-20755	{{p|hemi|last-first}}		Sales (APAC)	{{p|jordan|initial-last|J. Mwangi-Hale}}	{{d|2024-02-19|19/02/2024}}	{{e|hemi.tuhoro@corvanefreight.example}}	{{ph|+64 21 555 0119}}	{{g|nz-ird|63-482-919}}`)),

	data("json-records", "json",
		usr(`Write a JSON Schema for these records. Mark which fields are personal data so we can tag them in the catalogue.

[
  {
    "patient_id": "{{n|uuid|9b2d7f4e-1c3a-4e5b-8f6d-0a9c8b7e6d5f}}",
    "name": {"family": "{{p|sunita|last}}", "given": ["{{p|sunita|first}}"], "prefix": "Mrs"},
    "birthDate": "{{d|1961-04-09|1961-04-09}}",
    "identifiers": [{"system": "aadhaar", "value": "{{g|in-aadhaar|5837 2946 1080}}"}, {"system": "pan", "value": "{{g|in-pan|BXKPR4829L}}"}],
    "telecom": [{"use": "mobile", "value": "{{ph|+91 98555 01432}}"}, {"use": "email", "value": "{{e|sunita.raghunathan@chennaimail.example}}"}],
    "address": {"line": ["{{a|Flat 4B, Kaveri Apartments, 27 Third Main Road, Adyar}}"], "city": "{{l|Chennai}}", "postalCode": "600020"},
    "contact": [{"relationship": "son", "name": "{{p|karthik}}", "phone": "{{ph|+91 98555 01789}}"}],
    "generalPractitioner": "Dr. {{p|rajesh|initial-first|S. Rajeshwari}}",
    "lastVisit": "{{d|2026-03-17|2026-03-17T10:30:00+05:30}}"
  },
  {
    "patient_id": "{{n|uuid|4a7c1e9b-6d2f-4b8a-9c3e-5f1d0b7a2e6c}}",
    "name": {"family": "{{p|gurpreet|last}}", "given": ["{{p|gurpreet|first}}", "{{p|gurpreet|name-part|Kaur}}"], "prefix": "Ms"},
    "birthDate": "{{d|1994-10-26|1994-10-26}}",
    "identifiers": [{"system": "aadhaar", "value": "{{g|in-aadhaar|741290583367}}"}],
    "telecom": [{"use": "mobile", "value": "{{ph|098555 01655}}"}],
    "address": {"line": ["{{a|H. No. 1184, Sector 22-B}}"], "city": "{{l|Chandigarh}}", "postalCode": "160022"},
    "contact": [],
    "generalPractitioner": null,
    "lastVisit": null
  }
]`)),

	data("yaml-records", "yaml",
		usr(`This YAML fails to parse on line 9 for my colleague but not for me. Why? And can you convert it to JSON once it is fixed?

# team directory, edited by hand, sorry
team: platform
updated: {{d|2026-05-30|2026-05-30}}
members:
  - name: {{p|takumi}}
    kana: {{p|takumi|native-script|ナカシマ タクミ}}
    handle: "@{{p|takumi|handle|tnakashima}}"
    email: {{e|takumi.nakashima@lumenwharf.example}}
    phone: {{ph|+81 6 5550 0143}}
    tz: Asia/Tokyo
    emergency_contact: {name: {{p|kazuko}}, relation: mother, phone: "{{ph|090-5550-0167}}"}
  - name: {{p|adaeze}}
    aka: [{{p|adaeze|nickname}}, {{p|adaeze|alias|Adaeze Okpara}}]
    handle: "@{{p|adaeze|handle|adaeze-ol}}"
    email: {{e|adaeze.okpara-lindqvist@lumenwharf.example}}
    phone: {{ph|+46 70 174 06 51}}
    tz: Europe/Stockholm
    notes: |
      On leave {{d|2026-06-15|15 June}} to {{d|2026-07-03|3 July}}. Cover: {{p|kwabena|nickname}}.
      Do not page; text first.
  - name: {{p|kwabena}}
    handle: "@{{p|kwabena|handle|kobby}}"
    email: {{e|kobby@lumenwharf.example}}
    phone: {{ph|+233 20 555 0158}}
    tz: Africa/Accra
    born: {{d|1993-08-04|1993-08-04}}   # for the cake rota, {{p|sue|possessive}}'s idea
    no: true   # yes he is from norway originally? no. this key is a bug, see the Norway problem`)),

	data("auth-log-lines", "log",
		usr(`Which of these lines indicate an actual compromise versus someone fat-fingering a password? I need to brief {{p|ibrahim|first}} in ten minutes.

{{d|2026-04-11|Apr 11}} 02:13:07 bastion sshd[20417]: Failed password for {{p|wade|handle|wkimathi}} from 203.0.113.61 port 51422 ssh2
{{d|2026-04-11|Apr 11}} 02:13:11 bastion sshd[20417]: Failed password for {{p|wade|handle|wkimathi}} from 203.0.113.61 port 51422 ssh2
{{d|2026-04-11|Apr 11}} 02:13:19 bastion sshd[20417]: Accepted password for {{p|wade|handle|wkimathi}} from 203.0.113.61 port 51422 ssh2
{{d|2026-04-11|Apr 11}} 02:13:20 bastion sshd[20419]: pam_unix(sshd:session): session opened for user {{p|wade|handle|wkimathi}}(uid=1014) by (uid=0)
{{d|2026-04-11|Apr 11}} 02:14:02 bastion sudo: {{p|wade|handle|wkimathi}} : TTY=pts/3 ; PWD=/home/{{p|wade|path|wkimathi}} ; USER=root ; COMMAND=/usr/bin/cat /etc/shadow
{{d|2026-04-11|Apr 11}} 02:14:40 bastion sshd[20455]: Invalid user {{p|clay|lower-first}} from 198.51.100.200 port 40022
{{d|2026-04-11|Apr 11}} 02:14:41 bastion sshd[20455]: Invalid user admin from 198.51.100.200 port 40022
{{d|2026-04-11|Apr 11}} 02:14:43 bastion sshd[20455]: Invalid user {{p|sunny|lower-first}} from 198.51.100.200 port 40022
{{d|2026-04-11|Apr 11}} 02:15:10 bastion app[881]: login ok user={{e|wade.kimathi@lumenwharf.example}} mfa=sms:{{ph|+254 722 555 014}} otp={{s|otp|418 207}} ua="Mozilla/5.0"
{{d|2026-04-11|Apr 11}} 02:15:55 bastion app[881]: password changed user={{e|wade.kimathi@lumenwharf.example}} by=self new_hash={{s|password-hash|$2b$12$Xq9vL2mT7pRz4KcW8bYn3uHsJd6AfG1uEo5QiNvXtKr0ZaPw3yBcS}}
{{d|2026-04-11|Apr 11}} 02:16:30 bastion app[881]: api key created user={{e|wade.kimathi@lumenwharf.example}} key={{s|api-key|bk_live_7Hq2LmX9vRt4ZpW8cYb3NfK6sJd1AeG5}} scope=admin
{{d|2026-04-11|Apr 11}} 08:01:12 bastion sshd[23010]: Accepted publickey for {{p|wade|handle|wkimathi}} from 192.0.2.44 port 60211 ssh2: ED25519 SHA256:{{n|public-key|n7Q2xLmV9pRt4ZkW8cYb3NfH6sJd1AeG5uTo0XaBcDe}}
{{d|2026-04-11|Apr 11}} 08:01:40 bastion app[881]: support chat user={{e|wade.kimathi@lumenwharf.example}} msg="this is {{p|wade}}, I did not log in last night, I was asleep in {{l|Nakuru}}"`)),

	data("markdown-table", "markdown,table",
		usr(`Sort this by renewal date and add a column with days until renewal from {{d|2026-06-01|June 1, 2026}}. Keep the markdown.

| Account | Champion | Role | Email | Phone | Renewal | ARR | Notes |
|---|---|---|---|---|---|---|---|
| {{o|Duna Spedíció Kft.}} | **{{p|zoltan|markdown|Zoltán}}** {{p|zoltan|last}} | MD | [{{e|zoli@dunaspedicio.example}}](mailto:{{e|zoli@dunaspedicio.example}}) | {{ph|+36 1 555 0128}} | {{d|2026-09-30|Sep 30, 2026}} | €5,760 | Prefers *{{p|zoltan|nickname}}*. Damaged pallets in April. |
| {{o|Pohjolan Rahti Oy}} | {{p|aino}} | Finance lead | {{e|aino.makiranta@pohjolanrahti.example}} | {{ph|+358 40 555 0149}} | {{d|2026-07-15|Jul 15, 2026}} | €15,000 | Double charge in May, refunded. |
| {{o|Marmara Lojistik A.Ş.}} | {{p|oguz}} | Owner | {{e|oguzhan@marmaralojistik.example}} | {{ph|+90 212 555 01 58}} | {{d|2027-01-10|Jan 10, 2027}} | €9,900 | Call him _{{p|oguz|honorific|Oğuz Bey}}_. Invoices to {{p|gulsen|first}} Hanım. |
| {{o|Lagos Freightline}} | {{p|oluwaseun}} | COO | {{e|seun@lagosfreightline.example}} | {{ph|+234 1 555 0167}} | {{d|2026-06-20|Jun 20, 2026}} | €11,880 | Married to our {{p|ngozi|first}}. Conflict check done. |
| {{o|Transportes Illimani}} | {{p|fernando}} | Dueño | {{e|fernando.iturralde@illimani.example}} | {{ph|+591 2 555 0138}} | {{d|2026-08-01|Aug 1, 2026}} | €3,720 | Angry about #2291. Dispatcher: {{p|xiadani|nickname}}. |
| {{o|Addis Air Cargo}} | {{p|tigist}} | Ops | {{e|tigist.h@addisair.example}} | {{ph|+251 11 555 0152}} | {{d|2026-12-05|Dec 5, 2026}} | €3,720 | Ethiopian naming: address her as W/ro {{p|tigist|first}}, not "Ms {{p|tigist|name-part|Woldemariam}}". |`)),

	data("form-label-value", "form,label-value",
		usr(`Extract this scanned form into JSON with snake_case keys. Leave blanks as null.

PATIENT REGISTRATION / REGISTRO DE PACIENTE

Last name / Apellido(s): {{p|ximena|last}}
First name / Nombre(s): {{p|ximena|first}}
Preferred name: {{p|ximena|nickname}}
Date of birth (MM/DD/YYYY): {{d|1992-08-14|08/14/1992}}
Sex: F
SSN or ITIN: {{g|us-ssn|987-65-4320}}
Street address: {{a|4418 E Burnside Ave, Apt 12}}
City: {{l|Las Cruces}}    State: NM    ZIP: 88011
Home phone: n/a
Cell phone: {{ph|575-555-0153}}
Email: {{e|xime.villasenor@correo-rapido.example}}
Preferred language: Español
Emergency contact: {{p|ernesto}}     Relationship: tío / uncle
Emergency phone: {{ph|(575) 555-0190}}
Primary insurance: {{o|Desert Sky Health Plan}}
Member ID: {{f|policy|DSH-884-417-2291}}        Group no.: {{n|identifier|GRP-55102}}
Policy holder (if not patient): {{p|ernesto|last-first|Zavaleta Bermúdez, Ernesto}}
Policy holder DOB: {{d|1968-01-30|01/30/1968}}
Referring physician: Dr. {{p|anjali|initial-last|A. Deshpande}}
Pharmacy: {{o|Walgreens|company-eponym}}, {{a|1250 El Paseo Rd}}
Signature: {{p|ximena|initial-last|X. Villaseñor A.}}        Date: {{d|2026-02-09|02/09/2026}}

OFFICE USE ONLY   MRN: {{x|medical-record-number|00417733}}   Verified ID: Mexican passport {{g|mx-passport|G48291730}} / CURP {{g|mx-curp|VIAX920814MJCLRM08}}   Clerk: {{p|+T. Begay|initial-last}}`)),
}

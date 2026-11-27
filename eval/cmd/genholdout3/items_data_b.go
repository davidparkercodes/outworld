package main

var itemsDataB = []def{
	data("ocr-scan", "ocr,messy",
		usr(`This came out of the scanner app. Can you reconstruct the letter as clean text? Guess the obvious OCR errors but mark anything you are unsure of with [?].

    S AGUARO  MUTUAL   lNSURANCE  C0MPANY
  C1aims Departrnent   {{a|P.0. Box 44l0|ocr}}  {{l|Phoenix}} AZ 8500l

  {{d|2026-10-21|0ctober 2l, 2O26|ocr}}

  {{p|itzel|ocr|ltzel Carrasco-Whitfie1d}}
  {{a|22l5 W Calle Paraiso}}
  {{l|Tucson}} AZ  85745

  Re:  Appea1 of c1aim  CLM-AZ-2O26-ll8843
       Po1icy  {{f|policy|HO-3-77204918-02}}
       lnsured:  {{p|desmond|ocr|Desrnond Whitfield}}

  Dear Ms. {{p|itzel|ocr|Carrasco-Whitfie1d}}:

  We acknow1edge receipt of your appea1 dated {{d|2026-10-14|October 14, 2026}} and the
  report of Mr. {{p|ernesto|ocr|Emesto Zava1eta}} ({{o|Zavaleta Roofing|company-eponym}}).   A second inspection
  has been schedu1ed for  {{d|2026-11-04|Wednesday, November 4, 2026}}  between 8 and l2.
  The inspector,  {{p|hunter|caps}} ,  wi11 ca11 you the day before
  at  {{ph|(52O) 555-Ol47|ocr}}  .   lf this number is incorrect p1ease
  ca11  {{ph|1-800-555-0199}}  or emai1   {{e|appeals@saguaromutual.example}}

  S incere1y ,
  {{p|sue|ocr|Susan Harr0wgate}}
  C1aims Superv isor

                 page l of l      ref  SH/hm      {{d|2026-10-21|10/21/2026}}`)),

	data("all-caps-notice", "caps,notice",
		usr(`REWRITE THIS IN NORMAL SENTENCE CASE AND MAKE IT LESS ALARMING. IT IS GOING ON THE DEPOT NOTICEBOARD.

NOTICE TO ALL DRIVERS AND WAREHOUSE STAFF, {{l|AVONMOUTH}} DEPOT
ISSUED {{d|2026-01-12|12 JANUARY 2026}} BY {{p|obinna|caps}}, CUSTOMS MANAGER

EFFECTIVE {{d|2026-01-19|MONDAY 19 JANUARY}} NO VEHICLE LEAVES THE YARD WITHOUT A SIGNED T1 FROM {{p|siobhan|caps-first|SIOBHÁN}} OR {{p|maianh|caps|TRẦN THỊ MAI ANH}}. NO EXCEPTIONS. THIS FOLLOWS THE INCIDENT ON {{d|2026-01-08|8 JANUARY}} WHEN A LOAD FOR {{o|HALDEN & PRYCE|company-eponym}} LEFT WITHOUT PAPERWORK AND WAS STOPPED AT {{l|DOVER}}.

OUT OF HOURS: CALL THE DUTY SUPERVISOR, {{p|tevita|caps}}, ON {{ph|07700 900388}}. IF {{p|tevita|caps-first|TEVITA}} DOES NOT ANSWER CALL {{p|jurgen|caps|JÜRGEN WEISSGERBER}} ON {{ph|+49 171 39200 18}}. DO NOT CALL {{p|marguerite|caps-first|MARGUERITE}} AT HOME AGAIN.

DRIVERS {{p|baraka|caps-last|OLE SANKALE}}, {{p|hemi|caps-last|TŪHORO}} AND {{p|wade|caps-last|KIMATHI}} (AGENCY) HAVE ALREADY BEEN BRIEFED. EVERYONE ELSE MUST SIGN THE SHEET IN THE OFFICE BY FRIDAY.

LOST PROPERTY: ONE HI-VIS MARKED "{{p|reed|caps-first|REED}}", ONE THERMOS, ONE COPY OF "{{n|fictional|JANE EYRE}}". SEE {{p|rashida|caps-first|RASHIDA}}.

QUESTIONS TO {{e|CUSTOMS@CORVANEFREIGHT.EXAMPLE}} OR EXT. 4417.`)),

	data("voice-dictation", "nocaps,dictation,typos",
		usr(`i dictated these notes in the car and the phone made a mess of it. fix punctuation and capitals and give it back as a to do list. dont change what i meant

note to self its {{d|2026-03-10|tuesday the tenth of march}} ok first thing call {{p|sunita|lower-first}} aunty about the flat in {{l|adyar}} she said the tenant mr {{p|arjun|lower-last}} wants to renew but at the same rent which no then call {{p|karthik|lower-first}} hes got her power of attorney papers his number is {{ph|nine eight five five five zero one seven eight nine|spoken}} i think or its in whatsapp second thing the bank the manager is {{p|rajesh|lower|rajeshwari subramaniam}} at the {{l|mylapore}} branch she needs the pan card copy mine is {{g|in-pan-spoken|b x k p r four eight two nine l}} no wait thats ammas mine is in the blue folder third email {{p|arjun|lower|arjun thekkekara}} the new rent figure thirty eight thousand from {{d|2026-06-01|june first}} his email is {{e|arjun.t@chennaimail.example}} {{e|arjun dot t at chennai mail dot example}} fourth book tickets for {{p|sunita|honorific|sunita aunty}} and {{p|karthik|nickname|kathu}} to come for diwali {{p|karthik|lower-first}} wants the window as always fifth remind {{p|zainab|lower-first}} at work that i am out on the {{d|2026-03-20|twentieth of march}} she keeps booking me into the friday review ok thats it oh and pick up the cake the bakery closes at seven`)),

	data("vcard", "vcard,contacts",
		usr(`Merge these two vCards into one. Where they conflict, prefer the newer REV and list the conflicts for me.

BEGIN:VCARD
VERSION:3.0
N:{{p|khalid|last}};{{p|khalid|first}};{{p|khalid|name-part|bin Rashid}};Mr.;
FN:{{p|khalid}}
ORG:{{o|Al Falasi General Trading LLC|company-eponym}}
TITLE:Managing Partner
TEL;TYPE=CELL:{{ph|+971 50 555 0139}}
TEL;TYPE=WORK:{{ph|+971 4 555 0172}}
EMAIL;TYPE=WORK:{{e|khalid@alfalasi-trading.example}}
ADR;TYPE=WORK:;;{{a|Office 1204\, Al Manara Tower\, Business Bay}};{{l|Dubai}};;;UAE
BDAY:{{d|1979-06-21|1979-06-21}}
NOTE:Emirates ID {{g|ae-eid|784-1987-4829307-7}}. Assistant: {{p|zainab}} ({{e|zainab.chaudhry@alfalasi-trading.example}}). Do not call during Friday prayers.
REV:{{d|2025-11-02|2025-11-02T09:00:00Z}}
END:VCARD

BEGIN:VCARD
VERSION:3.0
N:{{p|khalid|alias|Alfalasi}};{{p|khalid|first}};;;
FN:{{p|khalid|alias|Khalid Alfalasi}}
NICKNAME:{{p|khalid|teknonym|Abu Rashid}}
ORG:{{o|Al Falasi Trading|company-eponym}}
TEL;TYPE=CELL:{{ph|00971505550139}}
TEL;TYPE=HOME:{{ph|+971 4 555 0111}}
EMAIL:{{e|k.alfalasi@gulfmail.example}}
ADR;TYPE=HOME:;;{{a|Villa 27\, Street 14B\, Jumeirah 2}};{{l|Dubai}};;;AE
item1.X-ABRELATEDNAMES:{{p|layla}}
item1.X-ABLabel:sister
NOTE:IBAN for the deposit refund {{f|ae-iban|AE10 0260 0010 4829 3175 601}}
REV:{{d|2026-04-18|2026-04-18T16:42:10Z}}
END:VCARD`)),

	data("xml-soap", "xml,soap",
		usr(`The partner API rejects this request with "schema validation failed at PartyIdentification". Which element is wrong?

<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:pay="urn:example:payments:v3">
  <soapenv:Header>
    <pay:Auth>
      <pay:ClientId>corvane-prod</pay:ClientId>
      <pay:ApiKey>{{s|api-key|pk_9Xw2LmQ7vRt4ZpK8cYb3NfH6sJd1AeG5uTo0}}</pay:ApiKey>
    </pay:Auth>
    <pay:MessageId>{{n|uuid|c9bf9e57-1685-4c89-bafb-ff5af830be8a}}</pay:MessageId>
  </soapenv:Header>
  <soapenv:Body>
    <pay:CreditTransfer>
      <pay:RequestedExecutionDate>{{d|2026-05-25|2026-05-25}}</pay:RequestedExecutionDate>
      <pay:Debtor>
        <pay:Name>{{o|Corvane Freight Ltd}}</pay:Name>
        <pay:IBAN>{{f|gb-iban|GB09 LOYD 3096 1745 8203 61}}</pay:IBAN>
      </pay:Debtor>
      <pay:Creditor>
        <pay:Name>{{p|agnieszka}}</pay:Name>
        <pay:NameLatin>{{p|agnieszka|ascii-fold|AGNIESZKA SZCZEPANSKA}}</pay:NameLatin>
        <pay:IBAN>{{f|pl-iban|PL97114020040000310274829153}}</pay:IBAN>
        <pay:PartyIdentification scheme="PESEL">{{g|pl-pesel|86092304686}}</pay:PartyIdentification>
        <pay:PostalAddress>{{a|ul. Świętojańska 71/4, 81-389 Gdynia}}</pay:PostalAddress>
        <pay:Contact phone="{{ph|+48 58 555 0127}}" email="{{e|a.szczepanska@baltyk-cargo.example}}"/>
      </pay:Creditor>
      <pay:Amount currency="PLN">{{n|money|4820.00}}</pay:Amount>
      <pay:RemittanceInfo>Zwrot kosztów podróży, {{p|agnieszka|initial-last|A. Szczepańska}}, delegacja {{d|2026-05-11|11.05.2026}}-{{d|2026-05-14|14.05.2026}}</pay:RemittanceInfo>
      <!-- approved by {{p|chad}} / second approver {{p|ingrid|initials|IAT}} -->
    </pay:CreditTransfer>
  </soapenv:Body>
</soapenv:Envelope>`)),

	data("fixed-width-report", "fixed-width,report",
		usr(`Parse this mainframe-style report into CSV. Column positions are fixed. The name column is SURNAME GIVEN and gets truncated at 20 characters.

RUN DATE {{d|2026-06-30|30JUN2026}}   PAYROLL EXCEPTION REPORT   PAGE 001   REGION EMEA-NORTH
EMPNO   NAME                   COUNTRY  ID-NUMBER        BANK-ACCOUNT                  EXC  AMOUNT
------  ---------------------  -------  ---------------  ----------------------------  ---  ---------
300417  {{p|linnea|caps-reversed|ÅKERBLOM LINNÉA}}        SE       {{g|se-pnr|840612-2385}}      {{f|se-iban|SE9080000832790441682395}}      E12    1204.50
300418  {{p|svenerik|caps-reversed|BERGQVIST SVEN-ERIK}}    SE       {{g|se-pnr|791130-4173}}      {{f|se-account|8327-9 044 168 239}}            E07     310.00
300502  {{p|aino|caps-reversed|MÄKIRANTA AINO}}         FI       {{g|fi-hetu|140390-152K}}      {{f|fi-iban|FI7040550012394827}}            E12     988.10
300611  {{p|mikkel|caps-reversed|ØSTERGAARD MIKKEL}}      DK       {{g|dk-cpr|020583-1436}}      {{f|dk-iban|DK2530004829173056}}            E03      75.00
300612  {{p|torbjorn|caps-reversed|HAUGLAND TORBJØRN}}      NO       {{g|no-fnr|19087534106}}      {{f|no-iban|NO1736240751928}}               E12    2210.00
300700  {{p|gudrun|caps-truncated|VILHJÁLMSDÓTTIR GUÐR}}   IS       {{g|is-kennitala|120174-3399}}      {{f|is-account|0159-26-007654}}                E09     140.25
300701  {{p|bjarni|caps-reversed|GUÐRÚNARSON BJARNI}}     IS       {{g|is-kennitala|170499-2129}}      {{f|is-account|0159-26-007655}}                E09     140.25
300815  {{p|ingrid|caps-truncated|AASGAARD-THORSEN ING}}  NO       {{g|no-fnr|19087534297}}      {{f|no-account|3624.07.51928}}                 E01       0.00
------  ---------------------  -------  ---------------  ----------------------------  ---  ---------
EXC CODES  E01 MISSING TAX CARD  E03 ROUNDING  E07 ACCOUNT FORMAT  E09 NEW STARTER  E12 RETRO PAY
PREPARED BY {{p|anastasia|caps-reversed|SOKOLOVSKAYA ANASTASIA}} ({{p|anastasia|handle|ASOKOLOVSKAYA}})   REVIEWED BY {{p|ingrid|initials|IAT}}`)),

	data("ndjson-events", "ndjson,events",
		usr(`Write a jq filter that keeps only events where the actor is not a service account, and prints time, actor and action. Sample from the audit stream:

{"ts":"{{d|2026-05-05|2026-05-05T06:00:01.004Z}}","actor":{"type":"service","id":"svc-backup"},"action":"snapshot.create","target":"db-berth-prod","ip":"192.0.2.10"}
{"ts":"{{d|2026-05-05|2026-05-05T07:42:19.551Z}}","actor":{"type":"user","id":"u_1014","email":"{{e|wade.kimathi@lumenwharf.example}}","name":"{{p|wade}}"},"action":"user.invite","target":"{{e|mostafa.shalaby@nilecargo.example}}","meta":{"invitee_name":"{{p|mostafa}}","role":"viewer"},"ip":"203.0.113.61"}
{"ts":"{{d|2026-05-05|2026-05-05T07:44:02.118Z}}","actor":{"type":"user","id":"u_1014","email":"{{e|wade.kimathi@lumenwharf.example}}","name":"{{p|wade}}"},"action":"apikey.reveal","target":"key_3fa9","meta":{"last4":"AeG5"},"ip":"203.0.113.61"}
{"ts":"{{d|2026-05-05|2026-05-05T08:15:40.730Z}}","actor":{"type":"user","id":"u_2291","email":"{{e|mostafa.shalaby@nilecargo.example}}","name":"{{p|mostafa|native-script|مصطفى شلبي}}"},"action":"profile.update","target":"u_2291","meta":{"phone":"{{ph|+20 2 5550 0171}}","national_id":"{{g|eg-nid|29001011234567}}","display":"{{p|mostafa|first}} {{p|mostafa|last}}"},"ip":"198.51.100.31"}
{"ts":"{{d|2026-05-05|2026-05-05T08:15:41.002Z}}","actor":{"type":"service","id":"svc-webhook"},"action":"webhook.deliver","target":"https://hooks.nilecargo.example/berth?sig={{s|webhook-signature|wh`+`sec_Q7xVp4LtZ8cW3bYn6HsKd1AeG5uRo0Mj}}","status":200}
{"ts":"{{d|2026-05-05|2026-05-05T09:03:27.411Z}}","actor":{"type":"user","id":"u_0007","email":"{{e|younes.benkirane@atlas-transit.example}}","name":"{{p|younes}}"},"action":"export.download","target":"shipments_{{d|2026-04-30|2026-04-30}}.csv","meta":{"rows":48213,"requested_for":"{{p|younes|ascii-fold|Younes Benkirane}}","cin":"{{g|ma-cin|BK482917}}"},"ip":"203.0.113.145"}
{"ts":"{{d|2026-05-05|2026-05-05T09:03:30.000Z}}","actor":{"type":"service","id":"svc-metrics"},"action":"heartbeat","build":"{{n|hash|4be1a02d}}","version":"{{n|version|4.18.2}}"}`)),
}

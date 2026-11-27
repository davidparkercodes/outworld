package main

var itemsIntlB = []def{
	intl("pt-pt-senhorio", "portuguese,portugal,letter",
		usr(`Recebi esta carta do senhorio e queria responder de forma educada mas firme. Pode ajudar-me? Vivo em Portugal há pouco tempo e o meu português escrito ainda é fraco.

{{l|Porto}}, {{d|2026-09-14|14 de setembro de 2026}}

Exma. Senhora
D. {{p|beatriz}}
{{a|Rua de Cedofeita, 318, 2.º Esq.
4050-174 Porto}}

Assunto: atualização de renda e dados para recibo eletrónico

Exma. Senhora,

Na qualidade de senhorio do locado acima identificado, venho comunicar que a renda mensal passará de 850,00 € para 910,00 € a partir de {{d|2026-11-01|1 de novembro de 2026}}, nos termos do contrato celebrado em {{d|2022-11-01|01/11/2022}}.

Solicito ainda que me confirme o seu NIF, que julgo ser {{g|pt-nif|247819360}}, para emissão dos recibos no Portal das Finanças. O pagamento deverá continuar a ser feito por transferência para o IBAN {{f|pt-iban|PT02 0035 0683 0000 4821 9307 1}}.

Para qualquer esclarecimento, pode contactar-me pelo {{ph|22 555 01 46}} ou o meu filho, Eng.º {{p|mateus|first}} {{p|mateus|name-part|Figueiredo}}, pelo {{ph|+351 91 555 01 73}}.

Com os melhores cumprimentos,

{{p|+António Figueiredo dos Anjos|full}}
NIF {{g|pt-nif|213509873}}

O que eu sei: a lei só permite aumentar pelo coeficiente anual, e 60 euros parece-me muito acima. A minha vizinha, a D. {{p|+Graça|first}}, diz que o Sr. {{p|+António Figueiredo dos Anjos|first|António}} tentou o mesmo com ela em {{n|month|março}}. O meu NIF está certo.`)).on("2026-11-01", "beatriz"),

	intl("it-condominio", "italian,italy,email",
		usr(`Mi aiuti a rispondere all'amministratore di condominio? Vorrei contestare la ripartizione ma senza litigare. Ecco la sua mail.

Da: {{p|gianmarco}} <{{e|amministrazione@studiodallorto.example}}>
A: {{p|chiara}} <{{e|chiara.deangelis@postacerta.example}}>
Cc: {{p|ludovica}} <{{e|ludo.sanfilippo@postacerta.example}}>
Data: {{d|2026-03-05|giovedì 5 marzo 2026, 18:47}}
Oggetto: Condominio {{a|Via dei Serpenti 41}}, conguaglio riscaldamento 2025

Gentile dott.ssa {{p|chiara|title-last|De Angelis}},

in riferimento alla Sua del {{d|2026-02-27|27 febbraio}}, Le confermo che il conguaglio a Suo carico è di € 612,40, calcolato sui millesimi dell'interno 7 (ex proprietà {{p|giuseppina|last}}).

Come deliberato dall'assemblea del {{d|2025-11-18|18/11/2025}}, il pagamento va effettuato entro il {{d|2026-03-31|31 marzo p.v.}} sul conto del condominio:
IBAN {{f|it-iban|IT64 K030 6909 6061 0000 0048 215}}
intestato a Condominio Serpenti 41, causale "conguaglio 2025 int. 7 {{p|chiara|caps-last|DE ANGELIS}}".

Per la voltura mi occorre ancora il codice fiscale della precedente intestataria, la sig.ra {{p|giuseppina}}; a me risulta {{g|it-cf|LPRGPP41B52G273S}}, me lo conferma? Il mio, per la delega, è {{g|it-cf|DLLGMR79S03F205B}}.

Resto a disposizione al numero {{ph|06 5550 0129}} (lun-ven 9-13) o al cellulare {{ph|333 555 0164}}.

Cordiali saluti
geom. {{p|gianmarco}}
{{o|Studio Dall'Orto|company-eponym}}, {{a|Via Cavour 212, 00184 Roma}}

Nota mia: nonna {{p|giuseppina|nickname}} ha lasciato l'appartamento il {{d|2025-06-30|30 giugno 2025}}, quindi metà stagione non era ancora mia. E {{p|ludovica|nickname}}, che è in copia, è l'inquilina dell'interno 8 e ha lo stesso problema.`)).on("2026-03-31", "chiara", "2025-06-30", "giuseppina"),

	intl("nl-gemeente-brief", "dutch,netherlands,letter",
		usr(`Kun je deze brief van de gemeente in gewone taal uitleggen? En klopt het dat ik binnen zes weken bezwaar moet maken?

{{o|Gemeente Zwolle}}
Afdeling Belastingen
{{a|Grote Kerkplein 15, 8011 PK Zwolle}}

Datum: {{d|2026-02-27|27 februari 2026}}
Kenmerk: {{n|identifier|WOZ-2026-0041773}}
Behandeld door: mw. {{p|anouk|initial-last|A. Hoogendoorn}}, tel. {{ph|038 555 0119}}

Aan de erven van dhr. {{p|willem|initial-last|W. van der Meulen}}
p/a mw. {{p|femke|initial-last|F. van der Meulen}}
{{a|Assendorperstraat 212
8012 CE Zwolle}}

Geachte mevrouw {{p|femke|title-last|Van der Meulen}},

Hierbij ontvangt u de WOZ-beschikking 2026 voor de woning aan de {{a|Assendorperstraat 212}}. De waarde is vastgesteld op € 418.000 (peildatum {{d|2025-01-01|1 januari 2025}}).

Volgens de Basisregistratie Personen stond op dit adres tot {{d|2026-02-19|19-02-2026}} ingeschreven: {{p|willem|caps-last|VAN DER MEULEN}}, {{p|willem|first}} (BSN eindigend op 362). De aanslag gemeentelijke belastingen ad € 1.284,16 wordt naar u als contactpersoon van de erven gezonden. Wij hebben uw BSN {{g|nl-bsn|304561824}} en dat van de heer {{p|maarten}} ({{g|nl-bsn|217849362}}) van de notaris ontvangen.

U kunt in termijnen betalen via automatische incasso van rekening {{f|nl-iban|NL07 INGB 0005 2931 84}}. Bent u het niet eens met deze beschikking, dan kunt u binnen zes weken na dagtekening bezwaar maken, dus uiterlijk {{d|2026-04-10|10 april 2026}}.

Met vriendelijke groet,
namens het college van burgemeester en wethouders,

{{p|anouk}}
{{e|belastingen@zwolle.example}}

Even voor de duidelijkheid: die {{p|anouk|first}} van de gemeente is niet mijn tante {{p|anouk|first}}, die heet toevallig hetzelfde. En mijn broer heet {{p|maarten|ascii-fold|Maarten Ten Brinke}}, de gemeente schrijft "ten" met een kleine letter, het paspoort met een hoofdletter.`)).on("2026-04-10", "femke", "2026-02-19", "willem"),

	intl("pl-urzad-email", "polish,poland,email",
		usr(`Dostałam taką wiadomość z urzędu od pana {{p|bartosz|inflected|Bartosza Wiśniewskiego}} i nie wiem, czy to nie jest phishing. Możesz sprawdzić, czy dane się zgadzają z tym, co podaję niżej, i pomóc mi odpisać?

Od: {{p|bartosz}} <{{e|b.wisniewski@um-gdynia.example}}>
Do: {{p|agnieszka}} <{{e|aga.szczepanska@poczta-baltyk.example}}>
Data: {{d|2026-06-03|środa, 3 czerwca 2026 09:15}}
Temat: Wniosek o wymianę dowodu osobistego, braki formalne

Szanowna Pani {{p|agnieszka|inflected|Agnieszko}},

w związku z wnioskiem złożonym {{d|2026-05-27|27.05.2026 r.}} uprzejmie informuję, że fotografia nie spełnia wymogów. Prosimy o dostarczenie nowej do dnia {{d|2026-06-17|17 czerwca 2026 r.}}

Dane we wniosku:
Nazwisko i imię: {{p|agnieszka|caps-last|SZCZEPAŃSKA}} {{p|agnieszka|first}}
Nazwisko rodowe: {{p|agnieszka|maiden|Wojciechowska}}
PESEL: {{g|pl-pesel|86092304686}}
Adres zameldowania: {{a|ul. Świętojańska 71/4, 81-389 Gdynia}}
Seria i numer dotychczasowego dowodu: {{g|pl-dowod|CBA 482917}}

Odbiór dokumentu będzie możliwy osobiście w pokoju 114. W razie pytań proszę o kontakt pod numerem {{ph|58 555 01 38}}.

Z poważaniem
{{p|bartosz}}
inspektor, {{o|Urząd Miasta Gdyni}}

Moje dane: PESEL się zgadza, adres też. Ale nie rozumiem, skąd pan {{p|bartosz|last}} zna moje nazwisko panieńskie. Siostra, {{p|katarzyna|nickname}} (ta sama {{p|katarzyna|last}}, PESEL {{g|pl-pesel|79021423875}}), mówi, że to normalne, bo jest w rejestrze. Mój numer: {{ph|+48 601 555 019}}.`)).on("2026-06-17", "agnieszka"),

	intl("sv-forsakringskassan", "swedish,sweden,chat",
		usr(`Hej! Kan du hjälpa mig att förstå vad Försäkringskassan vill ha? Jag klistrar in chatten med handläggaren. Jag är inte så bra på myndighetssvenska.

Chatt med {{o|Försäkringskassan}}, {{d|2026-04-22|2026-04-22}}

Handläggare {{p|linnea|first}}: Hej och välkommen! Jag heter {{p|linnea}}. För att kunna hjälpa dig behöver jag ditt personnummer.
Du: Hej. {{g|se-pnr|791130-4173}}
Handläggare {{p|linnea|first}}: Tack {{p|svenerik|first}}. Jag ser att du har ansökt om föräldrapenning från och med {{d|2026-06-15|15 juni}}. Stämmer det att den andra föräldern är {{p|adaeze}}, personnummer {{g|se-pnr|880229-1461}}?
Du: Ja, {{p|adaeze|first}}. Men hon heter bara {{p|adaeze|alias|Adaeze Lindqvist}} i folkbokföringen tror jag.
Handläggare {{p|linnea|first}}: Hos oss står {{p|adaeze|caps-reversed|OKPARA-LINDQVIST, ADAEZE}}. Det är Skatteverket som avgör, vi följer deras uppgift. Vi saknar ett arbetsgivarintyg från {{o|Lumenwharf AB}}. Kan din chef skicka det?
Du: Min chef är {{p|takumi}}, han sitter i {{l|Osaka}}. Går det bra på engelska?
Handläggare {{p|linnea|first}}: Ja. Det kan mejlas till {{e|arbetsgivarintyg@forsakringskassan.example}} senast {{d|2026-05-13|13 maj}}. Ange ärendenummer {{n|identifier|FK-2026-4417733}}.
Du: Och utbetalningen, går den till mitt konto? Clearingnummer {{f|se-clearing|8327-9}}, konto {{f|se-account|044 168 239-5}}.
Handläggare {{p|linnea|first}}: Det stämmer med det vi har. Vi skickar beslutet till {{a|Västra Hamngatan 14 lgh 1102, 411 17 Göteborg}}. Vill du ha sms-avisering till {{ph|070-174 06 25}}?
Du: Ja tack. Tack {{p|linnea|ascii-fold|Linnea}}!
Handläggare {{p|linnea|first}}: Tack själv, ha en fin dag {{p|svenerik|nickname}}! Nej förlåt, {{p|svenerik|first}} ska det stå, jag läste fel i anteckningen.`)).on("2026-06-15", "svenerik", "2026-05-13", "takumi"),

	intl("tr-banka-sms", "turkish,turkey,chat",
		usr(`Merhaba, annemin telefonuna gelen bu mesajlar gerçek mi yoksa dolandırıcılık mı? Hangileri bankadan, hangileri sahte, ayırabilir misin? Annemin bilgilerini de aşağıya yazdım, karşılaştırmak için.

1) {{o|MARMARABANK}}: Sayin {{p|gulsen|ascii-fold|GULSEN KARABULUT}}, {{d|2026-03-14|14.03.2026}} tarihli 12.500,00 TL havale isleminiz gerceklesmistir. Alici: {{p|oguz|ascii-fold|OGUZHAN CELIKKAYA}}. Bilginiz disindaysa {{ph|0850 555 01 44}}

2) Sayın müşterimiz, hesabınız bloke edilmiştir. Kimlik doğrulama için TC kimlik numaranızı ve şifrenizi marmarabank-dogrulama.example adresine giriniz. Son gün: {{d|2026-03-15|15 Mart}}.

3) {{o|MARMARABANK}}: Tek kullanimlik sifreniz {{s|otp|482917}}. Kimseyle paylasmayin. B001

4) Anne ben {{p|oguz|nickname}}, telefonum bozuldu bu yeni numaram {{ph|0532 555 01 91}}. Acil 8.000 TL lazım, şu IBAN'a atar mısın: {{f|tr-iban|TR18 0001 0017 4538 2910 6750 02}} hesap {{p|timur}} adına.

Annemin bilgileri:
Adı soyadı: {{p|gulsen}}
TC Kimlik No: {{g|tr-tckn|34827190576}}
Doğum tarihi: {{d|1958-10-29|29 Ekim 1958}}
Adres: {{a|Bağdat Caddesi No: 318 D: 7, 34728 Kadıköy/İstanbul}}
Telefon: {{ph|+90 532 555 01 20}}

Ben {{p|oguz}}, oğluyum. TC'm {{g|tr-tckn|51739082690}}. Telefonum bozulmadı ve {{p|timur|first}} diye birini tanımıyoruz. Annem 4. mesaja inanıp parayı göndermek üzereydi, komşumuz {{p|+Nermin|honorific|Nermin Hanım}} durdurmuş.`)).on("2026-03-14", "gulsen"),

	intl("vi-thue-nha", "vietnamese,vietnam,chat",
		usr(`Giúp mình soạn lại hợp đồng thuê nhà ngắn gọn từ mấy tin nhắn Zalo này với. Chủ nhà nhắn lộn xộn quá.

[{{d|2026-08-02|02/08/2026}} 19:20] {{p|thanhhuong|honorific|Cô Hương}}: Chào con. Cô là {{p|thanhhuong}}, chủ nhà ở {{a|25/7 Nguyễn Trãi, Phường Bến Thành, Quận 1}}. Con là {{p|maianh|first}} phải không?
[{{d|2026-08-02|02/08/2026}} 19:21] {{p|maianh|first}}: Dạ con chào cô. Con là {{p|maianh}} ạ.
[{{d|2026-08-02|02/08/2026}} 19:23] {{p|thanhhuong|honorific|Cô Hương}}: Giá thuê 9 triệu một tháng, cọc 2 tháng. Dọn vào ngày {{d|2026-09-01|1 tháng 9}} được không con
[{{d|2026-08-02|02/08/2026}} 19:24] {{p|maianh|first}}: Dạ được ạ. Cô cho con xin số tài khoản để chuyển cọc.
[{{d|2026-08-02|02/08/2026}} 19:26] {{p|thanhhuong|honorific|Cô Hương}}: {{o|Vietcombank}} stk {{f|vn-account|0071004829173}} chủ tk {{p|thanhhuong|ascii-fold|NGUYEN THANH HUONG}}. Con ghi nội dung "{{p|maianh|ascii-fold|TRAN THI MAI ANH}} coc nha"
[{{d|2026-08-02|02/08/2026}} 19:27] {{p|thanhhuong|honorific|Cô Hương}}: Cô cần số căn cước của con để làm tạm trú
[{{d|2026-08-02|02/08/2026}} 19:29] {{p|maianh|first}}: Dạ CCCD của con là {{g|vn-cccd|079190004829}}, sinh ngày {{d|1990-05-14|14/05/1990}}. Số điện thoại con {{ph|090 555 01 47}}.
[{{d|2026-08-02|02/08/2026}} 19:31] {{p|thanhhuong|honorific|Cô Hương}}: Ok con. Chồng cô là chú {{p|+Bảo|first}} sẽ đưa chìa khoá. Số chú {{ph|0283 555 0162}}
[{{d|2026-08-02|02/08/2026}} 19:32] {{p|maianh|first}}: Dạ con cảm ơn cô {{p|thanhhuong|name-part|Hương}} ạ.`)).on("2026-09-01", "maianh"),

	intl("id-tl-mixed", "indonesian,tagalog,chat",
		usr(`Two short messages from our regional agents, one in Indonesian and one in Taglish. Translate both to English for the ops channel and keep names and numbers exactly.

--- from {{l|Surabaya}} ---
Selamat pagi Bu {{p|wulan|nickname}}. Saya {{p|bambang}} dari {{o|PT Nugroho Samudra Logistik|company-eponym}}. Kontainer {{n|identifier|TCLU 482917 3}} sudah tiba di {{l|Tanjung Perak}} tanggal {{d|2026-07-09|9 Juli 2026}}. Mohon kirim surat kuasa atas nama saya, NIK {{g|id-nik|3578101708750003}}, NPWP {{g|id-npwp|09.254.294.3-407.000}}. Biaya penumpukan bisa ditransfer ke {{o|BCA|company}} {{f|id-account|482 917 3056}} a.n. {{p|bambang|caps|BAMBANG PRASETYO NUGROHO}}. Hubungi saya di {{ph|0812-5550-0193}} atau {{p|bambang|honorific|Pak Bambang}} saja di WhatsApp. Terima kasih, salam untuk Ibu {{p|khin|first}}.

--- from {{l|Manila}} ---
Hi po Ma'am {{p|khin|first}}, si {{p|jejomar}} po ito ({{p|jejomar|nickname}} na lang). Na-release na po yung shipment kahapon, {{d|2026-07-08|July 8}}, pero may kulang pa sa BOC. Kailangan po ng TIN ng consignee, si Mrs. {{p|malou}}: {{g|ph-tin|482-917-305-000}}. Yung address niya po sa waybill is {{a|Unit 12-C, Dela Rosa Tower, 118 Dela Rosa St., Legazpi Village, Makati 1229}} pero sabi ni Ate {{p|malou|nickname}} lumipat na raw sila sa {{l|Quezon City}}. Paki-confirm na lang po. Pwede niyo po akong tawagan sa {{ph|+63 917 555 0148}} or kay {{p|malou|honorific|Ate Malou}} mismo sa {{ph|(02) 8555 0127}}. GCash po niya for the storage fee: {{ph|0917 555 0162}}. Salamat po! {{p|jejomar|nickname}}`)),
}

package main

var itemsIntlA = []def{
	intl("es-gestoria-email", "spanish,spain,tax",
		usr(`Necesito contestar a la gestoría hoy. ¿Me resumes qué me piden y me redactas la respuesta? Les mando lo que falta en el mismo correo.

De: {{p|inaki}} <{{e|inaki@gestoria-etxeberria.example}}>
Para: {{p|maricarmen}} <{{e|maricarmen.ortiz@correo-rapido.example}}>
Fecha: {{d|2026-04-08|miércoles, 8 de abril de 2026, 10:14}}
Asunto: Renta 2025, documentación pendiente

Buenos días, {{p|maricarmen|nickname}}:

Para presentar tu declaración antes del {{d|2026-04-30|30 de abril}} me falta lo siguiente:

1. Fotocopia del DNI por las dos caras. En la ficha tengo el {{g|es-dni|47382915V}}, confírmame que es correcto.
2. El certificado de retenciones de {{o|Conservas Ortiz de Zárate, S.L.|company-eponym}}. Ya sé que es la empresa de tu hermano, pero lo necesito igualmente.
3. El IBAN para la devolución. El año pasado usamos el {{f|es-iban|ES09 0182 4170 5802 0163 9471}}; si has cambiado de banco, dímelo.
4. Los datos de tu inquilina de {{l|Vitoria-Gasteiz}}: nombre completo, NIE y fecha de inicio del contrato.
5. La referencia catastral del piso de la {{a|calle Dato, 14, 3.º izda., 01005 Vitoria-Gasteiz}}.

Si te viene mejor, llámame al {{ph|945 55 01 37}} o pásate por el despacho ({{a|Avenida de Gasteiz 62, bajo}}), estoy hasta las 14:00.

Un abrazo,
{{p|inaki|first}}
{{o|Gestoría Etxeberria|company-eponym}}

Mis datos para contestar: el DNI es correcto. La inquilina es {{p|florencia}}, argentina, NIE {{g|es-nie|Y4829173S}}, contrato desde el {{d|2025-09-01|1 de septiembre de 2025}}. El IBAN ahora es {{f|es-iban|ES98 2038 5778 9830 0076 0236}}. Mi móvil nuevo: {{ph|+34 612 55 01 84}}. Firmo siempre como {{p|maricarmen|alias|Mª Carmen Ortiz de Zárate}}, que {{p|inaki|first}} se lía con el nombre largo.`)).on("2026-04-30", "maricarmen", "2025-09-01", "florencia"),

	intl("es-mx-whatsapp", "spanish,mexico,chat",
		usr(`me ayudas a armar un mensaje para mi tío con todo lo que me pidió el banco? él no entiende de apps, que sea paso a paso. te pego el chat

[{{d|2026-02-16|16/02/26}} 18:02] {{p|ernesto|display|Tío Neto}}: Mija ya fui al banco y no me dejaron abrir la cuenta. Que me falta un papel
[{{d|2026-02-16|16/02/26}} 18:03] {{p|ximena|nickname}}: qué papel tío? llevaste la INE y el comprobante de domicilio?
[{{d|2026-02-16|16/02/26}} 18:05] {{p|ernesto|display|Tío Neto}}: Sí. Me pidieron la CURP impresa y el RFC con homoclave. Yo qué sé qué es eso
[{{d|2026-02-16|16/02/26}} 18:06] {{p|ximena|nickname}}: yo te las saco. tu curp es {{g|mx-curp|ZABE680130HVZVRR05}} verdad? naciste el {{d|1968-01-30|30 de enero del 68}} en {{l|Xalapa}}
[{{d|2026-02-16|16/02/26}} 18:06] {{p|ernesto|display|Tío Neto}}: Sí mija
[{{d|2026-02-16|16/02/26}} 18:09] {{p|ximena|nickname}}: listo. RFC {{g|mx-rfc|ZABE680130QT4}}. te lo mando en pdf. a nombre de {{p|ernesto|caps|ERNESTO ZAVALETA BERMUDEZ}} sin acento, así sale en el SAT
[{{d|2026-02-16|16/02/26}} 18:10] {{p|ernesto|display|Tío Neto}}: Y la cuenta donde me va a depositar don {{p|fernando|first}}? Me dio una clave larguísima
[{{d|2026-02-16|16/02/26}} 18:11] {{p|ximena|nickname}}: esa es la clabe de él no la tuya. la tuya te la dan cuando abras. la de don {{p|fernando|nickname}} es {{f|mx-clabe|646180110458293617}} guárdala
[{{d|2026-02-16|16/02/26}} 18:13] {{p|ernesto|display|Tío Neto}}: Ah. Y la señorita del banco, la licenciada {{p|xiadani|first}} {{p|xiadani|name-part|Coyotl}}, me dio su tarjeta. Dice sucursal {{l|Coatepec}}, tel {{ph|228 555 0146}}
[{{d|2026-02-16|16/02/26}} 18:14] {{p|ximena|nickname}}: va. yo le marco mañana. tu domicilio sigue siendo {{a|Calle Úrsulo Galván 118, Col. Centro, 91000 Xalapa, Ver.}}?
[{{d|2026-02-16|16/02/26}} 18:15] {{p|ernesto|display|Tío Neto}}: Sí. Dile que voy el jueves {{d|2026-02-19|19 de febrero}} a las 10
[{{d|2026-02-16|16/02/26}} 18:15] {{p|ximena|nickname}}: 👍 y tío no le des tu NIP a nadie ni a la licenciada
[{{d|2026-02-16|16/02/26}} 18:16] {{p|ernesto|display|Tío Neto}}: Ya se lo di a tu tía. Es {{s|pin|4471}}
[{{d|2026-02-16|16/02/26}} 18:16] {{p|ximena|nickname}}: TÍO`),
		ast(`Claro. Un mensaje paso a paso para don {{p|ernesto|first}}:

"Tío, para abrir la cuenta el jueves lleve: 1. Su INE. 2. El comprobante de domicilio. 3. La CURP impresa que le mando en PDF. 4. La constancia del RFC, también en PDF. Pregunte por la licenciada {{p|xiadani|first}} en la sucursal de {{l|Coatepec}}. La clave larga que le dio don {{p|fernando|first}} no es para el banco, guárdela. Y por favor cambie su NIP en el cajero, ya no es secreto."

No incluí los números completos de CURP y RFC en el mensaje; es mejor que solo vayan en los PDF.`),
		usr(`Perfecto. Agrégale que la cita es el {{d|2026-02-19|jueves 19}} a las 10 y que si se pierde me marque a mí, {{p|ximena}}, al {{ph|575-555-0153}}, no a la {{p|xiadani|nickname}}. Y fírmalo "tu sobrina {{p|ximena|nickname}}".`)),

	intl("es-ar-cl-chat", "spanish,argentina,chile",
		usr(`Che, necesito que me armes una planilla con estos datos para la contadora, y decime si falta algo para darme de alta como monotributista. Te copio lo que me pasó mi socio y lo mío.

Lo mío:
{{p|florencia}}, DNI {{g|ar-dni|28.461.935}}, CUIL {{g|ar-cuil|27-28461935-3}}
Nacida el {{d|1981-04-23|23/4/1981}} en {{l|Rosario}}
Domicilio fiscal: {{a|Av. Pellegrini 1840, piso 7 "B", S2000BTP Rosario, Santa Fe}}
CBU: {{f|ar-cbu|0170482640000319572645}}, alias {{f|ar-alias|flor.eche.mp}}
Cel: {{ph|+54 9 341 555-0174}}

Mi socio, que es chileno y factura desde allá:
"Hola {{p|florencia|nickname}}, acá van mis datos po. {{p|cristobal}}, RUT {{g|cl-rut|15.482.937-7}}. Dirección {{a|Av. Providencia 2594, depto 1108, Providencia, Santiago}}. Cuenta RUT del {{o|BancoEstado}} n.º {{f|cl-account|15482937}}. Mi fono es el {{ph|+56 9 5550 0143}}. La boleta de marzo la emití el {{d|2026-03-31|31 de marzo}}, te mando el PDF. Ojo que mi papá se llama igual, {{p|cristobal|first}} {{p|cristobal|name-part|Aravena}} a secas, y su RUT es el {{g|cl-rut|9.371.508-K}}, no los confundan como la otra vez. Saludos a {{p|wayra|first}}. {{p|cristobal|nickname}}"

La contadora es {{p|wayra}}, {{e|wayra.yupanqui@estudio-mamani.example}}. Me dijo que el trámite hay que hacerlo antes del {{d|2026-04-20|20 de abril}} y que le mande todo junto, no de a pedacitos como hace {{p|cristobal|nickname}}.`)).on("2026-03-31", "cristobal", "2026-04-20", "florencia"),

	intl("fr-courrier-mutuelle", "french,france,letter",
		usr(`J'ai reçu ce courrier de ma mutuelle et je ne comprends pas pourquoi le remboursement est refusé. Pouvez-vous m'expliquer et m'aider à rédiger une réclamation ?

{{o|Mutuelle des Monts d'Arrée}}
Service Prestations, {{a|12 quai de la Douane, 29200 Brest}}

{{l|Brest}}, le {{d|2026-03-12|12 mars 2026}}

Madame {{p|solene|caps-last|KERBRAT}} {{p|solene|first}}
{{a|14 rue de Siam
29200 BREST}}

N° adhérent : {{f|policy|MA-2948-31750-K}}
N° de sécurité sociale : {{g|fr-nir|2 87 03 29 019 042 13}}
Objet : refus de prise en charge, soins du {{d|2026-02-03|03/02/2026}}

Madame,

Nous avons bien reçu votre demande de remboursement concernant la consultation du Docteur {{p|thibault|title-last|de La Rochefoucault-Mbaye}}, ophtalmologue à {{l|Nantes}}, en date du {{d|2026-02-03|3 février 2026}}, pour un montant de 85,00 €.

Après étude de votre dossier, nous ne pouvons y donner une suite favorable : le praticien n'ayant pas été déclaré comme médecin correspondant par votre médecin traitant, le Dr {{p|genevieve|title-last|Tremblay-Ouellet}}, la consultation est considérée hors parcours de soins.

Le remboursement de vos lunettes ({{o|Optique Vasseur|company-eponym}}, facture du {{d|2026-02-10|10 février}}) sera effectué par virement sur le compte {{f|fr-iban|FR83 1027 8060 4100 0204 8390 147}} sous dix jours.

Pour toute réclamation, vous pouvez écrire à l'adresse ci-dessus, nous joindre au {{ph|02 99 00 12 80}} ou par courriel : {{e|reclamations@mutuelle-arree.example}}.

Nous vous prions d'agréer, Madame, l'expression de nos salutations distinguées.

Votre conseillère, {{p|aminata|initial-last|A. Diallo-Lefèvre}}

Précision : c'est bien le Dr {{p|genevieve|last}} qui m'a adressée à l'ophtalmo, j'ai sa lettre datée du {{d|2026-01-20|20 janvier}}. Mon numéro : {{ph|06 39 98 27 15}}.`),
		ast(`Le refus repose sur un seul point : la mutuelle considère que le Dr {{p|thibault|title-last|de La Rochefoucault-Mbaye}} a été consulté hors parcours de soins, parce que votre médecin traitant ne l'aurait pas déclaré comme correspondant. Si le Dr {{p|genevieve|title-last|Tremblay-Ouellet}} vous a bien adressée par écrit le {{d|2026-01-20|20 janvier}}, vous étiez dans le parcours et le refus n'est pas fondé. Je vous propose une réclamation courte, avec la lettre d'adressage en pièce jointe, envoyée en recommandé. Souhaitez-vous que je la rédige au nom de Mme {{p|solene}} ?`),
		usr(`Oui, à mon nom, {{p|solene}}, et adressée à ma conseillère Mme {{p|aminata|title-last|Diallo-Lefèvre}}. Ajoutez mon numéro d'adhérent et demandez une réponse avant le {{d|2026-04-15|15 avril}}. Ne mettez pas mon numéro de sécurité sociale dans la lettre, ils l'ont déjà.`)).on("2026-02-03", "thibault", "2026-01-20", "genevieve"),

	intl("fr-whatsapp-famille", "french,chat,senegal",
		usr(`tu peux me faire un résumé de qui arrive quand, et qui doit aller chercher qui ? c'est le groupe famille, y a du wolof dedans, laisse-le tel quel

[{{d|2026-12-09|09/12/2026}} 20:31] {{p|aminata|nickname}}: Salam la famille. {{p|+Ndèye Fatou Diallo|honorific|Yaay Fatou}} a son visa !! alhamdoulillah
[{{d|2026-12-09|09/12/2026}} 20:32] {{p|mamadou|nickname}}: Machallah. On prend le vol du {{d|2026-12-15|15 décembre}}, {{l|Dakar}} {{l|Lyon}} via {{l|Casablanca}}, arrivée 21h40
[{{d|2026-12-09|09/12/2026}} 20:33] {{p|etienne}}: Super. Je viens vous chercher à Saint-Exupéry. {{p|mamadou|first}} tu as toujours le même numéro ? Le {{ph|+221 77 555 01 42}} ?
[{{d|2026-12-09|09/12/2026}} 20:35] {{p|mamadou|nickname}}: Oui tonton {{p|etienne|ascii-fold|Etienne}}. Mais là-bas j'aurai la puce française, le {{ph|07 55 50 01 63}}
[{{d|2026-12-09|09/12/2026}} 20:36] {{p|aminata|nickname}}: {{p|etienne|first}} n'oublie pas le fauteuil pour maman, elle ne peut pas marcher jusqu'au parking
[{{d|2026-12-09|09/12/2026}} 20:40] {{p|thibault|display|Titi 🦅}}: Et moi je descends de {{l|Nantes}} le {{d|2026-12-23|23/12}} avec {{p|solene|first}}. Train de 14h04. Quelqu'un à la Part-Dieu ?
[{{d|2026-12-09|09/12/2026}} 20:41] {{p|aminata|nickname}}: {{p|thibault|nickname}} tu viens avec {{p|solene|first}} {{p|solene|first}} ou {{p|solene|first}} ta collègue
[{{d|2026-12-09|09/12/2026}} 20:41] {{p|thibault|display|Titi 🦅}}: 🙄 {{p|solene|first}} {{p|solene|last}}. Ma copine. Depuis deux ans {{p|aminata|first}}
[{{d|2026-12-09|09/12/2026}} 20:43] {{p|mamadou|nickname}}: Nanga def {{p|thibault|nickname}}. Dis à madame {{p|solene|last}} que yaay veut la rencontrer enfin
[{{d|2026-12-09|09/12/2026}} 20:45] {{p|etienne}}: Je note : {{d|2026-12-15|15/12}} aéroport (moi), {{d|2026-12-23|23/12}} gare (qui ?). Adresse pour les taxis au cas où : {{a|27 rue des Capucins, 69001 Lyon}}, code porte {{s|door-code|27A41}}
[{{d|2026-12-09|09/12/2026}} 20:46] {{p|aminata|nickname}}: {{p|etienne|lower-first}} on ne met pas le code dans le groupe 🤦🏾‍♀️`)),

	intl("de-steuerberater", "german,germany,tax",
		usr(`Kannst du mir helfen, auf diese Mail vom Steuerberater zu antworten? Ich verstehe Punkt 2 nicht und weiß nicht, ob ich die Nummern einfach so schicken soll.

Von: {{p|friedhelm}} <{{e|oberlaender@stb-oberlaender.example}}>
An: {{p|emine}} <{{e|emine.yh@nordpost.example}}>; {{p|matthias}} <{{e|matthias.hr@nordpost.example}}>
Datum: {{d|2026-05-06|Mittwoch, 6. Mai 2026 um 11:02}}
Betreff: Einkommensteuererklärung 2025, fehlende Angaben

Sehr geehrte Frau {{p|emine|title-last|Yıldırım-Hoffmann}}, sehr geehrter Herr {{p|matthias|title-last|Hoffmann-Reuter}},

für die Zusammenveranlagung 2025 benötige ich bis zum {{d|2026-05-29|29.05.2026}} noch:

1. Ihre beiden Steuer-Identifikationsnummern. Bei mir sind hinterlegt: {{g|de-steuerid|38472619355}} (Frau {{p|emine|ascii-fold|Yildirim-Hoffmann}}) und {{g|de-steuerid|71029485378}} (Herr {{p|matthias|last}}). Bitte prüfen.
2. Die Anlage Kind für {{p|elif|first}}, geb. {{d|2014-02-11|11.02.2014}}: Kinderbetreuungskosten und den Nachweis über das Schulgeld der {{o|Rudolf-Steiner-Schule|company-eponym}}.
3. Die Bescheinigung der Pflegekasse für Ihre Mutter, Frau {{p|hannelore}}, sofern Sie den Pflege-Pauschbetrag geltend machen möchten.
4. Die aktuelle Bankverbindung für die Erstattung. Bisher: {{f|de-iban|DE53 1203 0000 1048 2957 36}}.

Bitte senden Sie mir die Unterlagen per Post an {{a|Rothenbaumchaussee 141, 20149 Hamburg}} oder verschlüsselt per E-Mail. Telefonisch erreichen Sie mich unter {{ph|040 23125 147}}, Frau {{p|anneliese|title-last|Vogelsang}} im Sekretariat unter der Durchwahl -148.

Mit freundlichen Grüßen
Dr. {{p|friedhelm|initial-last|F. Oberländer}}
Steuerberater

P.S.: Herr {{p|matthias|misspelled|Hofmann-Reuter}}, Ihre Fahrtkostenaufstellung vom {{d|2026-04-02|2.4.}} ist angekommen, danke.

Meine Notizen: Die Nummern stimmen. {{p|matthias|nickname}} meint, wir sollen einfach alles per Mail schicken. Oma {{p|hannelore|nickname}} hat Pflegegrad 2 seit {{d|2025-03-01|1. März 2025}}.`),
		ast(`Zu Punkt 2: Herr Dr. {{p|friedhelm|title-last|Oberländer}} möchte für {{p|elif|first}} zwei Dinge, die Betreuungskosten (zum Beispiel Hort oder Tagesmutter, mit Rechnung und Überweisungsbeleg) und eine Bescheinigung der Schule über das gezahlte Schulgeld, weil 30 Prozent davon als Sonderausgaben abziehbar sind. Zu den Nummern: Die Steuer-IDs kennt er bereits, Sie müssen sie nur bestätigen, nicht noch einmal schicken. Die IBAN und die Bescheinigung für Frau {{p|hannelore|title-last|Reuter}} würde ich nicht unverschlüsselt mailen. Soll ich die Antwort entwerfen?`),
		usr(`Ja bitte. Schreib, dass beide Nummern stimmen, dass die Unterlagen per Post kommen und dass {{p|matthias|first}} sie am {{d|2026-05-20|20. Mai}} persönlich im Büro abgibt. Und bitte korrigiere höflich, dass er {{p|matthias|last}} heißt und nicht {{p|matthias|misspelled|Hofmann-Reuter}}. Unterschrift: {{p|emine}}.`)).on("2026-05-29", "friedhelm", "2025-03-01", "hannelore"),

	intl("de-ch-versicherung", "german,switzerland,form",
		usr(`Ich fülle das Unfallformular für meinen Vater aus (er lebt im Wallis). Bitte prüfe, ob die Angaben vollständig sind, und formuliere den Unfallhergang in drei sachlichen Sätzen. Schweizer Schreibweise bitte, also kein ß.

UNFALLMELDUNG UVG
Arbeitgeber: {{o|Zurbriggen Bergbahnen AG|company-eponym}}, {{a|Bahnhofstrasse 7, 3920 Zermatt}}
Kontaktperson: {{p|gesine}}, HR, {{ph|027 555 01 44}}, {{e|g.pfannenstiel@zurbriggen-bahnen.example}}

Verunfallte Person
Name, Vorname: {{p|urs|last-first|Zurbriggen, Urs}}
Geburtsdatum: {{d|1965-08-01|1. August 1965}}
AHV-Nr.: {{g|ch-ahv|756.3058.6174.20}}
Adresse: {{a|Haus Alpenrose, Riedweg 22, 3920 Zermatt}}
Telefon: {{ph|079 555 01 87}}
Zivilstand: verwitwet
Bankverbindung: {{f|ch-iban|CH56 0483 5012 3456 7800 9}}, lautend auf {{p|urs|initial-last|U. Zurbriggen}}

Unfall
Datum, Zeit: {{d|2026-01-17|Samstag, 17. Januar 2026}}, ca. 07.40 Uhr
Ort: Talstation, Perron 2
Zeugen: {{p|christian}} (Kollege), {{p|+Reto Imboden|full}} (Gast, Tel. {{ph|+41 79 555 01 12}})
Erstbehandlung: Dr. med. {{p|+Madlaina Caduff|title-last|Caduff}}, Spital {{l|Visp}}

Was mein Vater am Telefon gesagt hat: "Es het Iis gha uf em Perron, i bi usgrutscht und uf d Schulter gheit. Dr {{p|christian|nickname}} het mi ufglüpft." Er ist seit dem {{d|2026-01-19|19.1.}} krankgeschrieben, voraussichtlich bis {{d|2026-03-02|2. März}}.`)).on("2026-01-17", "urs"),

	intl("pt-br-cadastro", "portuguese,brazil,chat",
		usr(`Oi! Preciso preencher o cadastro do plano de saúde da empresa pra mim e pros meus dependentes e o RH mandou um monte de campo. Me ajuda a conferir se tá tudo certo antes de eu enviar? Segue:

Titular: {{p|thais}}
CPF: {{g|br-cpf|381.527.946-19}}    RG: {{g|br-rg|28.417.390-5}} SSP/RJ
Nascimento: {{d|1990-06-18|18/06/1990}}
Nome da mãe: {{p|+Regina Albuquerque de Souza|full}}
Endereço: {{a|Rua Barão de Mesquita, 482, apto 302, Tijuca, Rio de Janeiro - RJ, 20540-006}}
Celular: {{ph|(21) 95550-0142}}   E-mail: {{e|thais.albuquerque@correiorapido.example}}

Dependente 1 (cônjuge): {{p|joaopedro}}
CPF: {{g|br-cpf|704.193.258-32}}   Nascimento: {{d|1988-12-01|01/12/1988}}
Celular: {{ph|+55 21 95550-0177}}

Dependente 2 (avó, sob minha guarda): {{p|josefa}}
CPF: {{g|br-cpf|216.840.375-90}}   Nascimento: {{d|1938-03-27|27/03/1938}}
Cartão do SUS: {{g|br-cns|898 0041 7733 0015}}

Reembolso via Pix: chave CPF do titular, ou a chave aleatória {{f|br-pix|7c9e6679-7425-40de-944b-e07fc1f90ae7}}. Banco {{o|Itaú|company}}, agência 0482, conta {{f|br-conta|29173-5}}.

A moça do RH, a {{p|beatriz|nickname}} ({{p|beatriz}}, ramal 4417), disse que o prazo é {{d|2026-07-31|31 de julho}} e que o {{p|joaopedro|nickname}} precisa assinar também. Ele tá em {{l|Recife}} até dia 20. Dá pra ele assinar pelo gov.br?`)).on("2026-07-31", "thais"),
}

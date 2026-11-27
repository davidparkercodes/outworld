package main

var itemsIntlC = []def{
	intl("zh-tenant-form", "chinese,non-latin,form",
		usr(`帮我把下面的信息整理成表格，发给物业。不要改动任何号码。

租客姓名：{{p|liuyang|native-script|上官玉兰}}（英文名 {{p|liuyang|western-order|Yulan Shangguan}}）
身份证号：{{g|cn-rid|330106198910214523}}
出生日期：{{d|1989-10-21|1989年10月21日}}
手机：{{ph|138 5550 0147}}
邮箱：{{e|yulan.shangguan@hangmail.example}}
现住址：{{a|浙江省杭州市西湖区文三路478号2幢1104室}}
紧急联系人：{{p|zhangwei|native-script|欧阳志明}}（丈夫），电话：{{ph|+86 139 5550 0162}}，身份证号：{{g|cn-rid|510104197603083718}}
银行卡：{{o|招商银行}}，卡号：{{f|card-unionpay|6214 8301 4829 1734}}
入住日期：{{d|2026-09-01|2026年9月1日}}
房东：{{p|+王秀英|full}}，电话：{{ph|0571-5550-0138}}`)),

	intl("ja-address-change", "japanese,non-latin,form",
		usr(`以下の内容で、会社の総務あてに転居届のメールを書いてください。敬語でお願いします。

氏名：{{p|haruto|native-script|藤沢 陽翔}}（{{p|haruto|native-script|フジサワ ハルト}}）
生年月日：{{d|1991-07-07|1991年7月7日}}（{{d|1991-07-07|平成3年7月7日}}）
マイナンバー：{{g|jp-mynumber|4830 2957 1644}}
旧住所：{{a|〒530-0001 大阪府大阪市北区梅田1-2-3 グランメゾン梅田 804号}}
新住所：{{a|〒150-0002 東京都渋谷区渋谷4-5-6 コーポ青山 201号室}}
電話：{{ph|090-5550-0143}}
メール：{{e|haruto.fujisawa@sakuramail.example}}
転居日：{{d|2026-04-01|令和8年4月1日}}
勤務先：{{o|株式会社ルーメンワーフ・ジャパン}}（上司：{{p|takumi|native-script|中島 拓海}}、内線 4417）
緊急連絡先：{{p|kazuko|native-script|南出 和子}}（母）、{{ph|06-5550-0167}}
給与振込口座：{{o|みずほ銀行}} 渋谷支店 普通 {{f|jp-account|4829173}}`)).on("2026-04-01", "haruto"),

	intl("ko-move-in-report", "korean,non-latin,form",
		usr(`아래 정보로 전입신고서 초안을 써 주세요. 번호는 그대로 두세요.

성명: {{p|donghyun|native-script|백동현}} (영문 {{p|donghyun|western-order|Donghyun Baek}})
주민등록번호: {{g|kr-rrn|881203-1458272}}
연락처: {{ph|010-5550-0147}}
이메일: {{e|donghyun.baek@hanmail.example}}
이전 주소: {{a|부산광역시 해운대구 센텀중앙로 97, 1204호}}
새 주소: {{a|서울특별시 마포구 월드컵북로 396, 302동 1504호}}
전입일: {{d|2026-03-02|2026년 3월 2일}}
세대주: {{p|jiyeon|native-script|박지연}} (배우자), 주민등록번호 {{g|kr-rrn|930725-2264932}}
환급 계좌: {{o|국민은행}} {{f|kr-account|482917-04-305671}}`)),

	intl("ru-name-change", "russian,non-latin,patronymic",
		usr(`Здравствуйте! Помогите, пожалуйста, составить заявление на имя директора о смене фамилии в кадровых документах. Данные:

Сотрудник: {{p|ekaterina|native-script|Волконская Екатерина Сергеевна}} (до брака {{p|ekaterina|maiden|Журавлёва}})
Дата рождения: {{d|1989-09-12|12 сентября 1989 г.}}
Паспорт: {{g|ru-passport|45 19 482917}}, выдан {{d|2019-10-03|03.10.2019}}
ИНН: {{g|ru-inn|165548291740}}
СНИЛС: {{g|ru-snils|148-392-605 90}}
Адрес регистрации: {{a|г. Казань, ул. Баумана, д. 44, кв. 17}}
Телефон: {{ph|+7 (843) 555-01-39}}, моб. {{ph|8 917 555-01-62}}
Эл. почта: {{e|e.volkonskaya@kazanpochta.example}}
Свидетельство о браке от {{d|2026-02-14|14 февраля 2026 года}}, супруг: {{p|arseny|native-script|Волконский Арсений Тимурович}}
Директор: {{p|timur|native-script|Ахметзянов Тимур Ринатович}}. Обращаться к нему нужно по имени-отчеству: {{p|timur|patronymic|Тимур Ринатович}}.
Бухгалтер {{p|irina|patronymic|Ирина Сергеевна}} ({{p|irina|native-script|Журавлёва И. С.}}) просит указать зарплатный счёт: {{f|ru-account|40817 810 4 0048 2917305}}.

Спасибо! {{p|ekaterina|native-script|Катя}}`)).on("2026-02-14", "ekaterina"),

	intl("ar-bank-update", "arabic,non-latin,form",
		usr(`السلام عليكم، أريد صياغة رسالة رسمية للبنك لتحديث بياناتي. هذه هي المعلومات:

الاسم: {{p|layla|native-script|ليلى يوسف المنصوري}}
رقم الهوية الإماراتية: {{g|ae-eid|784-1990-3058172-4}}
تاريخ الميلاد: {{d|1990-11-05|5 نوفمبر 1990}}
رقم الهاتف: {{ph|+971 55 555 0127}}
البريد الإلكتروني: {{e|layla.almansouri@gulfmail.example}}
العنوان: {{a|شقة 1408، برج النخيل، شارع الكورنيش، أبوظبي}}
رقم الآيبان: {{f|ae-iban|AE22 0350 0000 0482 9173 056}}
ولي الأمر: {{p|yusuf|native-script|يوسف المنصوري}} ({{p|yusuf|teknonym|أبو خالد}})، هاتفه: {{ph|050 555 0139}}`)),

	intl("el-he-hi-snippets", "greek,hebrew,hindi,non-latin",
		usr(`Three short notes from three relatives, each needs a polite two-line reply in the same language. Keep every number as written.

1.
Καλησπέρα, στέλνω τα στοιχεία μου για τον λογιστή: {{p|stavros|native-script|Σταύρος Παπαδημητρακόπουλος}}, ΑΦΜ {{g|gr-afm|048291739}}, ΑΜΚΑ {{g|gr-amka|27026901341}}, ημ. γέννησης {{d|1969-02-27|27 Φεβρουαρίου 1969}}, διεύθυνση {{a|Λεωφόρος Συγγρού 218, 17672 Καλλιθέα}}, κινητό {{ph|694 555 0136}}, IBAN {{f|gr-iban|GR28 0172 0480 0054 8291 7305 612}}.

2.
שלום, אני צריכה לעדכן פרטים בקופת החולים. שם: {{p|noa|native-script|נועה בן-שושן}}, ת.ז. {{g|il-id|034859272}}, תאריך לידה: {{d|1995-03-08|8 במרץ 1995}}, טלפון {{ph|052-555-0143}}, כתובת: {{a|רחוב דיזנגוף 214, דירה 9, תל אביב}}, מייל {{e|noa.bs@telavivmail.example}}.

3.
नमस्ते, मुझे बैंक के लिए एक पत्र लिखना है। नाम: {{p|sunita|native-script|सुनीता रघुनाथन}}, आधार: {{g|in-aadhaar|5837 2946 1080}}, जन्म तिथि: {{d|1961-04-09|9 अप्रैल 1961}}, मोबाइल: {{ph|98555 01432}}, पता: {{a|फ्लैट 4बी, कावेरी अपार्टमेंट्स, अड्यार, चेन्नई 600020}}। बेटे का नाम {{p|karthik|native-script|कार्तिक}} है।`)),

	intl("in-kyc-hinglish", "india,hinglish,kyc",
		usr(`Bhai mutual fund KYC ka form reject ho gaya, "name mismatch" bol raha hai. Dekh ke bata kya galat hai? Details niche hain, PAN aur Aadhaar dono ke hisaab se.

As per PAN card: {{p|arjun|caps|ARJUN THEKKEKARA}}, father's name {{p|+Mathew Thekkekara|caps|MATHEW THEKKEKARA}}, DOB {{d|1993-01-26|26/01/1993}}, PAN {{g|in-pan|CQTPT4829K}}
As per Aadhaar: {{p|arjun|initial-first|Arjun T}}, S/O {{p|+Mathew Thekkekara|first|Mathew}}, DOB {{d|1993-01-26|26-01-1993}}, Aadhaar {{g|in-aadhaar|3609 4721 8558}}
Aadhaar address: {{a|Thekkekara House, TC 14/482, Pattom Palace PO, Thiruvananthapuram 695004}}
Current address: {{a|#417, 3rd Cross, 8th Main, Koramangala 4th Block, Bengaluru 560034}}

Bank: {{o|HDFC Bank|company}}, a/c {{f|in-account|50100482917305}}, IFSC HDFC0000417
UPI: {{f|in-upi|arjun.t93@okhdfcbank}}
Mobile linked: {{ph|+91 98555 01287}}, email {{e|arjun.thekkekara@bengalurumail.example}}
Nominee: {{p|gurpreet}} (wife), DOB {{d|1994-10-26|26 Oct 1994}}, she goes by {{p|gurpreet|nickname}}, Aadhaar {{g|in-aadhaar|7412 9058 3367}}

RM ka naam {{p|rajesh|first}} ma'am hai, {{p|rajesh|initial-first|S. Rajeshwari}}, unka number {{ph|080 5550 0142}}. Unhone bola {{d|2026-02-28|28 Feb}} tak resubmit karna hai warna SIP ka first instalment miss ho jayega. Appa ko bolun kya PAN update karne ko, ya Aadhaar mein surname add karwaun?`)).on("2026-02-28", "arjun"),

	intl("africa-kyc-sheet", "africa,kyc,table",
		usr(`Our compliance tool choked on this onboarding sheet. Tell me which ID formats look malformed for their country, and translate the Swahili note at the bottom.

Agent onboarding, West/East/Southern Africa, batch {{d|2026-03-30|30-Mar-2026}}

Country | Full name | ID type | ID number | Mobile money / bank | Phone | Address
NG | {{p|chidinma}} | NIN | {{g|ng-nin|48291730564}} | BVN {{g|ng-bvn|22148291730}}, {{o|GTBank|company}} NUBAN {{f|ng-nuban|0148291730}} | {{ph|+234 803 555 0179}} | {{a|14 Adeola Odeku Street, Victoria Island, Lagos}}
NG | {{p|oluwaseun}} | Int'l passport | {{g|ng-passport|A04829173}} | {{o|Zenith|company}} NUBAN {{f|ng-nuban|2048291735}} | {{ph|0803 555 0167}} | {{a|Plot 7, Admiralty Way, Lekki Phase 1, Lagos}}
KE | {{p|wanjiru}} | National ID | {{g|ke-id|28417390}} | M-Pesa {{ph|0722 555 018}}, KRA PIN {{g|ke-kra-pin|A004829173Z}} | {{ph|+254 722 555 018}} | {{a|Mwanzi Road, Westlands, Nairobi, P.O. Box 4417-00100}}
KE | {{p|baraka}} | National ID | {{g|ke-id|31750482}} | M-Pesa {{ph|0722 555 014}} | {{ph|+254 722 555 014}} | {{l|Kajiado}} town
ZA | {{p|sibusiso}} | SA ID | {{g|za-id|8506175482080}} | {{o|Capitec|company}} acc {{f|za-account|1482917305}}, branch 470010 | {{ph|+27 82 555 0126}} | {{a|22 Vilakazi Street, Orlando West, Soweto, 1804}}
ZA | {{p|thandiwe}} | SA ID | {{g|za-id|9203040287089}} | {{o|FNB|company}} acc {{f|za-account|62048291730}} | {{ph|082 555 0126}} | {{a|7 Jan Smuts Avenue, Parktown, Johannesburg, 2193}}
GH | {{p|kwabena}} | Ghana Card | {{g|gh-card|GHA-728419305-4}} | MTN MoMo {{ph|024 555 0158}} | {{ph|+233 20 555 0158}} | {{a|House No. 14, Ringway Estates, Osu, Accra}}, GPS {{a|GA-048-2917|digital-address}}
EG | {{p|mostafa}} | National ID | {{g|eg-nid|29001011234567}} | {{o|CIB|company}} {{f|eg-account|100048291730}} | {{ph|+20 10 5550 0171}} | {{a|18 شارع طلعت حرب، وسط البلد، القاهرة}}
MA | {{p|younes}} | CIN | {{g|ma-cin|BK482917}} | RIB {{f|ma-rib|011 780 0000482917305612 74}} | {{ph|+212 661 55 01 43}} | {{a|27, rue Tarik Ibn Ziad, Maârif, Casablanca}}
ET | {{p|tigist}} | Kebele ID | {{g|et-kebele-id|AA-04-482917}} | {{o|CBE|company}} {{f|et-account|1000482917305}} | {{ph|+251 91 555 0152}} | {{l|Bole}} sub-city, woreda 03, {{l|Addis Ababa}}

Habari {{p|wanjiru|nickname}}, ni {{p|baraka|first}} hapa. Kitambulisho changu kimeandikwa "{{p|baraka|name-part|Sankale}}" bila "{{p|baraka|name-part|Ole}}", lakini jina langu kamili ni {{p|baraka}}. Tafadhali rekebisha kabla ya tarehe {{d|2026-04-03|3 Aprili}}. Asante sana.`)),

	intl("au-nz-onboarding", "australia,new-zealand,payroll",
		usr(`New starter paperwork for the {{l|Sydney}} and {{l|Auckland}} hires. Check I have everything payroll needs for each and draft one chase email per person for what is missing.

1. {{p|lachlan}} ("{{p|lachlan|nickname}}"), starts {{d|2026-07-06|6/7/2026}}, {{l|Sydney}}
   TFN: {{g|au-tfn|482 917 367}}
   Super: {{o|AustralianSuper|company}}, member no. {{n|identifier|0048291730}}
   Bank: BSB {{f|au-bsb|062-417}}, account {{f|au-account|1048 2917}}, name {{p|lachlan|initial-last|L J McAllister-Ng}}
   Medicare (for the health cover form): {{g|au-medicare|3847 29104 1}}, ref 2
   Address: {{a|Unit 9, 48 Glebe Point Road, Glebe NSW 2037}}
   Mobile: {{ph|0491 570 110}}   Emergency contact: {{p|minori}} (wife) {{ph|0491 570 156}}
   Driver licence NSW: {{g|au-driver-licence|14829173}}

2. {{p|matilda}}, starts {{d|2026-07-13|13/7/2026}}, {{l|Sydney}}, part-time
   TFN: not provided ("will send, it is in a box somewhere" - {{p|matilda|nickname}})
   Bank: BSB {{f|au-bsb|732-048}}, account {{f|au-account|291 730}}
   Address: {{a|27 Darling Street, Balmain NSW 2041}}
   Mobile: {{ph|0491 570 157}}   DOB: {{d|1999-10-31|31/10/1999}}
   Working with children check: {{n|identifier|WWC0482917E}}

3. {{p|aroha}}, starts {{d|2026-08-03|3 August 2026}}, {{l|Auckland}}
   IRD: {{g|nz-ird|084-271-934}}   Tax code: M
   KiwiSaver: yes, 4%
   Bank: {{f|nz-account|38-9017-0482917-00}}
   Address: moving, currently {{a|3/88 Aro Street, Aro Valley, Wellington 6021}}
   Mobile: {{ph|021 555 0137}}   Iwi affiliation (optional field): {{o|Ngāti Porou|iwi}}

4. {{p|hemi}}, transfer from the UK entity, {{d|2026-08-03|3 August 2026}}, {{l|Auckland}}
   IRD: {{g|nz-ird|63-482-919}}   UK NINO on file: {{g|gb-nino|QQ 41 77 33 C}}
   Bank: not provided
   Passport: {{g|nz-passport|LM482917}}`)).on("2026-07-06", "lachlan", "2026-07-13", "matilda", "2026-08-03", "aroha"),

	intl("asia-hr-sheet", "asia,hr,ids",
		usr(`HR ops asked me to sanity check the regional onboarding tracker before it goes to the payroll vendor. Which rows have an ID that does not fit the stated country, and whose name order did we get wrong?

Row | Legal name (as on ID) | Known as | Country | ID type and number | DOB | Mobile | Bank
1 | {{p|weiling|caps|TAN WEI LING}} | {{p|weiling|first}} | Singapore | NRIC {{g|sg-nric|S8431927I}} | {{d|1984-05-12|12 May 1984}} | {{ph|+65 9555 0142}} | {{o|DBS|company}} {{f|sg-account|048-29173-0}}
2 | {{p|kaho|caps|CHEUNG KA HO}} | {{p|kaho|western-order|Ka-ho Cheung}} | Hong Kong | HKID {{g|hk-hkid|K482917(4)}} | {{d|1979-12-01|1 Dec 1979}} | {{ph|+852 5550 0143}} | {{o|HSBC|company}} {{f|hk-account|048-291730-833}}
3 | {{p|yating|caps|LIN YA-TING}} | {{p|yating|first}} | Taiwan | ID {{g|tw-id|F228471934}} | {{d|1962-08-30|民國51年8月30日}} | {{ph|+886 912 555 014}} | {{o|Cathay United|company}} {{f|tw-account|048-29-173056-4}}
4 | Mr. {{p|somchai|first}} {{p|somchai|last}} | {{p|somchai|honorific|Khun Somchai}}, "{{p|somchai|nickname}}" | Thailand | ID {{g|th-id|3 1024 00581 92 8}} | {{d|1975-04-13|13 เม.ย. 2518}} | {{ph|+66 81 555 0147}} | {{o|Kasikorn|company}} {{f|th-account|048-2-91730-5}}
5 | {{p|nurul|full|NURUL AISYAH BINTI ROSLAN}} | {{p|nurul|nickname}} | Malaysia | MyKad {{g|my-mykad|900817-14-5482}} | {{d|1990-08-17|17/08/1990}} | {{ph|+60 12-555 0148}} | {{o|Maybank|company}} {{f|my-account|5148 2917 3056}}
6 | {{p|zainab}} | | Pakistan | CNIC {{g|pk-cnic|35202-4829173-6}} | {{d|1996-02-29|29-Feb-1996}} | {{ph|+92 300 5550149}} | {{o|HBL|company}} {{f|pk-iban|PK91 HABB 0000 0482 9173 0561}}
7 | {{p|jiyeon|caps|PARK JI-YEON}} | {{p|jiyeon|western-order|Ji-yeon Park}} | South Korea | RRN {{g|kr-rrn|930725-2264932}} | {{d|1993-07-25|1993.07.25}} | {{ph|+82 10-5550-0162}} | {{o|Shinhan|company}} {{f|kr-account|110-482-917305}}
8 | {{p|noa}} | | Israel | Teudat Zehut {{g|il-id|034859272}} | {{d|1995-03-08|08/03/1995}} | {{ph|+972 52-555-0143}} | {{o|Bank Hapoalim|company}} {{f|il-account|12-482-917305}}
9 | {{p|jejomar|last-first|Macaraeg, Jejomar}} | {{p|jejomar|nickname}} | Philippines | UMID {{g|ph-umid|0048-2917305-6}} | {{d|1988-06-12|June 12, 1988}} | {{ph|+63 917 555 0148}} | {{o|BDO|company}} {{f|ph-account|0048 2917 3056}}
10 | {{p|haruto|family-first|Fujisawa Haruto}} | {{p|haruto|honorific|Fujisawa-san}} | Japan | My Number {{g|jp-mynumber|483029571644}} | {{d|1991-07-07|H3.7.7}} | {{ph|+81 90-5550-0143}} | {{o|Mizuho|company}} {{f|jp-account|4829173}}

Notes from {{p|khin|first}}: row 5, "{{p|nurul|name-part|binti Roslan}}" is a patronymic, not a surname, do not put {{p|nurul|name-part|Roslan}} in the family name field. Row 4, Thai colleagues go by given name; never write "Mr. {{p|somchai|last}}".`)),

	intl("ca-bilingual-form", "canada,french,english,bilingual",
		usr(`Service Canada sent my partner's form back. Can you tell from this what they want corrected? It is half in French because she filled it in at her mother's in Québec.

DEMANDE DE PRESTATIONS / APPLICATION FOR BENEFITS

Nom de famille / Surname: {{p|genevieve|last}}
Prénom(s) / Given name(s): {{p|genevieve|first}} {{p|genevieve|name-part|Marie-Ève}}
NAS / SIN: {{g|ca-sin|046 454 286}}
Date de naissance / Date of birth (AAAA-MM-JJ): {{d|1985-12-24|1985-12-24}}
Adresse / Address: {{a|4482, rue Saint-Denis, app. 7, Montréal (Québec) H2J 2L1}}
Téléphone: {{ph|514-555-0162}}
Courriel: {{e|gen.tremblay@courrielquebec.example}}
Carte d'assurance maladie (RAMQ): {{g|ca-ramq|TREG 8562 2412}}
Institution financière: {{o|Desjardins|company}}, transit {{f|ca-transit|81530}}, institution 815, compte {{f|ca-account|0482917}}

Conjoint / Spouse: {{p|wade}}, SIN {{g|ca-sin|812 493 070}}, born {{d|1983-03-03|March 3, 1983}}, cell {{ph|(438) 555-0127}}
Employeur / Employer: {{o|Centre hospitalier de l'Université de Montréal}}, dernier jour de travail {{d|2026-05-29|29 mai 2026}}

RAISON DU RETOUR / REASON FOR RETURN ({{d|2026-06-18|2026-06-18}}):
"Le nom sur la demande ({{p|genevieve|alias|Geneviève Tremblay}}) ne correspond pas au nom au dossier du NAS ({{p|genevieve|caps|TREMBLAY-OUELLET, GENEVIEVE M E}}). Veuillez fournir une preuve d'identité. / The name on the application does not match the SIN record."
Agent: {{p|thibault|initial-last|T. de La Rochefoucault-Mbaye}}, bureau 4417, {{ph|1-800-555-0142}}`)).on("2026-06-18", "genevieve"),
}

package main

import (
	"fmt"
	"strings"
)

var inventoryRooms = []string{"Entrance hall", "Living room", "Kitchen", "Bedroom one", "Bedroom two", "Bathroom", "Balcony", "Storage cupboard"}

var inventoryThings = []string{"ceiling light with shade", "pendant fitting", "double radiator with thermostatic valve", "smoke alarm, tested and working", "curtain pole with two curtains", "roller blind", "fitted carpet", "laminate flooring", "double socket", "wall mirror", "set of three shelves", "door with brass handle and two keys"}

var inventoryStates = []string{"good condition, light wear", "new at start of tenancy", "fair condition, small mark noted in photograph", "good condition, professionally cleaned", "working, minor scuff on the left side", "good condition, no defects seen"}

func inventory() string {
	var sb strings.Builder
	n := 0
	for ri, room := range inventoryRooms {
		sb.WriteString(room + "\n")
		for k := 0; k < 9; k++ {
			n++
			sb.WriteString(fmt.Sprintf("  %d.%d  One %s: %s (photo %03d)\n", ri+1, k+1, inventoryThings[(ri*5+k*7)%len(inventoryThings)], inventoryStates[(ri+k*5)%len(inventoryStates)], 100+n*3))
		}
	}
	return sb.String()
}

func longWritten() []def {
	return []def{
		long("long-email-chain", "long,email,negotiation",
			usr(`I am taking over this account on Monday and have been forwarded the whole thread. Read it bottom to top and give me: the agreed terms, the open points, every promise we made with a date, and who on their side decides what.

From: {{p|jordan}} <{{e|jordan.mwangi-hale@corvanefreight.example}}>
To: {{p|dale}} <{{e|d.ferreira-stone@corvanefreight.example}}>
Date: {{d|2026-04-10|Fri, 10 Apr 2026 17:52:03 +0100}}
Subject: Fwd: Reefer lane agreement, {{o|Aegean Reefer Lines}} (everything you need)

{{p|dale|first}},

Welcome to the fun. Whole thread below, newest first. Three things that are not in writing anywhere:

1. {{p|stavros}} is the decision maker on volume and price. His lawyer, {{p|bertrand}} in {{l|Marseille}}, decides everything about liability and will not be hurried. Write to him as Maître {{p|bertrand|title-last|Vasseur}}, in French if you can, and never call his mobile.
2. Operations on their side is {{p|ludovica}}. She is the one who actually knows how many pallets will show up. If {{p|stavros|first}} says eighteen slots and {{p|ludovica|nickname}} says fourteen, believe {{p|ludovica|nickname}}.
3. Their finance contact changed last month. It used to be {{p|gulsen}}; that was my mistake, she is at the Turkish client and I mixed up two threads. The right person is {{p|beatriz}} in their {{l|Lisbon}} shared service centre, {{e|b.pestana@aegeanreefer.example}}, direct line {{ph|+351 21 555 0164}}.

I am on leave from {{d|2026-04-13|Monday 13 April}} and back on {{d|2026-04-27|the 27th of April}}. {{p|ngozi}} has the pen until then and {{p|marguerite|nickname}} is the escalation. My mobile if the building is on fire: {{ph|07700 900271}}.

{{p|jordan|first}}

-----Original Message-----
From: {{p|bertrand}} <{{e|b.vasseur@vasseur-avocats.example}}>
To: {{p|ngozi}} <{{e|ngozi.aderibigbe@corvanefreight.example}}>
Cc: {{p|stavros}} <{{e|s.papadimitrakopoulos@aegeanreefer.example}}>; {{p|jordan}} <{{e|jordan.mwangi-hale@corvanefreight.example}}>; {{p|oyelaran}} <{{e|m.oyelaran-whitcombe@brekkeoyelaran.example}}>
Date: {{d|2026-04-09|jeu. 9 avr. 2026 à 18:31}}
Subject: RE: Draft agreement v3, clause 14 (liability) and clause 9 (temperature excursions)

Chère Madame {{p|ngozi|title-last|Aderibigbe}},

I acknowledge receipt of version 3. My client accepts clauses 1 to 8 and 10 to 13 as drafted. Two points remain.

Clause 9. A temperature excursion of more than two degrees for more than thirty minutes must be treated as a total loss of the affected pallets unless my client's surveyor certifies otherwise within five working days. Your draft says ten. We will not accept ten.

Clause 14. A cap of twice the annual fees is not adequate for pharmaceutical cargo. My client requires either a cap of EUR 2,000,000 per event, or evidence of cargo insurance in that amount naming {{o|Aegean Reefer Lines S.A.}} as co-insured. Maître {{p|oyelaran|title-last|Oyelaran-Whitcombe}}, whom I copy, proposed the second option by telephone on {{d|2026-04-07|7 April}}; I would be grateful for her confirmation in writing.

My client's signatory will be Mr {{p|stavros}}, managing director, holder of Greek identity card {{g|gr-id-card|AK 482917}}. Please insert these details in the signature block. The registered office is {{a|Akti Miaouli 85, 185 38 Piraeus}}.

I am absent from {{d|2026-04-17|17 April}} to {{d|2026-04-24|24 April}} inclusive. In my absence my associate, Maître {{p|solene}}, has authority to agree drafting changes but not commercial terms ({{e|s.kerbrat@vasseur-avocats.example}}, {{ph|+33 4 99 00 41 27}}).

Veuillez agréer, chère Madame, l'expression de mes salutations distinguées.

{{p|bertrand}}
Avocat au Barreau de {{l|Marseille}}
{{o|Vasseur & Associés|company-eponym}}, {{a|41 rue Paradis, 13006 Marseille}}

-----Original Message-----
From: {{p|ngozi}} <{{e|ngozi.aderibigbe@corvanefreight.example}}>
To: {{p|bertrand}} <{{e|b.vasseur@vasseur-avocats.example}}>
Cc: {{p|stavros}}; {{p|jordan}}; {{p|oyelaran}}
Date: {{d|2026-04-03|Fri, 3 Apr 2026 16:20:44 +0100}}
Subject: Draft agreement v3

Dear Maître {{p|bertrand|title-last|Vasseur}},

As promised on our call, please find attached version 3 of the lane agreement, with the changes agreed between Mr {{p|stavros|title-last|Papadimitrakopoulos}} and {{p|jordan}} on {{d|2026-03-24|24 March}} now reflected:

- Eighteen reefer slots per week {{l|Piraeus}} to {{l|Southampton}} from {{d|2026-06-01|1 June 2026}}, six-month minimum term, rate unchanged.
- A named weekend contact on our side. That is {{p|tevita}}, depot supervisor, mobile {{ph|07700 900388}}, with {{p|hemi}} as deputy.
- Invoices monthly in arrears to your client's {{l|Lisbon}} office, payable within 30 days to our account, IBAN {{f|gb-iban|GB09 LOYD 3096 1745 8203 61}}.

I attach the rate table for all lanes for reference; only the first applies to this agreement.

I apologise that this reaches you three days later than I said. The delay was mine.

Kind regards,
{{p|ngozi}}
Commercial Manager, {{o|Corvane Freight Ltd}}
{{ph|0117 496 0144}}

[attachment pasted inline: rates_2026.txt]
`+rateTable(66)+`
-----Original Message-----
From: {{p|stavros}} <{{e|s.papadimitrakopoulos@aegeanreefer.example}}>
To: {{p|jordan}} <{{e|jordan.mwangi-hale@corvanefreight.example}}>
Cc: {{p|ludovica}} <{{e|l.sanfilippo@aegeanreefer.example}}>
Date: {{d|2026-03-25|Τετ, 25 Μαρ 2026 09:14}}
Subject: After our call

Dear {{p|jordan|first}},

Thank you for yesterday. To confirm what we said, because my English on the telephone is worse than my English on paper:

Eighteen slots, six months, same price. I want the weekend contact by name. I want the contract to go to Maître {{p|bertrand|title-last|Vasseur}} and not to me, I am not a lawyer and I do not wish to become one.

One thing I forgot. From {{d|2026-07-01|1 July}} we will ship insulin for a hospital group in {{l|Manchester}}. This cargo is the reason for the eighteen slots. If a pallet of this is lost it is not a commercial problem, it is a problem for patients. So the temperature clause matters to me personally more than the price. My daughter {{p|+Eleni|first}} is a pharmacist and she has explained to me what happens to insulin at twelve degrees.

{{p|ludovica|title-last|Sanfilippo}} will send you the weekly forecast every Thursday. Please do not call her before ten, she starts late and finishes late.

With respect,
{{p|stavros|initial-last|S. Papadimitrakopoulos}}
{{ph|+30 694 555 0136}}

-----Original Message-----
From: {{p|ludovica}} <{{e|l.sanfilippo@aegeanreefer.example}}>
To: {{p|jordan}}; {{p|ngozi}}
Date: {{d|2026-03-19|gio 19 mar 2026, 22:47}}
Subject: Forecast weeks 23 to 30

Ciao {{p|jordan|first}}, ciao {{p|ngozi|first}},

Forecast attached. Honest numbers: fourteen pallets a week in June, sixteen from the first week of July when the hospital contract starts, eighteen only if the {{l|Manchester}} tender is extended, which we will know on {{d|2026-05-15|15 May}}. {{p|stavros|first}} likes round numbers. I like true ones.

Please send pickup confirmations to me and to the gate office ({{e|gate.piraeus@aegeanreefer.example}}), not to {{p|stavros|first}}, he does not read them.

{{p|ludovica|nickname}}
Operations, {{o|Aegean Reefer Lines}}
{{ph|+30 21 0555 0188}} (after 10:00)

-----Original Message-----
From: {{p|jordan}} <{{e|jordan.mwangi-hale@corvanefreight.example}}>
To: {{p|stavros}} <{{e|s.papadimitrakopoulos@aegeanreefer.example}}>
Date: {{d|2026-03-12|Thu, 12 Mar 2026 11:03:19 +0000}}
Subject: Reefer capacity from June

Dear Mr {{p|stavros|title-last|Papadimitrakopoulos}},

{{p|chad}} tells me you spoke at the {{l|Munich}} fair and that you may need more reefer capacity into the UK this summer. I look after the lane. We currently give you twelve slots a week; I can go to eighteen if we can agree a minimum term, and I would like to understand the cargo before I promise temperature performance.

Could we speak on {{d|2026-03-24|Tuesday 24 March}} at 15:00 UK time? I will have {{p|ngozi}} with me.

Also, our system has you as "{{x|wrong-name|Stephen}}". I am told that is wrong and I am sorry. Would you tell me how you prefer to be addressed?

Yours sincerely,
{{p|jordan}}
Head of Sales, EMEA
{{o|Corvane Freight Ltd}}, {{a|Unit 14, Tidewater Business Park, 220 Quayside Road, Bristol BS1 9ZZ}}`)).on("2026-06-01", "stavros", "2026-04-07", "oyelaran", "2026-05-15", "ludovica", "2026-03-24", "jordan"),

		long("long-tenancy-agreement", "long,legal,contract",
			usr(`Our landlord sent this tenancy agreement for signature. Go through it clause by clause and flag anything unusual, one-sided or missing, in plain English. We are first-time renters in England. The inventory at the end is long, you can skim it, but tell me if anything in it could be used against the deposit.

ASSURED SHORTHOLD TENANCY AGREEMENT

THIS AGREEMENT is made on {{d|2023-08-04|4 August 2023}}

BETWEEN

(1) {{p|constantin|caps}} of {{a|The Old Forge, 3 Chew Lane, Chew Magna, Bristol BS40 9ZZ}} ("the Landlord"); and

(2) {{p|rupert|caps|RUPERT CLARKE-ASHBY}} and {{p|folasade|caps|FOLASADE ADEYEMI-CLARKE}}, both currently of {{a|Flat 9, 61 Gloucester Road, Bristol BS7 9ZZ}} (together "the Tenant").

The Landlord's agent for the service of notices is {{o|Brekke Oyelaran LLP|company-eponym}}, {{a|3rd Floor, 19 Corn Street, Bristol BS1 9ZZ}}, marked for the attention of {{p|oyelaran}}.

1. THE PROPERTY
1.1 The Landlord lets and the Tenant takes the property known as {{a|Flat 2, 114 Cotham Hill, Bristol BS6 9ZZ}} ("the Property") together with the fixtures, furniture and effects listed in the Inventory at Schedule 2.
1.2 The Property does not include the garden shed, the loft or the parking space, which the Landlord keeps for his own use.

2. TERM
2.1 The tenancy is for a fixed term of twelve months beginning on {{d|2023-08-12|12 August 2023}} and ending on {{d|2024-08-11|11 August 2024}}.
2.2 If the Tenant stays in the Property after the fixed term with the Landlord's consent, the tenancy continues as a periodic tenancy from month to month on the same terms until ended by notice.

3. RENT
3.1 The rent is £1,430.00 per calendar month, payable in advance on the first day of each month by standing order to the Landlord's account: sort code {{f|gb-sort-code|30-96-17}}, account number {{f|gb-account|45820361}}, account name {{p|constantin|initial-last|C. Dragomir}}.
3.2 The first payment, covering the period from {{d|2023-08-12|12 August}} to {{d|2023-08-31|31 August 2023}}, is £922.58 and is due on signature.
3.3 The Landlord may increase the rent once in any period of twelve months by giving not less than two months' notice in writing. The increase will not exceed the greater of five per cent and the annual change in the Consumer Prices Index.
3.4 Interest is payable on rent more than fourteen days late at three per cent above the Bank of England base rate.

4. DEPOSIT
4.1 The Tenant pays a deposit of £1,650.00 on signature. The Landlord will protect it with the {{o|Deposit Protection Service}} within thirty days and give the Tenant the prescribed information.
4.2 The deposit is held as security for unpaid rent, damage beyond fair wear and tear, missing items, and the reasonable cost of cleaning if the Property is not returned in the condition recorded in the Inventory.
4.3 No interest is payable to the Tenant on the deposit.
4.4 The Tenant must not treat the deposit as payment of the last month's rent.

5. THE TENANT'S OBLIGATIONS
The Tenant agrees:
5.1 to pay the rent on time and to pay council tax, water, gas, electricity, broadband and the television licence for the Property during the tenancy;
5.2 to use the Property as a private home for the Tenant and for the Tenant's child, {{p|temilola}}, and for no other person without the Landlord's written consent;
5.3 not to sublet, take in a lodger, or advertise the Property on any short-stay platform;
5.4 to keep the inside of the Property clean and in good order, to replace light bulbs and smoke alarm batteries, and to keep drains and wastes clear;
5.5 not to make alterations, drill into tiles, change the locks or redecorate without consent. The Landlord agrees that pictures may be hung using the existing hooks;
5.6 not to keep any animal other than one cat, for which consent is given by this clause on condition that the carpets are professionally cleaned at the end of the tenancy;
5.7 not to smoke or vape inside the Property;
5.8 not to cause nuisance to the neighbours, in particular the occupier of Flat 1, Mrs {{p|matilda|title-last|Kowalewski-Brown}}, who works nights;
5.9 to allow the Landlord or his agent to enter at reasonable times on at least twenty-four hours' written notice to inspect or repair, and without notice in an emergency;
5.10 to report any disrepair promptly in writing to the Landlord at {{e|constantin@dragomir-lettings.example}} or by text to {{ph|07700 900455}};
5.11 to return all keys on the last day of the tenancy. The Tenant has been given two sets, numbered {{n|identifier|CH114-2A}} and {{n|identifier|CH114-2B}};
5.12 to pay the Landlord's reasonable costs of replacing lost keys and of any call-out caused by the Tenant's fault.

6. THE LANDLORD'S OBLIGATIONS
The Landlord agrees:
6.1 to allow the Tenant to occupy the Property without unlawful interruption;
6.2 to keep in repair the structure and exterior, and the installations for water, gas, electricity, sanitation and heating, as required by section 11 of the Landlord and Tenant Act 1985;
6.3 to insure the building. The Tenant is responsible for insuring the Tenant's own belongings;
6.4 to provide a gas safety certificate each year, an electrical installation report every five years and the energy performance certificate (current rating D);
6.5 to refund rent for any period during which the Property is uninhabitable through no fault of the Tenant.

7. ENDING THE TENANCY
7.1 Either party may end the tenancy at the end of the fixed term by giving at least two months' notice in writing.
7.2 After the first six months the Tenant may end the tenancy early by giving two months' notice and paying the Landlord's reasonable re-letting costs, capped at £400.
7.3 The Landlord may seek possession during the fixed term on the grounds set out in Schedule 2 of the Housing Act 1988, including rent arrears of two months or more.
7.4 At the end of the tenancy the Tenant must leave the Property clean and empty, and attend the check-out inspection carried out by the inventory clerk.

8. NOTICES
8.1 Notices to the Tenant may be left at the Property or sent to {{e|rupert.ca@harbourmail.example}} and {{e|sade.ac@harbourmail.example}}.
8.2 Under section 48 of the Landlord and Tenant Act 1987 the Landlord's address for notices in England is that of his agent given above.

9. DATA PROTECTION
9.1 The Landlord will hold the Tenant's names, dates of birth ({{d|1982-03-15|15 March 1982}} and {{d|1984-07-29|29 July 1984}}), contact details, right to rent evidence and referencing results for the purposes of the tenancy and for six years afterwards.
9.2 Right to rent checks were completed on {{d|2023-07-28|28 July 2023}} using British passport {{g|gb-passport|548291730}} (Mr {{p|rupert|title-last|Clarke-Ashby}}) and British passport {{g|gb-passport|517304826}} (Mrs {{p|folasade|title-last|Adeyemi-Clarke}}).
9.3 The Tenant's employer references were given by {{p|halvard}} of {{o|Halden & Pryce|company-eponym}} for Mrs {{p|folasade|title-last|Adeyemi-Clarke}} and by {{p|lucinda}} of {{o|Corvane Freight Ltd}} for Mr {{p|rupert|title-last|Clarke-Ashby}}.

10. SPECIAL CONDITIONS
10.1 The Landlord will replace the kitchen worktop, which is water damaged beside the sink, before {{d|2023-09-30|30 September 2023}}.
10.2 The wardrobe in bedroom two has one door missing at the start of the tenancy. The Tenant is not responsible for it.
10.3 The Tenant may park one bicycle in the entrance hall.

SIGNED by the Landlord: {{p|constantin|initial-last|C. Dragomir}}     Date: {{d|2023-08-04|04/08/2023}}
SIGNED by the Tenant: {{p|rupert|initial-last|R. Clarke-Ashby}}     Date: {{d|2023-08-04|04/08/2023}}
SIGNED by the Tenant: {{p|folasade|initial-last|F. Adeyemi-Clarke}}     Date: {{d|2023-08-04|04/08/2023}}
Witness: {{p|gwenllian}}, {{a|12 Harbour Parade, Portishead, Bristol BS20 9ZZ}}, occupation: general practitioner

SCHEDULE 1: PRESCRIBED INFORMATION
Deposit certificate {{n|identifier|DPS-41772093}}. Lead tenant for deposit purposes: {{p|rupert}}. Contact after the tenancy: {{ph|07700 900261}}.

SCHEDULE 2: INVENTORY AND SCHEDULE OF CONDITION
Prepared by {{p|matilda}}, independent inventory clerk, on {{d|2023-08-11|11 August 2023}}. Meter readings: gas 04417, electricity 28173. Signed as agreed by {{p|matilda|initial-last|M. Kowalewski-Brown}} and {{p|folasade|nickname}} {{p|folasade|alias|Clarke}}.

`+inventory()+`
4.7  Wardrobe, bedroom two: ONE DOOR MISSING at check-in, see clause 10.2.
3.9  Kitchen worktop: water damage beside sink at check-in, see clause 10.1.

Our questions: is clause 3.3 normal? Can he really keep the loft? And {{p|rupert|nickname}} wants to know whether clause 5.6 means the cat is allowed or not, because {{p|temilola|nickname}} has already named it.`)).on("2023-08-12", "rupert", "2023-08-04", "constantin", "2023-08-11", "matilda"),

		long("long-postmortem", "long,incident,postmortem",
			usr(`Edit this postmortem draft for clarity and blamelessness. Several sentences name individuals in a way that reads like blame; rewrite those to describe the system instead, but keep the timeline entries and names as they are because the on-call log is the record. Then list the action items with owners and due dates in a table.

POSTMORTEM: Label service outage, {{d|2025-12-03|3 December 2025}}
Status: draft 2, for review on {{d|2025-12-10|10 December}}
Author: {{p|farzaneh}}. Reviewers: {{p|svenerik}}, {{p|thandiwe}}, {{p|sue}}.
Incident commander: {{p|ibrahim}}. Severity: 1. Duration: 3 hours 12 minutes.

SUMMARY
On the morning of {{d|2025-12-03|Wednesday 3 December}}, the label rendering service stopped producing PDFs for all tenants in the EU region. Depots could not print shipping labels. Three customers moved to manual fallback; one, {{o|Corvane Freight}}, had a written out-of-hours playbook and lost no shipments. The others lost between forty minutes and two hours of dispatch time. The trigger was a certificate rotation that {{p|lerato}} carried out the evening before. The cause was that the label workers cached the old certificate chain and the rotation runbook did not mention restarting them. {{p|lerato|first}} followed the runbook exactly.

IMPACT
- 4,112 label requests failed. 3,960 were retried successfully after recovery; 152 were abandoned by users.
- {{o|Pohjolan Rahti Oy}} missed a ferry cut-off at {{l|Turku}}. Their finance lead, {{p|aino}}, has asked for a service credit.
- {{o|Transportes Illimani}} was unaffected (different region).
- Support received 61 tickets. {{p|leilani}} and {{p|xiadani}} handled all of them; {{p|leilani|nickname}} stayed four hours past her shift.

WHAT HAPPENED
At 17:40 on {{d|2025-12-02|2 December}}, {{p|lerato|handle|lfakude}} rotated the internal CA intermediate as scheduled. The gateway and API picked up the new chain within a minute. The label workers did not: they load the chain once at start-up. They kept working overnight because existing connections to the template store stayed open.

At 05:58 the template store restarted for an unrelated kernel patch. Every label worker then tried to reconnect, presented the old chain, and was refused. Renders began to fail at 06:01.

The alert for render failures fired at 06:03 and paged the primary on-call, {{p|dawit}}. {{p|dawit|first}} was on a flight from {{l|Addis Ababa}} to {{l|Frankfurt}}; the on-call calendar had not been updated after a swap he agreed with {{p|kwabena}} by direct message. The page escalated to the secondary, {{p|thandiwe}}, at 06:18. She acknowledged at 06:19 from {{l|Johannesburg}}.

{{p|thandiwe|nickname}} first suspected the template store, because it had just restarted, and spent twenty-five minutes on it with {{p|clay}}, who owns its database. At 06:47 {{p|clay|first}} noticed TLS handshake errors in the worker logs. At 06:52 {{p|thandiwe|nickname}} paged {{p|svenerik}} as the owner of the certificate tooling. He was reachable only on his personal mobile ({{ph|070-1740 625}}) because his work phone was being repaired; {{p|sue}} had the number.

At 07:05 {{p|ibrahim}} took over as incident commander so that {{p|thandiwe|nickname}} could keep debugging. At 07:20 {{p|svenerik|nickname}} confirmed the workers held the old chain. Restarting them required a deploy approval, which under the change freeze needed a director. {{p|takumi}} approved at 07:41 from {{l|Osaka}}, where it was late afternoon.

The first restarted worker rendered successfully at 07:49. A second problem then appeared: the retry queue held four thousand jobs and the workers were sized for normal load. {{p|bartosz}} raised the worker count from six to twenty at 08:10; {{p|anneliese}} disabled label previews in the web app to shed load. The queue drained at 09:13 and the incident was closed.

CUSTOMER COMMUNICATION
The status page was updated at 06:40 by {{p|rosalind}}, thirty-seven minutes after the first alert. That is too slow. The first customer email went out at 07:15, drafted by {{p|rosalind|nickname}} and approved by {{p|ibrahim|first}}.

{{p|penelope}} at {{o|Corvane Freight}} telephoned support at 06:20, before the status page changed, and was told by the answering service that there were no known issues. She switched her depot to the paper playbook on her own judgement. {{p|tevita}}, their depot supervisor, later sent us a photograph of forty hand-written labels with the note "you owe {{p|penelope|nickname}} a drink". We agree.

WHAT WENT WELL
- The secondary acknowledged within a minute of being paged.
- {{p|clay|possessive}}'s read of the worker logs cut the diagnosis short.
- Nobody tried to roll back the certificate, which would have broken the gateway as well.
- {{p|takumi|nickname}} approved the freeze exception in four minutes.

WHAT WENT BADLY
- The rotation runbook did not list the label workers. It was written by {{p|cormac}} in 2024, before the workers existed, and nobody owned it after he changed teams.
- The on-call swap between {{p|dawit|first}} and {{p|kwabena|nickname}} lived in a direct message. The paging tool did not know.
- The page to {{p|svenerik|first}} went to a phone in a repair shop.
- The answering service had no way to learn about an incident before the status page changed.
- Worker capacity for a retry storm was never tested.

WHERE WE GOT LUCKY
- The template store restart happened at 05:58 and not at 09:00, when volume is five times higher.
- {{p|penelope|possessive}}'s playbook existed. It was her own initiative.

ACTION ITEMS
1. Make label workers reload the certificate chain without a restart. Owner: {{p|bartosz}}. Due {{d|2026-01-16|16 January 2026}}.
2. Add every TLS client to the rotation runbook and make the runbook a checked script. Owner: {{p|lerato}}. Due {{d|2025-12-19|19 December}}.
3. On-call swaps must be made in the paging tool; a bot will reject swaps announced only in chat. Owner: {{p|svenerik}}. Due {{d|2025-12-17|17 December}}.
4. Paging tool to hold a personal fallback number for each engineer, with consent. Owner: {{p|sue}}. Due {{d|2026-01-09|9 January}}.
5. Status page to be updated automatically when a severity 1 alert is acknowledged. Owner: {{p|rosalind}}. Due {{d|2026-01-23|23 January}}.
6. Load test the retry path at five times normal volume. Owner: {{p|anneliese}} with {{p|teodora}}. Due {{d|2026-02-06|6 February}}.
7. Offer {{o|Pohjolan Rahti Oy}} a service credit and a call with engineering. Owner: {{p|leilani}} with {{p|hyunwoo}}. Due {{d|2025-12-12|12 December}}.
8. Ask {{p|penelope|first}} whether we may publish her playbook as a template for other customers. Owner: {{p|rosalind|nickname}}.

APPENDIX A: PAGING AND ALERT LOG (UTC, from the paging tool, unedited)
`+alertTimeline(64, 21600, map[int]string{
				0:  `FIRING [sev1] render failures above 1%, region eu-central, paging primary {{p|dawit}} ({{ph|+251 91 555 0143}})`,
				2:  `NOTE no acknowledgement from {{p|dawit|handle|dawitg}} after 5m, 10m, 15m; calendar shows him on call until {{d|2025-12-04|4 Dec}}`,
				4:  `ESCALATED to secondary {{p|thandiwe}} ({{ph|+27 82 555 0126}}), acknowledged after 58s`,
				9:  `NOTE {{p|thandiwe|handle|thandi-mp}}: "suspect template store, pulling in {{p|clay|lower-first}}"`,
				14: `PAGE to {{p|svenerik}} work phone {{ph|+46 70 174 06 25}} failed (unreachable), retried twice`,
				15: `NOTE {{p|sue|handle|sharrowgate}}: "his work phone is at the repair shop, calling his own mobile"`,
				19: `NOTE incident commander is now {{p|ibrahim|handle|ielsayed}}`,
				27: `NOTE freeze exception approved by {{p|takumi|handle|tnakashima}}, change {{n|identifier|CHG-2025-1203-07}}`,
				31: `NOTE first successful render after restart, worker label-worker-3, build {{n|hash|f2ca8a1b}}`,
				40: `NOTE {{p|bartosz|handle|bwisniewski}} scaled label-worker from 6 to 20 replicas`,
				63: `RESOLVED [sev1] render failures above 1%, region eu-central, closed by {{p|ibrahim}}, postmortem owner {{p|farzaneh}} ({{e|farzaneh.moghaddam-roth@lumenwharf.example}})`,
			}))).on("2025-12-03", "ibrahim", "2026-01-16", "bartosz", "2025-12-19", "lerato", "2025-12-17", "svenerik", "2026-01-09", "sue", "2026-01-23", "rosalind", "2026-02-06", "anneliese", "2025-12-12", "leilani"),
	}
}

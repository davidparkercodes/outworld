package main

const designDoc = `Review this RFC as a skeptical staff engineer. I want the three weakest points and the questions you would ask in the review meeting. Do not rewrite it.

# RFC-0142: At-least-once webhook delivery with idempotent receivers

Status: In review
Author: {{p|priya}} ({{e|priyanka.venkataraman@harborline.example}})
Reviewers: {{p|bjorn|first}}, {{p|seun|nickname}}, {{p|thuy|ascii-first|Thuy}}
Last updated: {{d|2026-03-02|2 March 2026}}

## 1. Summary

Our webhook dispatcher promises that every event is delivered to every subscribed endpoint, but in practice it delivers most events once, some events twice, and a small number of events never. This document proposes a redesign of the dispatcher around a durable outbox, explicit delivery attempts, and an idempotency contract that receivers can rely on. The goal is not exactly-once delivery, which we cannot offer over HTTP to servers we do not control. The goal is at-least-once delivery with enough metadata that a careful receiver can make the effect exactly-once on their side, and a careless receiver is no worse off than today.

The proposal is deliberately boring. It uses the relational database we already operate, the queue we already operate, and a retry schedule that is simple enough to explain in one paragraph of public documentation. Where a cleverer design was available, the section on alternatives explains why it was not chosen.

## 2. Background

The current dispatcher was written in the first year of the product, when there were fewer than a hundred subscriptions and all of them belonged to customers we could call on the phone. It works as follows. When a domain event is committed, a database trigger inserts a row into a notifications table. A pool of workers polls that table, renders a JSON payload for each subscription that matches the event type, and performs an HTTP POST. If the POST returns a 2xx status the row is deleted. If it fails, the worker sleeps and tries again, up to five times, holding the row and a database connection for the entire time.

This design has three structural problems.

First, the delivery state lives in worker memory. If a worker is restarted between a successful POST and the delete, the event is delivered again by another worker with no indication that it is a duplicate. If a worker is restarted during its sleep, the retry count is lost and the event gets a fresh five attempts. During the deploy freeze in December we measured a duplicate rate of roughly four in every thousand deliveries, almost all of them caused by routine restarts.

Second, a slow endpoint holds a worker hostage. The HTTP timeout is thirty seconds and the pool has forty workers, so forty subscriptions that each take thirty seconds to answer can stall delivery for every other customer. This has happened twice. The second time, the stalled endpoints all belonged to a single integration partner whose load balancer was accepting connections and then doing nothing with them.

Third, there is no record of what was attempted. When a customer asks whether we sent an event, the honest answer is that we do not know. Support can see that the row is gone, which means either that it was delivered or that it exhausted its retries, and those two cases are indistinguishable after the fact. The largest account affected so far was {{o|Abara Foods|company-eponym}}, whose integration lead spent two days reconciling orders by hand before we could confirm which notifications had been dropped.

## 3. Goals and non-goals

Goals:

- Every event is attempted for every matching subscription until it is acknowledged or until a documented retry budget is exhausted.
- Every attempt is recorded with its time, its outcome and the response status, and that record is available to the customer.
- A receiver can detect duplicates using a stable identifier that does not change between attempts.
- A slow or failing endpoint cannot delay deliveries to other endpoints by more than a bounded amount.
- A customer can replay any event from the last thirty days on demand.

Non-goals:

- Ordering. Events for the same resource may arrive out of order, as they can today. We will include a monotonically increasing sequence number per resource so that receivers who care can reorder or discard stale updates, but the dispatcher will not hold back a later event because an earlier one is still retrying.
- Exactly-once delivery. See the summary.
- Payload transformation, filtering beyond event type, or fan-out to anything other than HTTPS endpoints. These are reasonable features and they are out of scope here.

## 4. Proposed design

### 4.1 The outbox

Domain services already write events to an events table in the same transaction as the state change. We keep that. What changes is the step after it. Instead of a trigger that fans out to a notifications table, a single relay process reads the events table in commit order and, for each event, inserts one delivery row per matching subscription into a new deliveries table. The relay records its position in a small checkpoint table, so that a restart resumes from the last event whose deliveries were fully written.

A delivery row has a stable identifier, the event identifier, the subscription identifier, a state, an attempt count, the time of the next attempt, and a lease. The stable identifier is derived deterministically from the event identifier and the subscription identifier, so that if the relay crashes after inserting some but not all deliveries for an event and then reprocesses that event, the inserts that already happened are rejected by a unique constraint instead of creating duplicates. This is the property that the current trigger does not have and the main reason the relay is a separate process: we want one place where the mapping from events to deliveries is computed, and we want it to be restartable.

### 4.2 Attempts

Workers no longer hold a delivery for the length of its retry schedule. A worker claims a batch of deliveries whose next attempt time has passed by setting a lease that expires after ninety seconds. It makes one HTTP request per delivery, with a connect timeout of three seconds and a total timeout of ten. It then writes an attempt row with the outcome and updates the delivery: on a 2xx response the state becomes delivered; on a 410 the state becomes cancelled and the subscription is disabled; on anything else the attempt count is incremented and the next attempt time is pushed out according to the schedule below. If the worker dies, the lease expires and another worker claims the delivery. The worst case is one duplicate request, and that duplicate carries the same identifier as the original.

The retry schedule is exponential with jitter, starting at ten seconds and doubling up to a ceiling of one hour, for a total budget of seventy-two hours. In round numbers that is about eighty attempts. After the budget is exhausted the state becomes failed, the subscription owner is notified by email, and the delivery remains available for manual replay.

### 4.3 Isolation between endpoints

Deliveries are claimed per endpoint, not globally. Each endpoint has a concurrency limit, initially four, and a circuit breaker. If the last ten attempts to an endpoint all failed or timed out, the breaker opens and the endpoint is skipped for a cool-down period that grows with consecutive openings. While the breaker is open, deliveries for that endpoint accumulate with their next attempt time unchanged, so that when the endpoint recovers they are sent promptly and in roughly the original order. A single probe request is let through at the end of each cool-down.

This is the part of the design most likely to need tuning. A limit of four concurrent requests is conservative for large customers and generous for small ones. We propose to ship with a fixed default and a per-subscription override that support can set, and to revisit once we have a month of data.

### 4.4 The receiver contract

Every request carries three headers: an event identifier, a delivery identifier, and an attempt number. The delivery identifier is the idempotency key. The public documentation will say, in these words, that a receiver should record the delivery identifier of every request it has processed and should acknowledge without reprocessing any request whose delivery identifier it has seen before. We will publish reference middleware for the four languages that account for most of our integrations.

Payloads are signed as they are today, with a timestamped signature over the raw body. The signing secret is per subscription and can be rotated with an overlap window during which both the old and the new secret are accepted.

### 4.5 Replay and inspection

Because attempts are rows, the customer dashboard can show them. For each delivery the customer sees the event type, the time of each attempt, the response status, the first two kilobytes of the response body, and the next scheduled attempt if there is one. A replay button creates a new delivery with a new delivery identifier that references the same event, so that a receiver following the contract will process it again, which is the point of a replay.

Attempt rows are retained for thirty days and then dropped by partition. Event payloads are retained for the same period. We considered retaining metadata without payloads for longer, and decided against it for now because nobody has asked.

## 5. Data model and capacity

At current volume we create about nine million deliveries a day and, with a first-attempt success rate of ninety-seven percent, about nine and a half million attempts. Thirty days of attempts at roughly two hundred bytes per row is under sixty gigabytes before indexes, which fits comfortably on the existing cluster. The deliveries table is the hot one. Its claim query needs an index on endpoint, state and next attempt time, and that index will churn. We have run a synthetic load at five times current volume for six hours on a replica-sized instance and saw stable latency for the claim query, with autovacuum keeping up after we lowered the scale factor for that table.

The relay is a single writer by design. Its throughput ceiling in the same test was about four thousand events per second, which is forty times our present peak. If we ever need more, the events table can be partitioned by tenant and one relay run per partition, at the cost of losing global commit order, which no consumer depends on.

## 6. Migration

The migration has to be invisible to receivers, with one exception: they will start receiving two new headers, which well-behaved HTTP servers ignore.

Phase one deploys the relay and the new tables in shadow mode. The relay writes deliveries, and workers record what they would have attempted, but no requests are sent from the new path. We compare the shadow deliveries against the old notifications table for a week and investigate every difference.

Phase two moves subscriptions to the new path in cohorts, starting with our own internal consumers, then customers who have opted in to the beta, then everyone else by tenant identifier. A subscription is on exactly one path at a time. The switch is a flag on the subscription row that both the trigger and the relay respect.

Phase three removes the trigger and the old worker pool. We keep the notifications table read-only for thirty days in case a customer asks about a delivery from before their cohort moved.

Rollback during phase two is the same flag in the other direction. Deliveries already created on the new path for a subscription that is rolled back are cancelled, and the events they refer to are re-enqueued on the old path. This can produce duplicates, which the old path could produce anyway.

## 7. Alternatives considered

Use the message queue as the source of truth. We could publish one message per delivery and let the queue handle redelivery. We rejected this because the queue's redelivery semantics are tied to consumer acknowledgements with a maximum visibility timeout of twelve hours, which does not fit a seventy-two hour budget without building a second scheduling layer on top, and because per-message inspection would require copying attempt data into a database anyway.

Buy instead of build. There are hosted services that do this well. The evaluation {{p|seun|first}} ran last quarter found two that met the functional requirements. Both would require sending full event payloads, which include our customers' order data, to a third party, and the data processing terms of our larger contracts do not currently allow that without customer consent. We may revisit this for a future product tier.

Keep the current dispatcher and add an attempts log. This would address the third problem in the background section and neither of the first two.

## 8. Risks

The largest risk is the relay. It is a single process whose failure stops all new deliveries. It will run as two replicas with leader election on a database advisory lock, and its lag is the first alert on the new dashboard. In the shadow phase we will kill it on purpose every day.

The second risk is that the circuit breaker hides a real outage on our side. If a network change in our own environment makes many endpoints unreachable at once, every breaker opens and the system goes quiet instead of loud. We will alert on the fraction of endpoints with an open breaker, not only on the error rate.

The third risk is operational. The on-call rotation will own a system with more moving parts than the one it replaces. A runbook is part of the deliverable, and the first on-call shift after general availability will be shadowed by the author. For the launch week the escalation contact is {{p|priya|nickname}} on {{ph|+91 00000 55017}}.

## 9. Open questions

- Should a 429 response be treated differently from a 503? The draft treats them the same, but honors a Retry-After header up to the one hour ceiling.
- Should replay be rate limited? A customer could replay thirty days of events at once. The per-endpoint concurrency limit bounds the damage, but the dashboard should probably ask for confirmation above some size.
- {{p|bjorn|first}} asked in an earlier draft whether the delivery identifier should be a UUID or a hash. It is currently a hash of the event and subscription identifiers, which makes the relay's deduplication trivial, at the cost of being guessable by someone who knows both. Since both are already visible to the receiver, we think this is acceptable.
- {{p|thuy|ascii-fold|Thuy Nguyen}} raised the question of receivers in regions with strict data residency rules. Nothing in this design changes where requests originate, but the attempts table will store response bodies, and those may contain personal data returned by the receiver. Truncating to two kilobytes reduces the exposure and does not eliminate it. Legal review is scheduled for {{d|2026-03-18|18 March}}.

## 10. Timeline

Shadow mode by the end of the first month, internal cohort two weeks later, general availability by the end of the quarter. These are estimates and depend on what the shadow comparison finds.`

func longItems() []def {
	appLog := "Something leaked customer data into this log between 02:10 and 02:40 and I need to file the incident. Find every line that contains personal data or credentials, quote it, and tell me which component wrote it. Ignore the noise.\n\n" +
		"== shipments-api, pod shipments-api-7c9f, log opened {{d|2026-03-06|2026-03-06}} 02:10 UTC ==\n" +
		logLines(270, 2*3600+600, map[int]string{
			41:  "[http-3] WARN  api.auth - login failed 3 times for {{e|khalid.altamimi@example.net}} from 203.0.113.41, account locked for 15 min",
			97:  "[http-2] ERROR api.labels - carrier rejected label: {\"recipient\":{\"name\":\"{{p|khalid}}\",\"phone\":\"{{ph|+971 50 555 0142}}\",\"street\":\"{{a|Villa 22, Street 14B, Al Safa 1}}\",\"city\":\"{{l|Dubai}}\"},\"reason\":\"postcode missing\"}",
			98:  "[http-2] INFO  api.labels - retry queued for shipment 8841207, attempt 2 of 5",
			156: "[consumer-4] DEBUG carriers.client - refreshing token, old value {{s|bearer-token|car_tok_9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a}} expires in 41 s",
			203: "[http-5] INFO  api.audit - user {{p|zainab|handle|zqureshi}} exported 1 shipment for tenant 4410 on behalf of support case 77120",
			244: "[scheduler-1] WARN  jobs.digest - skipped digest for {{e|k.altamimi@example.net}}: address bounced on {{d|2026-03-01|2026-03-01}}",
		}) + "== end of excerpt =="

	tsHeader := "Split this file into one module per resource and tell me what else you would clean up while you are in there.\n\n" +
		"/**\n * resources.ts\n *\n * Generated from the service template, then edited by hand.\n * @author {{p|jiwoo|western-order|Ji-woo Seo}} <{{e|jiwoo.seo@example.com}}>\n * @since {{d|2025-08-11|2025-08-11}}\n */\n"
	tsFile := tsHeader + crudFile("", []string{"shipment", "label", "carrier", "manifest", "pickup", "rate", "zone", "tariff", "claim", "invoice", "refund", "webhook", "tenant", "depot"}, map[string]string{
		"carrier": "\n  // TODO({{p|kaho|handle|felix.tsang}}): move this to the secret store, it is the sandbox key but still\n  private readonly carrierApiKey = '{{s|api-key|cx_sandbox_3d2e1f0a9b8c7d6e5f4a3b2c1d0c9d8e}}';\n",
		"claim":   "\n  // {{p|meilin|first}} wants claims older than 90 days hidden from this listing, see her note in the tracker.\n",
		"webhook": "\n  // Signature scheme agreed with {{p|priya|initial-last|P. Venkataraman}}; do not change without updating RFC-0142.\n",
	})

	ci := "CI is red on main and I cannot see why in all this output. What actually failed, and is anything in this log something we should not be printing?\n\n" +
		"Run started {{d|2026-03-09|Mon Mar  9}} 06:14:02 UTC 2026 on runner linux-x64-12\n" +
		ciLog(250, 6*3600+14*60, map[int]string{
			0:   "Checking out {{n|hash|5e1c309dae7f45e0f39b1bf3ac3cd9db12e7d689}} (main), last commit by {{p|ngozi}} <{{e|ngozi@harborline.example}}>",
			1:   "Workspace is /home/runner/work/shipments-api, cache restored from key node-{{n|hash|c3ab8ff13720e8ad9047dd39466b3c89}}",
			60:  "[env] NODE_ENV=test CI=true TZ=UTC LOG_LEVEL=warn",
			61:  "[env] DATABASE_URL=postgres://ci:{{s|password|ci-only-Pg-2291}}@127.0.0.1:5432/shipments_test",
			62:  "[env] CARRIER_SANDBOX_KEY={{s|api-key|cx_sandbox_3d2e1f0a9b8c7d6e5f4a3b2c1d0c9d8e}}",
			171: "FAIL tests/labels/labels_case_171.test.ts expected recipient name \"{{p|+Test Recipient|placeholder|Test Recipient}}\" but received \"{{p|khalid}}\", fixture was copied from a production payload",
			172: "    at Object.<anonymous> (/home/runner/work/shipments-api/tests/labels/labels_case_171.test.ts:44:29)",
			249: "Tests: 1 failed, 124 passed, 125 total. Notifying {{e|seun.babatunde@harborline.example}} (on call this week).",
		})

	notes := "Write the upgrade guide section for people going from 3.x to 4.2. Only mention changes that need action from the user.\n\n# Changelog\n" +
		changelog([]string{"4.2.0", "4.1.3", "4.1.2", "4.1.0", "4.0.1", "4.0.0", "3.19.4", "3.19.0", "3.18.2", "3.18.0", "3.17.5"}) +
		"\nBuilt with {{o|Bazel|tool}} and tested against {{o|MariaDB|product}}, {{o|Percona|product}} and {{o|Aurora|product}}. Fuzzing by {{o|Jazzer|tool}}."

	agentFile := crudFile("// rates.ts, owner: {{p|thuy|handle|thuy.nguyen}}\n", []string{"rate", "surcharge", "discount", "quote", "contract", "lane"}, map[string]string{
		"discount": "\n  // {{p|tomasz|nickname}}: discounts above 40 percent need approval, enforced in the API layer not here.\n",
	})
	agentLog := logLines(110, 9*3600, map[int]string{
		37: "[http-1] ERROR api.rates - quote failed for tenant 4410: contract owner {{e|tomasz.kowalczyk-brandt@harborline.example}} has no approval role",
		88: "[http-4] INFO  api.audit - user {{p|tomasz|handle|tkb}} approved discount 55 percent on lane 1402",
	})

	return []def{
		long("design-doc", "long,document", usr(designDoc)),
		long("app-log", "long,logs", usr(appLog)),
		long("generated-source", "long,source,author-comment", usr(tsFile)),
		long("ci-log", "long,logs,secrets", usr(ci)),
		long("changelog", "long,negative,versions,hashes", usr(notes)),
		long("agent-session", "long,tool-calls,agent",
			sys("You are a coding agent working in a TypeScript repository. Use the provided tools. Keep edits minimal."),
			usr("Discount approvals are broken for one of our own staff accounts. Start with src/rates.ts and the last hour of logs. The affected user is {{p|tomasz}}."),
			ast("I will read the file first.\n\n<tool_use name=\"read_file\">{\"path\":\"src/rates.ts\"}</tool_use>"),
			usr("<tool_result name=\"read_file\">\n"+agentFile+"</tool_result>"),
			ast("The service layer does not check roles, as the comment from {{p|tomasz|nickname}} says. I will look at the logs next.\n\n<tool_use name=\"bash\">{\"command\":\"kubectl logs deploy/shipments-api --since=1h | head -110\"}</tool_use>"),
			usr("<tool_result name=\"bash\">\n"+agentLog+"</tool_result>\n\nAlso: the approval role was renamed last week, {{p|thuy|ascii-first|Thuy}} did that migration. Could that be it?"),
		),
	}
}

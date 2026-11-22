package main

import (
	"fmt"
	"strings"
)

func clock(start, step, i int) string {
	t := start + step*i + (i*37)%step
	return fmt.Sprintf("%02d:%02d:%02d.%03d", (t/3600)%24, (t/60)%60, t%60, (i*271)%1000)
}

var logTemplates = []string{
	"[consumer-%d] INFO  queue.consumer - polled %d messages from orders.v2 partition %d in %d ms",
	"[http-%d] INFO  api.access - GET /v2/shipments/%d/events returned 200 in %d ms, cache hit ratio %d percent",
	"[scheduler-%d] DEBUG jobs.rollup - window %d closed with %d late events, watermark advanced by %d ms",
	"[consumer-%d] INFO  ledger.writer - committed batch %d with %d rows to shard %d",
	"[http-%d] WARN  api.ratelimit - tenant %d is at %d percent of its quota, next refill in %d s",
	"[gc-%d] DEBUG runtime.memory - pause %d ms, heap %d MiB after sweep, %d MiB released to the OS",
	"[http-%d] INFO  api.access - POST /v2/labels/%d/print returned 202 in %d ms, payload %d bytes",
	"[replicator-%d] INFO  storage.replica - follower %d acknowledged offset %d with lag %d ms",
	"[scheduler-%d] INFO  jobs.cleanup - removed %d expired idempotency keys older than %d h from bucket %d",
	"[consumer-%d] DEBUG tracing.export - flushed %d spans in %d ms to the collector, queue depth %d",
	"[http-%d] INFO  api.access - GET /v2/rates?zone=%d returned 304 in %d ms, upstream calls %d",
	"[health-%d] DEBUG probes.readiness - dependency check %d passed in %d ms, consecutive successes %d",
}

func logLines(n, start int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		if s, ok := special[i]; ok {
			sb.WriteString(clock(start, 7, i) + " " + s + "\n")
			continue
		}
		t := logTemplates[(i*5+i/7)%len(logTemplates)]
		sb.WriteString(clock(start, 7, i) + " " + fmt.Sprintf(t, 1+i%6, 11+(i*13)%89, 2+(i*7)%40, 3+(i*29)%211) + "\n")
	}
	return sb.String()
}

func title(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

func crudModule(resource string, extra string) string {
	r, R := resource, title(resource)
	return fmt.Sprintf(`
export interface %[2]sRecord {
  id: string;
  tenantId: string;
  status: 'draft' | 'active' | 'archived';
  createdAt: number;
  updatedAt: number;
  attributes: Record<string, unknown>;
}

export class %[2]sService {
  constructor(private readonly store: Store<%[2]sRecord>, private readonly events: EventBus, private readonly clock: Clock) {}
%[3]s
  async list%[2]ss(tenantId: string, cursor?: string, limit = 50): Promise<Page<%[2]sRecord>> {
    if (limit < 1 || limit > 200) {
      throw new ValidationError('limit must be between 1 and 200');
    }
    const page = await this.store.query({ tenantId, cursor, limit, order: 'createdAt' });
    this.events.emit('%[1]s.listed', { tenantId, count: page.items.length });
    return page;
  }

  async get%[2]s(tenantId: string, id: string): Promise<%[2]sRecord> {
    const record = await this.store.get(id);
    if (!record || record.tenantId !== tenantId) {
      throw new NotFoundError('%[1]s ' + id + ' was not found for this tenant');
    }
    return record;
  }

  async create%[2]s(tenantId: string, attributes: Record<string, unknown>): Promise<%[2]sRecord> {
    const now = this.clock.now();
    const record: %[2]sRecord = { id: newId('%[1]s'), tenantId, status: 'draft', createdAt: now, updatedAt: now, attributes };
    await this.store.put(record);
    this.events.emit('%[1]s.created', { tenantId, id: record.id });
    return record;
  }

  async archive%[2]s(tenantId: string, id: string): Promise<void> {
    const record = await this.get%[2]s(tenantId, id);
    if (record.status === 'archived') {
      return;
    }
    await this.store.put({ ...record, status: 'archived', updatedAt: this.clock.now() });
    this.events.emit('%[1]s.archived', { tenantId, id });
  }
}
`, r, R, extra)
}

func crudFile(header string, resources []string, extras map[string]string) string {
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\nimport { Store, EventBus, Clock, Page, newId } from './runtime';\nimport { ValidationError, NotFoundError } from './errors';\n")
	for _, r := range resources {
		sb.WriteString(crudModule(r, extras[r]))
	}
	return sb.String()
}

var npmPackages = []string{"glob", "rimraf", "inflight", "uuid", "request", "har-validator", "querystring", "source-map-url", "resolve-url", "urix", "mkdirp", "chokidar", "fsevents", "core-js", "popper.js", "tslint", "node-sass", "svgo", "stable", "sane", "left-pad", "w3c-hr-time", "abab", "domexception"}

var ciSuites = []string{"auth", "billing", "catalog", "checkout", "exports", "imports", "inventory", "invoices", "labels", "ledger", "notifications", "orders", "rates", "refunds", "reports", "routing", "search", "sessions", "shipments", "tenants", "tracking", "webhooks"}

func ciLog(n, start int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		ts := clock(start, 3, i)
		if s, ok := special[i]; ok {
			sb.WriteString(ts + " " + s + "\n")
			continue
		}
		switch {
		case i < n/4:
			p := npmPackages[i%len(npmPackages)]
			sb.WriteString(fmt.Sprintf("%s npm warn deprecated %s@%d.%d.%d: this version is no longer supported, upgrade to %d.x or later before the next major release\n", ts, p, 1+i%4, (i*3)%10, (i*7)%10, 2+i%4))
		case i < n/2:
			s := ciSuites[i%len(ciSuites)]
			sb.WriteString(fmt.Sprintf("%s [tsc] src/%s/handler_%02d.ts compiled in %d ms with 0 errors and %d warnings, emitted %d bytes\n", ts, s, i%17, 40+(i*31)%900, i%3, 1200+(i*977)%48000))
		default:
			s := ciSuites[(i*3)%len(ciSuites)]
			sb.WriteString(fmt.Sprintf("%s PASS tests/%s/%s_case_%03d.test.ts ran %d assertions across %d tests in %d ms, coverage %d.%d percent\n", ts, s, s, i, 4+(i*5)%60, 1+i%9, 8+(i*19)%400, 71+i%27, i%10))
		}
	}
	return sb.String()
}

var changeScopes = []string{"parser", "lexer", "planner", "executor", "cache", "wire", "cli", "docs", "driver", "pool", "tls", "metrics", "migrations", "types", "json", "retry"}

var changeTexts = []string{
	"handle empty input without allocating a scratch buffer on every call",
	"return a typed error instead of panicking when the peer closes early",
	"reduce lock contention on the hot path by splitting the shard map",
	"document the default timeout and the environment variable that overrides it",
	"stop retrying requests that failed with a non-retryable status code",
	"accept trailing commas in array literals when lenient mode is enabled",
	"emit one histogram per route instead of a single global histogram",
	"reject negative offsets early with a clear message for the caller",
	"keep prepared statements across reconnects when the server allows it",
	"fall back to the portable implementation on big-endian targets",
	"avoid copying large payloads twice when compression is turned off",
	"make the progress bar respect terminals that do not support color",
	"validate certificate chains before reusing a pooled connection",
}

func changelog(versions []string) string {
	var sb strings.Builder
	k := 0
	for vi, v := range versions {
		sb.WriteString(fmt.Sprintf("\n## {{n|version|%s}}\n\n", v))
		for j := 0; j < 16; j++ {
			kind := []string{"fix", "feat", "perf", "docs", "chore"}[(k+vi)%5]
			hash := fmt.Sprintf("%07x", ((k+17)*2654435761+vi*40503)&0xfffffff)
			if j == 0 {
				hash = "{{n|hash|" + hash + "}}"
			}
			sb.WriteString(fmt.Sprintf("- %s(%s): %s (#%d, %s)\n", kind, changeScopes[(k*7+vi)%len(changeScopes)], changeTexts[(k*5+vi*3)%len(changeTexts)], 3100+k*3+vi, hash))
			k++
		}
	}
	return sb.String()
}

package main

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"
)

func clock(start, step, i int) string {
	t := start + step*i + (i*37)%step
	return fmt.Sprintf("%02d:%02d:%02d.%03d", (t/3600)%24, (t/60)%60, t%60, (i*271)%1000)
}

var svcTemplates = []string{
	"level=info svc=rates msg=\"zone lookup\" carrier=%d prefix=%d cache=hit took=%dus shard=%d",
	"level=info svc=rates msg=\"quote computed\" lane=%d weight_kg=%d price_cents=%d rule=%d",
	"level=debug svc=rates msg=\"table refreshed\" rows=%d changed=%d took=%dms generation=%d",
	"level=info svc=labels msg=\"render ok\" template=%d pages=%d bytes=%d queue=%d",
	"level=warn svc=labels msg=\"slow render\" template=%d pages=%d took=%dms budget=%dms",
	"level=info svc=gateway msg=\"request\" route=%d status=200 bytes=%d upstream_ms=%d",
	"level=debug svc=gateway msg=\"rate limit bucket\" tenant=%d remaining=%d refill_s=%d burst=%d",
	"level=info svc=worker msg=\"job done\" kind=%d attempts=%d took=%dms backlog=%d",
	"level=debug svc=worker msg=\"heartbeat\" worker=%d idle_ms=%d leases=%d lag=%d",
	"level=info svc=ledger msg=\"batch committed\" batch=%d rows=%d shard=%d fsync_ms=%d",
	"level=debug svc=cache msg=\"evicted\" keys=%d freed_kb=%d ttl_s=%d generation=%d",
}

func svcLog(n, start int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		ts := clock(start, 5, i)
		if s, ok := special[i]; ok {
			sb.WriteString(ts + " " + s + "\n")
			continue
		}
		t := svcTemplates[(i*7+i/5)%len(svcTemplates)]
		sb.WriteString(ts + " " + fmt.Sprintf(t, 2+i%9, 10+(i*17)%83, 5+(i*31)%977, 1+(i*3)%12) + "\n")
	}
	return sb.String()
}

var testAreas = []string{"rates", "zones", "labels", "ledger", "auth", "tenants", "exports", "webhooks", "carriers", "manifests", "tracking", "billing"}

var testNames = []string{
	"returns_expected_zone_for_known_prefix", "rejects_empty_postcode_with_validation_error", "falls_back_to_carrier_default_when_table_misses",
	"rounds_weight_up_to_next_half_kilogram", "keeps_idempotency_key_for_fifteen_minutes", "renders_address_block_within_label_bounds",
	"writes_one_ledger_row_per_charge", "refuses_token_signed_with_rotated_key", "paginates_export_in_stable_order",
	"retries_webhook_three_times_then_parks", "parses_manifest_with_trailing_blank_lines", "applies_fuel_surcharge_after_discount",
}

func pytestLog(n int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		if s, ok := special[i]; ok {
			sb.WriteString(s + "\n")
			continue
		}
		a := testAreas[(i*5)%len(testAreas)]
		sb.WriteString(fmt.Sprintf("PASSED tests/%s/test_%s_%03d.py::test_%s in 0.%03ds (setup 0.%03ds, call 0.%03ds, teardown 0.001s) [%3d%%]\n",
			a, a, i, testNames[(i*7+i/3)%len(testNames)], 4+(i*13)%90, 1+i%4, 3+(i*11)%80, (i+1)*100/n))
	}
	return sb.String()
}

var crates = []string{"anyhow", "bytes", "chrono", "clap", "futures", "hyper", "itoa", "log", "memchr", "mio", "once_cell", "parking_lot", "rand", "regex", "ryu", "serde", "serde_json", "smallvec", "socket2", "syn", "thiserror", "tokio", "tracing", "url"}

func buildLog(n, start int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		ts := clock(start, 2, i)
		if s, ok := special[i]; ok {
			sb.WriteString(ts + " " + s + "\n")
			continue
		}
		c := crates[(i*5)%len(crates)]
		a := testAreas[(i*7)%len(testAreas)]
		switch {
		case i < n/3:
			sb.WriteString(fmt.Sprintf("%s    Compiling %s v%d.%d.%d (registry index, %d of %d units, %d ms, codegen-units=%d)\n", ts, c, i%3, (i*7)%20, (i*3)%15, i+1, n/3, 90+(i*41)%2100, 1+i%16))
		case i < n/2:
			sb.WriteString(fmt.Sprintf("%s warning: unused variable in crates/%s/src/handler_%02d.rs at line %d, column %d; prefix it with an underscore to silence this\n", ts, a, i%19, 10+(i*13)%400, 5+i%30))
		default:
			sb.WriteString(fmt.Sprintf("%s test %s::case_%03d::%s ... ok (%d ms, %d assertions, %d allocations)\n", ts, a, i, testNames[(i*5)%len(testNames)], 1+(i*17)%240, 2+(i*3)%40, 10+(i*29)%900))
		}
	}
	return sb.String()
}

var npmNames = []string{"acorn", "ajv", "ansi-regex", "ansi-styles", "argparse", "balanced-match", "brace-expansion", "braces", "callsites", "chalk", "color-convert", "color-name", "concat-map", "cross-spawn", "debug", "deep-is", "doctrine", "escape-string-regexp", "eslint-scope", "espree", "esquery", "esrecurse", "estraverse", "esutils", "fast-deep-equal", "fast-glob", "fastq", "fill-range", "find-up", "flat-cache", "flatted", "glob-parent", "globals", "graphemer", "has-flag", "ignore", "import-fresh", "imurmurhash", "is-extglob", "is-glob", "is-number", "isexe", "js-yaml", "json-buffer", "keyv", "levn", "locate-path", "lodash.merge", "merge2", "micromatch", "minimatch", "ms", "natural-compare", "optionator", "p-limit", "p-locate", "parent-module", "path-exists", "path-key", "picomatch", "prelude-ls", "punycode", "queue-microtask", "resolve-from", "reusify", "run-parallel", "semver", "shebang-command", "shebang-regex", "slash", "strip-ansi", "strip-json-comments", "supports-color", "text-table", "to-regex-range", "type-check", "type-fest", "uri-js", "which", "word-wrap", "yocto-queue"}

func integrity(name, version string) string {
	sum := sha512.Sum512([]byte("holdout3:" + name + "@" + version))
	return "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
}

func lockEntries(n int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		if s, ok := special[i]; ok {
			sb.WriteString(s)
			continue
		}
		name := npmNames[i%len(npmNames)]
		path := "node_modules/" + name
		if i >= len(npmNames) {
			path = "node_modules/" + npmNames[(i*7)%len(npmNames)] + "/node_modules/" + name
		}
		version := fmt.Sprintf("%d.%d.%d", 1+i%8, (i*3)%12, (i*7)%10)
		hash := integrity(name, version)
		ver := version
		if i%9 == 0 {
			ver = "{{n|version|" + version + "}}"
			hash = "{{n|checksum|" + hash + "}}"
		}
		sb.WriteString(fmt.Sprintf("    \"%s\": {\n      \"version\": \"%s\",\n      \"resolved\": \"https://registry.npmjs.org/%s/-/%s-%s.tgz\",\n      \"integrity\": \"%s\",\n      \"dev\": %v,\n      \"license\": \"%s\",\n      \"engines\": { \"node\": \">=%d\" }\n    },\n",
			path, ver, name, name, version, hash, i%3 != 0, []string{"MIT", "ISC", "BSD-2-Clause", "Apache-2.0"}[i%4], 14+2*(i%4)))
	}
	return sb.String()
}

var eventKinds = []string{"picked_up", "arrived_hub", "departed_hub", "customs_hold", "customs_released", "out_for_delivery", "delivered", "exception"}

var hubs = []string{"BRS1", "RTM2", "HAM4", "LYS1", "MXP3", "GDN1", "OPO2", "ATH1", "IST5", "LOS1", "NBO2", "SIN3"}

func sqlEventRows(n int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		if s, ok := special[i]; ok {
			sb.WriteString(s + "\n")
			continue
		}
		sb.WriteString(fmt.Sprintf("INSERT INTO shipment_event (id, shipment_id, kind, hub, seq, weight_g, scanned_by, note) VALUES (%d, %d, '%s', '%s', %d, %d, 'scanner-%02d', 'auto: lane %d, belt %d, tote %d');\n",
			700000+i*13, 48000+(i*37)%900, eventKinds[(i*3+i/8)%len(eventKinds)], hubs[(i*5)%len(hubs)], 1+i%7, 250+(i*977)%24000, 1+i%40, 1+(i*7)%60, 1+i%12, 100+(i*31)%800))
	}
	return sb.String()
}

var etlFields = []string{"consignee_name", "consignee_street", "consignee_city", "consignee_postcode", "consignee_country", "consignee_phone", "consignee_email", "declared_value", "currency", "weight_kg", "incoterm", "hs_code", "carrier_code", "service_level", "pickup_window", "delivery_window", "shipper_name", "shipper_reference", "shipper_phone", "notify_party", "notify_email", "dangerous_goods_class", "temperature_range", "insurance_reference"}

func etlModule() string {
	var sb strings.Builder
	sb.WriteString("import csv\nimport logging\nfrom dataclasses import dataclass\nfrom typing import Iterator, Optional\n\nlog = logging.getLogger(\"legacy_import\")\n\n\n@dataclass\nclass Row:\n    line: int\n    raw: dict\n    errors: list\n\n")
	for i, f := range etlFields {
		sb.WriteString(fmt.Sprintf(`
def clean_%[1]s(row: Row) -> Optional[str]:
    value = (row.raw.get("%[1]s") or "").strip()
    if not value:
        row.errors.append("line %%d: %[1]s is empty" %% row.line)
        return None
    if value.lower() in ("n/a", "none", "null", "-", "unknown"):
        row.errors.append("line %%d: %[1]s holds a placeholder instead of a value" %% row.line)
        return None
    value = " ".join(value.split())
    if len(value) > %[2]d:
        log.warning("line %%d: %[1]s longer than %[2]d characters, truncating", row.line)
        value = value[:%[2]d]
    return value

`, f, 24+(i*17)%120))
	}
	sb.WriteString("\ndef read_rows(path: str) -> Iterator[Row]:\n    with open(path, newline=\"\", encoding=\"utf-8-sig\") as handle:\n        for number, raw in enumerate(csv.DictReader(handle, delimiter=\";\"), start=2):\n            yield Row(line=number, raw=raw, errors=[])\n\n\ndef transform(row: Row) -> dict:\n    out = {}\n")
	for _, f := range etlFields {
		sb.WriteString(fmt.Sprintf("    out[\"%[1]s\"] = clean_%[1]s(row)\n", f))
	}
	sb.WriteString("    return out\n")
	return sb.String()
}

var lanePorts = []string{"Bristol", "Rotterdam", "Hamburg", "Gdynia", "Bergen", "Tampere", "Porto", "Lisbon", "Piraeus", "Istanbul", "Lagos", "Mombasa", "Singapore", "Surabaya", "Santiago", "Auckland"}

func rateTable(n int) string {
	var sb strings.Builder
	sb.WriteString("Lane | Service | Transit days | Rate per pallet (EUR) | Fuel surcharge | Minimum pallets | Valid from quarter\n")
	for i := 0; i < n; i++ {
		a, b := lanePorts[(i*3)%len(lanePorts)], lanePorts[(i*5+7)%len(lanePorts)]
		if a == b {
			b = lanePorts[(i*5+8)%len(lanePorts)]
		}
		sb.WriteString(fmt.Sprintf("{{l|%s}} to {{l|%s}} | %s | %d | %d.%02d | %d.%d%% | %d | Q%d\n", a, b, []string{"standard", "express", "reefer", "groupage"}[i%4], 2+(i*7)%19, 84+(i*37)%410, (i*13)%100, 3+i%9, i%10, 1+i%6, 1+i%4))
	}
	return sb.String()
}

var alertNames = []string{"p99 latency above 5s", "error ratio above 2%", "queue depth above 10k", "replica lag above 30s", "disk usage above 85%", "render failures above 1%", "certificate expires within 14d", "connection pool saturated"}

func alertTimeline(n, start int, special map[int]string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		ts := clock(start, 41, i)[:8]
		if s, ok := special[i]; ok {
			sb.WriteString(ts + " " + s + "\n")
			continue
		}
		state := "FIRING"
		if i%3 == 2 {
			state = "RESOLVED"
		}
		sb.WriteString(fmt.Sprintf("%s %s [sev%d] %s, region %s, value %d.%d, threshold %d, for %dm, fingerprint %08x\n", ts, state, 2+i%3, alertNames[(i*3+i/4)%len(alertNames)], []string{"eu-central", "eu-west", "ap-southeast"}[i%3], 3+(i*17)%90, i%10, 2+(i*5)%40, 1+(i*7)%25, (i+11)*2654435761&0xffffffff))
	}
	return sb.String()
}

/*
Package redact provides lightweight, pattern-based redaction utilities for
obscuring sensitive data before logging or debugging output is emitted.

# Problem

Operational logs and diagnostics often include raw HTTP headers, JSON payloads,
or URL-encoded form data. Without sanitization, these records can leak secrets
such as credentials, tokens, API keys, session identifiers, personal data, or
payment details. This package offers a fast, single-pass redaction step that
can be applied at log boundaries.

# API

The canonical entry points are [String], [Bytes], [AppendTo], [BytesToString]
and [Pooled]; they all run the same redaction engine and differ only in input
and output handling. The HTTPData* functions are retained as compatibility
aliases.

# Custom Redactors

The package-level functions use the default configuration ([Default]). [New]
builds an independent [Redactor] instance with the same API when a component
needs different behavior:

	re := redact.New(
		redact.WithMarker("#REDACTED#"),          // custom placeholder
		redact.WithLuhnCheck(true),                // instance-scoped Luhn gate
		redact.WithExtraTokens("floof"),           // company-specific key tokens
		redact.WithoutTokens("amount", "balance"), // keep fintech fields readable
		redact.WithoutRules(redact.RuleCards),     // disable whole rule classes
	)
	safe := re.String(rawPayload)

Instances are immutable after construction and safe for concurrent use.

# What It Redacts

Each call applies multiple redaction rules in a single pass:

  - HTTP headers whose name is sensitive (`Authorization`,
    `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key`,
    `X-Auth-Token`, `X-Amz-Security-Token`, ...), preserving the header name
    while replacing the whole value. Sensitivity uses the same tokenized
    keyword check as JSON and URL-encoded keys, including a shared allowlist of
    well-known keys that tokenize to a sensitive keyword but never carry a
    secret (CSP/HSTS response headers, the `Access-Control-Allow-Credentials`
    and `WWW-Authenticate`/`Proxy-Authenticate` headers, and the
    `securityContext`/`securityGroups` structural fields), which stay visible
    on every surface.
  - JSON key/value pairs whose key name contains sensitive keywords
    (authentication/session, crypto markers, legal-signing, financial, and PII
    keyword groups). When the value is a JSON object or array, the entire
    nested container is replaced with the marker.
  - URL-encoded key/value pairs with matching sensitive key names.
  - XML/HTML elements with sensitive names and flat text content
    (`<password>...</password>`); sensitive XML attributes are covered by the
    URL-encoded rule.
  - URL userinfo passwords anywhere in the text
    (`postgres://user:secret@host` -> `postgres://user:***@host`), catching
    bare DSNs in error messages.
  - JWT/JWE compact tokens (`eyJ...`) in free text, query strings, and JSON
    values.
  - Well-known vendor credential literals by their unmistakable prefixes:
    GitHub (`ghp_`, `github_pat_`, ...), Slack (`xoxb-`, ...), Stripe
    (`sk_live_`, `whsec_`, ...), OpenAI/Anthropic (`sk-`), Hugging Face
    (`hf_`), SendGrid (`SG.`), AWS access key ids (`AKIA`, `ASIA`), Google
    (`AIza`), GitLab (`glpat-`), DigitalOcean, Docker Hub, and Shopify
    tokens.
  - PEM `-----BEGIN ... PRIVATE KEY-----` blocks: the base64 body is replaced
    with a single marker line (public blocks such as CERTIFICATE stay
    visible). Blocks embedded mid-line — e.g. a JSON string carrying a blob
    with escaped `\n` sequences — are redacted too.
  - Credit-card numbers: contiguous digit runs, and runs grouped by single
    spaces or dashes (`4012 8888 8888 1881`), bounded by non-word characters.

# Key Matching

Keyword matching is token-exact after normalization: camelCase, snake_case,
kebab-case, and acronym runs all tokenize (`apiKey`, `api_key`, `API-KEY` and
`APIKey` all yield `api` + `key`). Two bounded retries widen this:

  - a trailing digit run is stripped, so a numbered field matches its base
    (`password2`, `cvv2`, `key1`); a digit rarely ends an ordinary key word, so
    this retry runs against the whole keyword set. A key-size label whose stem is
    a keyword (`rsa2048`, `dsa1024`) is redacted as a harmless corollary — it is
    a false positive on non-secret metadata, never a leak.
  - a trailing plural `s` is stripped, but only the strong roots below are
    retried — so `tokens`, `apikeys` and `newpasswords` redact while plurals of
    ordinary-word keywords (`keys`, a JWKS array; `cells`; `accounts`;
    `payments`) stay visible.

A few names are sensitive only as a two-word pair, where neither word is a
keyword alone: `first`/`last` + `name`, `national` + `id`, and `connection` +
`string`. So `firstName`, `nationalId`/`national_id` and
`connectionString`/`connection_string` all match, while `international` and
`connectionTimeout` do not.

A closed compound — an all-lowercase glued name with no boundary to split on, as
HTML forms use (`<input name="newpassword">`) — matches in two cases only:

  - it is one of the enumerated keywords (`apikey`, `passphrase`,
    `clientsecret`, `connectionstring`, `masterkey`, `sessioncookie`, ...); or
  - it ends in one of a short list of unambiguous roots: `password`, `passwd`,
    `pwd`, `passphrase`, `passcode`, `secret`, `token`, `signature`,
    `credential`, `creditcard`, `debitcard`, `cardnumber`, and the `*key`
    family (`apikey`, `accesskey`, `secretkey`, `sessionkey`, `signingkey`,
    `encryptionkey`, `authkey`, `privatekey`, `privkey`, `sshkey`). So
    `newpassword`, `oldpassword` and `awssecretkey` are redacted.

The root list is deliberately short, and matching is never a plain substring
search: near-miss words like "monkey" never match `key`, and "wildcard" never
matches `card`. House-style names outside these rules are best added with
[WithExtraTokens].

CRLF line endings (as produced by httputil.DumpRequest and DumpResponse) are
preserved. All matched sensitive values are replaced with a constant marker so
output remains structurally useful while hiding private content. Redaction is
convergent: re-redacting output never reveals more, is byte-stable in a single
pass on well-formed input, and reaches a fixed point after at most one extra
pass on pathological (structurally ambiguous) input — so layered redaction does
not keep mangling logs.

# Credit-Card Detection and the Optional Luhn Gate

By default, any 13-19 digit run (contiguous, or grouped by single spaces or
dashes) that matches a known card prefix and network length (Visa, Mastercard,
Amex, Discover, Diners, JCB, UnionPay, ...) and is bounded by non-word
characters is redacted. This is deliberate over-redaction: it is the safe
default and may also redact unrelated numeric identifiers that happen to share
a card prefix and length. Grouped-format detection excludes the 14-digit
Diners and the legacy 15-digit ranges (JCB 2131, Diners 1800), whose prefixes
collide with common phone-number formats ("1 800 555 0199 1234"); those still
match as contiguous runs.

Maestro is handled conservatively: its well-known 4-digit issuer prefixes
(5018, 5020, 5038, 5893, 6304, 6759, 6761-6763) are detected at 16-19 digits by
default, while the short 12-15 digit Maestro forms are only detected when the
Luhn gate below is enabled, because a short prefix-and-length match alone would
collide with far too many ordinary identifiers. The broader Maestro ranges are
not enrolled as Maestro, though several of those prefixes are still redacted
under other networks (56-57 as Mastercard, 62 as UnionPay, 65 as Discover); only
50, 58, 59, 61, 63, 66, 68 and 69 (minus the well-known IINs) stay visible at 16
digits.

Callers that prefer fewer false positives can enable an additional Luhn-checksum
gate with [WithLuhnCheck]. When enabled, a digit run is only redacted if it
matches a known prefix AND passes the Luhn checksum:

	re := redact.New(redact.WithLuhnCheck(true))
	safe := re.String(rawPayload)

The gate is instance-scoped and fixed at construction, and defaults to off. The
package-level functions ([String], [Bytes], ...) always run with it off; a
component that wants it must build its own [Redactor]. Enabling it may cause
malformed or non-Luhn test numbers to be left visible.

# Key Features

  - Generic single-pass redaction for logs, HTTP dumps, JSON, and form data.
  - Broad keyword families to catch many real-world secret field names.
  - Preserves surrounding structure to keep logs searchable and debuggable.
  - No external dependencies beyond the Go standard library.

# Usage

	safe := redact.String(rawPayload)
	logger.Info("request", "payload", safe)

For high-throughput paths, reuse an output buffer to avoid per-call allocations:

	var dst []byte
	for _, payload := range payloads {
		dst = redact.AppendTo(dst, payload)
		logger.Info("request", "payload", string(dst))
	}

# Important Notes

This package is best-effort pattern redaction, not a formal data-loss
prevention system. Always treat output as potentially sensitive and combine this
with least-privilege logging practices and structured logging controls.

The recognized input shapes are the ones listed above (HTTP headers, JSON,
URL-encoded pairs, XML, DSNs, and free-text tokens). Go's native rendering of
composite values is not among them: neither `fmt.Sprintf("%+v", req)` nor
`map[password:secret]` is redacted, because they carry no `=`, `:` or quoted-key
structure the engine can anchor on. Redact the fields, or marshal to JSON first.

multipart/form-data body fields are also not redacted: the field name lives on a
`Content-Disposition` line and its value sits on a later line separated by a
blank line, so the two are structurally decoupled across lines and the engine
has nothing to anchor a single-pass rule on. Redact multipart field values
before logging a captured body.

The marker is assumed to be inert: a URL-encoded value whose text begins with
the marker followed by a separator (`password=*** tail`) is read as already
redacted and its tail is left alone. That is what keeps a second redaction pass
from eating the rest of a logfmt line (`password=*** user=bob`); the cost is
that an input value that literally starts with `*** ` keeps its tail visible.

An unquoted URL-encoded value ends at a structural byte (`&`, `,`, `;`, `<`,
`>`, quote, or a line break) so redaction cannot consume a following pair or the
document's structure. A secret that contains one of those bytes raw therefore
keeps the tail after it visible (`password=Pa,ssword1` -> `password=***,ssword1`).
Real form bodies percent-encode those bytes; this only surfaces in hand-written
`key=value` text. Quote the value (`password="Pa,ssword1"`) and it is redacted
whole.

Deliberate non-goals (too collision-prone to match on shape alone): Telegram
bot tokens (digits:base64 collides with host:port shapes) and bare
"Basic <base64>" blobs in prose (header-positioned Basic credentials are
covered by the Authorization rule); obsolete obs-fold header continuation
lines.
*/
package redact

import (
	"sync"
	"unsafe"
)

// Redaction patterns and replacements.
const (
	// RedactionMarker is the placeholder used to replace sensitive values.
	RedactionMarker = `***`
)

// Reusable byte constants.
var (
	redactedBytes = []byte(RedactionMarker) //nolint:gochecknoglobals

	redactionBufferPool = sync.Pool{New: newRedactionBuffer} //nolint:gochecknoglobals
)

// String redacts sensitive data from s (headers, secret fields, and card
// patterns; see the package documentation) and returns the sanitized string.
// It routes through the pooled output buffer to avoid a dedicated per-call
// allocation.
func String(s string) string {
	return defaultRedactor.String(s)
}

// Bytes redacts sensitive data from b and returns the result as a new byte
// slice. The input is never modified.
func Bytes(b []byte) []byte {
	return defaultRedactor.Bytes(b)
}

// AppendTo redacts sensitive data from src and appends the result into dst
// (after resetting its length to zero), allowing callers to reuse output
// buffers across calls. Like the append built-in, it returns the possibly
// reallocated destination slice. dst and src may share storage (an in-place
// AppendTo(b, b) is detected and handled by redacting into a fresh buffer).
func AppendTo(dst, src []byte) []byte {
	return defaultRedactor.AppendTo(dst, src)
}

// Pooled redacts sensitive data from src using an internal pooled buffer and
// passes the result to consume.
//
// The passed slice is only valid during the consume call and must not be
// retained after consume returns.
func Pooled(src []byte, consume func([]byte)) {
	defaultRedactor.Pooled(src, consume)
}

// BytesToString redacts sensitive data from a byte slice and returns the
// result as a string. It uses a pooled output buffer to reduce allocations and
// is the preferred form when the caller already holds a []byte (e.g. from
// httputil.DumpRequest / DumpResponse) and needs a string.
func BytesToString(b []byte) string {
	return defaultRedactor.BytesToString(b)
}

// redactInto applies all enabled redaction rules while appending output into
// dst, which is reset to length 0 before use.
//
//nolint:gocognit // Deliberately flat hot loop: one dispatch pass per byte class.
func (re *Redactor) redactInto(dst, src []byte) []byte {
	// Tolerate a zero-value Redactor (nil marker / key memo) without panicking
	// or emitting an empty marker; a [New]-built instance passes through here.
	if re.marker == nil || re.keyMemo == nil {
		re = re.usableRedactor()
	}

	dst = dst[:0]

	for i := 0; i < len(src); {
		if i == 0 || src[i-1] == '\n' {
			if next, redacted, ok := re.appendLineStartRedactionAt(src, i, dst); ok {
				dst = redacted
				i = next

				continue
			}
		}

		// Digit runs are handled before the rule dispatch: no trigger byte is
		// a digit, and identifier-heavy logs (trace ids, UUIDs) are dominated
		// by digit runs, so they skip the dispatch call entirely.
		if isDigitByte(src[i]) {
			i, dst = re.appendDigitRunAt(src, i, dst)

			continue
		}

		if next, redacted, ok := re.appendTriggeredRedactionAt(src, i, dst); ok {
			dst = redacted
			i = next

			continue
		}

		if src[i] == '\n' {
			dst = append(dst, src[i])
			i++

			continue
		}

		j := bulkTextEnd(src, i)
		dst = append(dst, src[i:j]...)
		i = j
	}

	return dst
}

// appendLineStartRedactionAt applies the line-anchored rules (PEM blocks and
// sensitive HTTP headers) at the line beginning at src[i].
func (re *Redactor) appendLineStartRedactionAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if src[i] == '-' && re.enabled(RulePEM) {
		if next, redacted, ok := re.appendRedactedPEMKeyAt(src, i, dst); ok {
			return next, redacted, true
		}
	}

	if re.enabled(RuleHeaders) {
		// Skip leading indentation and an optional curl/resty trace decoration
		// ("> "/"< ") so nested (indented) and trace-prefixed header lines are
		// covered too — the value of an indented "password:" or a "> Authorization:"
		// header leaked before. The prefix itself is preserved in the output.
		nameStart := skipHeaderLinePrefix(src, i)
		if valueStart, ok := re.sensitiveHeaderValueStart(src, nameStart); ok {
			dst = append(dst, src[i:valueStart]...)
			dst = append(dst, re.marker...)

			return headerValueEnd(src, valueStart), dst, true
		}
	}

	return 0, dst, false
}

// appendTriggeredRedactionAt dispatches the byte-triggered rules: JSON keys,
// URL-encoded pairs, URL userinfo passwords, JWTs, XML elements, inline PEM
// blocks, and vendor credential tokens.
//
//nolint:gocyclo,cyclop // Irreducible one-case-per-rule dispatch switch.
func (re *Redactor) appendTriggeredRedactionAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if rule := triggerRule(src[i]); rule == 0 || !re.enabled(rule) {
		return 0, dst, false
	}

	switch src[i] {
	case '"':
		if re.likelyJSONKeyStart(src, i) {
			return re.appendRedactedSensitiveJSONAt(src, i, dst)
		}
		// A '"' preceded by a backslash may open an escaped JSON key (a JSON
		// document embedded as a string value of another JSON document). The
		// inline backslash test keeps ordinary quotes off the escaped path.
		if i > 0 && src[i-1] == '\\' {
			return re.appendRedactedEscapedJSONAt(src, i, dst)
		}
	case '=':
		return re.appendRedactedURLEncodedValueAt(src, i, dst)
	case ':':
		return re.appendRedactedURLPasswordAt(src, i, dst)
	case 'e':
		return re.appendRedactedJWTAt(src, i, dst)
	case '<':
		return re.appendRedactedXMLValueAt(src, i, dst)
	case '-':
		return re.appendRedactedInlinePEMKeyAt(src, i, dst)
	case 'g', 'x', 's', 'r', 'w', 'd', 'A', 'h', 'S':
		return re.appendRedactedVendorTokenAt(src, i, dst)
	}

	return 0, dst, false
}

// triggerRule maps a dispatch byte to the rule class it triggers, or 0 when
// the byte triggers no rule.
func triggerRule(c byte) Rule {
	switch c {
	case '"':
		return RuleJSON
	case '=':
		return RuleURLEncoded
	case ':':
		return RuleUserinfo
	case 'e':
		return RuleJWT
	case '<':
		return RuleXML
	case '-':
		return RulePEM
	case 'g', 'x', 's', 'r', 'w', 'd', 'A', 'h', 'S':
		return RuleVendorTokens
	}

	return 0
}

// Trigger classes for the bulk-copy scan: most bytes are trigNone and cost a
// single table load; the rare candidate classes run their cheap prefilter
// before breaking out to the rule dispatch.
const (
	trigNone byte = iota
	trigStop
	trigColon
	trigJWT
	trigDash
	trigVendor
)

// bulkTrigger classifies every byte for bulkTextEnd: hard stops (rule bytes
// and digits), and the prefiltered candidate starts of the userinfo (':'),
// JWT ('e'), inline-PEM ('-'), and vendor-token rules. An escaped JSON key
// ('\"') is caught at its '"', already a hard stop, so '\' needs no class here.
var bulkTrigger = [256]byte{ //nolint:gochecknoglobals
	'"': trigStop, '=': trigStop, '\n': trigStop, '<': trigStop,
	'0': trigStop, '1': trigStop, '2': trigStop, '3': trigStop, '4': trigStop,
	'5': trigStop, '6': trigStop, '7': trigStop, '8': trigStop, '9': trigStop,
	':': trigColon,
	'e': trigJWT,
	'-': trigDash,
	'g': trigVendor, 'x': trigVendor, 's': trigVendor, 'r': trigVendor,
	'w': trigVendor, 'd': trigVendor, 'A': trigVendor, 'h': trigVendor,
	'S': trigVendor,
}

// bulkTextEnd returns the end of the plain-text run starting at src[i]: the
// scan stops at bytes that begin a redaction rule. Ordinary bytes cost one
// table load; candidate bytes run a short prefilter so the bulk copy stays
// tight for ordinary text.
//
//nolint:gocognit,gocyclo,cyclop // Deliberately flat hot loop: prefilters must stay inline.
func bulkTextEnd(src []byte, i int) int {
	j := i + 1
	for j < len(src) {
		switch bulkTrigger[src[j]] {
		case trigStop:
			return j
		case trigColon:
			if j+2 < len(src) && src[j+1] == '/' && src[j+2] == '/' {
				return j
			}
		case trigJWT:
			if j+2 < len(src) && src[j+1] == 'y' && src[j+2] == 'J' && !isWordChar(src[j-1]) {
				return j
			}
		case trigDash:
			if hasPrefixAt(src, j, "-----B") {
				return j
			}
		case trigVendor:
			if !isWordChar(src[j-1]) && isVendorTokenStart(src, j) {
				return j
			}
		}

		j++
	}

	return j
}

// appendDigitRunAt handles a digit at src[i]: digit runs glued to word
// characters are identifiers and copied verbatim; free-standing runs are
// checked as contiguous and grouped card numbers.
func (re *Redactor) appendDigitRunAt(src []byte, i int, dst []byte) (int, []byte) {
	if i > 0 && isWordChar(src[i-1]) {
		j := scanDigits(src, i)

		return j, append(dst, src[i:j]...)
	}

	j := scanDigits(src, i)

	if !re.enabled(RuleCards) || (j < len(src) && isWordChar(src[j])) {
		return j, append(dst, src[i:j]...)
	}

	if re.isCreditCard(src[i:j]) {
		return j, append(dst, re.marker...)
	}

	if end, ok := re.scanGroupedCardSpan(src, i, j); ok {
		return end, append(dst, re.marker...)
	}

	return j, append(dst, src[i:j]...)
}

func isWordChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// backingOverlap reports whether the backing arrays of a and b overlap in
// memory, using the same address-range test the standard library's crypto
// packages use to reject in-place aliasing.
func backingOverlap(a, b []byte) bool {
	if cap(a) == 0 || cap(b) == 0 {
		return false
	}

	aBeg := uintptr(unsafe.Pointer(unsafe.SliceData(a)))
	bBeg := uintptr(unsafe.Pointer(unsafe.SliceData(b)))

	return aBeg < bBeg+uintptr(cap(b)) && bBeg < aBeg+uintptr(cap(a))
}

func isDigitByte(c byte) bool {
	return c >= '0' && c <= '9'
}

// newRedactionBuffer is the sync.Pool factory for reusable output buffers.
func newRedactionBuffer() any {
	b := make([]byte, 0, 1024)

	return &b
}

func getPooledRedactionBuffer(minCap int) []byte {
	bp, _ := redactionBufferPool.Get().(*[]byte)
	if bp == nil {
		b := make([]byte, 0, minCap)

		return b
	}

	b := *bp
	if cap(b) < minCap {
		return make([]byte, 0, minCap)
	}

	return b[:0]
}

func putPooledRedactionBuffer(b []byte) {
	// Avoid keeping very large buffers indefinitely in the pool.
	const maxPooledCap = 1 << 20
	if cap(b) > maxPooledCap {
		return
	}

	b = b[:0]
	redactionBufferPool.Put(&b)
}

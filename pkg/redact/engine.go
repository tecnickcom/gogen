package redact

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
		// covered too, including an indented "password:" or a "> Authorization:"
		// header. The prefix itself is preserved in the output.
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

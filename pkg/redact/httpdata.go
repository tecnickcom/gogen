package redact

// This file holds the original HTTPData* entry points, retained as thin
// compatibility aliases of the canonical API in redact.go ([String], [Bytes],
// [AppendTo], [BytesToString], [Pooled]). New code should prefer the canonical
// names; existing callers keep working unchanged.

// HTTPData is an alias of [String], kept for backward compatibility.
func HTTPData(s string) string {
	return String(s)
}

// HTTPDataBytes is an alias of [Bytes], kept for backward compatibility.
func HTTPDataBytes(b []byte) []byte {
	return Bytes(b)
}

// HTTPDataBytesInto is an alias of [AppendTo], kept for backward compatibility.
func HTTPDataBytesInto(dst, src []byte) []byte {
	return AppendTo(dst, src)
}

// HTTPDataBytesPooled is an alias of [Pooled], kept for backward compatibility.
func HTTPDataBytesPooled(src []byte, consume func([]byte)) {
	Pooled(src, consume)
}

// HTTPDataString is an alias of [BytesToString], kept for backward
// compatibility.
func HTTPDataString(b []byte) string {
	return BytesToString(b)
}

package redact

// InsecureNoRedaction disables redaction entirely: it returns the input
// verbatim.
//
// It exists as a ready-made value for the redact-function option of the
// httpclient, httpserver, and httpreverseproxy packages, so a caller that must
// switch redaction off does not have to hand-roll a pass-through closure:
//
//	httpserver.WithRedactFn(redact.InsecureNoRedaction)
//
// Every secret reaching a logged request/response dump, query string, or error
// URL is then written to the logs in the clear. Use it only where the secrets
// are already handled elsewhere (redaction applied downstream in the log
// pipeline, or a log sink trusted with raw payloads), never merely to quiet
// noisy output or to inspect a payload in production.
//
// It is deliberately named to be conspicuous in a diff, so that disabling
// redaction is always a visible, greppable decision. Omitting the option keeps
// the safe default ([Default]), and a nil function is ignored rather than
// treated as a bypass: redaction can never be lost by oversight, only by
// naming this function.
func InsecureNoRedaction(b []byte) string {
	return string(b)
}

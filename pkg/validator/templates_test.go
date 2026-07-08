package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/errutil"
)

// TestErrorTemplates_CoversCustomTags guards that every custom validation tag
// (except the "falseif" helper, which never surfaces as an error) has a
// matching human-readable template, so adding a custom tag without a message
// fails the build.
func TestErrorTemplates_CoversCustomTags(t *testing.T) {
	t.Parallel()

	templates := ErrorTemplates()

	for tag := range CustomValidationTags() {
		if tag == "falseif" {
			continue
		}

		_, ok := templates[tag]
		require.Truef(t, ok, "custom tag %q has no error template", tag)
	}
}

// TestErrorTemplates_NoGenericFallback guards that built-in tags which
// previously lacked templates now render specific human-readable messages
// instead of the generic "fails the rule" fallback. It also implicitly proves
// each template parses, executes, and interpolates without error.
func TestErrorTemplates_NoGenericFallback(t *testing.T) {
	t.Parallel()

	v, err := New(WithFieldNameTag("json"), WithErrorTemplates(ErrorTemplates()))
	require.NoError(t, err)

	type sample struct {
		Mime   string `json:"mime"   validate:"mimetype=image/png"`
		None   string `json:"none"   validate:"noneof=a b c"`
		NoneCI string `json:"noneci" validate:"noneofci=a b c"`
		Origin string `json:"origin" validate:"origin"`
		BCP47  string `json:"bcp47"  validate:"bcp47_strict_language_tag"`
	}

	err = v.ValidateStruct(sample{
		Mime:   "x",
		None:   "a",
		NoneCI: "A",
		Origin: "http://example.com/#frag",
		BCP47:  "!!",
	})
	require.Error(t, err)

	errs := errutil.Errors(err)
	require.Len(t, errs, 5, "all five tags must fail: %v", errs)

	for _, e := range errs {
		require.NotContainsf(t, e.Error(), "fails the rule", "tag rendered generic fallback: %q", e.Error())
	}
}

package filter

const (
	// TypeEqual is a filter type that matches exactly the reference value.
	TypeEqual = "equal"
)

type equal struct {
	ref interface{}
}

func newEqual(reference interface{}) Evaluator {
	return &equal{
		ref: convertValues(reference),
	}
}

// Evaluate returns whether reference and actual are considered equal.
// It converts numerical values implicitly before comparison.
func (e *equal) Evaluate(value interface{}) bool {
	value = convertValues(value)

	if value == e.ref {
		return true
	}

	if isNil(value) && isNil(e.ref) {
		return true
	}

	return false
}

package validator

// Error is a custom error adding a Field member.
type Error struct {
	// Tag is the validation tag that failed (e.g. "max").
	Tag string

	// Param is the Tag's parameter value (if any - e.g. "10").
	Param string

	// FullTag is the validation tag that failed with included parameters (e.g. "max=10").
	FullTag string

	// Namespace for the field error, with the tag name taking precedence over the field's actual name.
	Namespace string

	// StructNamespace is the namespace for the field error, with the field's actual name.
	StructNamespace string

	// Field is the field name with the tag name taking precedence over the field's actual name.
	Field string

	// StructField is the field's actual name from the struct.
	StructField string

	// Kind is the Field's string representation of the kind (e.g. Int,Slice,...).
	Kind string

	// Value is the actual field's value.
	Value any

	// Err is the translated error message.
	Err string
}

// Error returns a string representation of the error.
func (e *Error) Error() string {
	return e.Err
}

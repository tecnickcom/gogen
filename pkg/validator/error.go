package validator

// Error is a custom error adding a Field member.
type Error struct {
	// Tag is the validation tag that failed.
	// If the validation was an alias, this will return the alias name and not the underlying tag that failed.
	//
	// eg. alias "iscolor": "hexcolor|rgb|rgba|hsl|hsla"
	// will return "iscolor"
	Tag string

	// ActualTag is the validation tag that failed,
	// even if an alias the actual tag within the alias will be returned.
	// If an 'or' validation fails the entire or will be returned.
	//
	// eg. alias "iscolor": "hexcolor|rgb|rgba|hsl|hsla"
	// will return "hexcolor|rgb|rgba|hsl|hsla"
	ActualTag string

	// Namespace for the field error,
	// with the tag name taking precedence over the field's actual name.
	Namespace string

	// StructNamespace is the namespace for the field error,
	// with the field's actual name.
	StructNamespace string

	// Field is the field name with the tag name taking precedence over the field's actual name.
	Field string

	// StructField is the field's actual name from the struct, when able to determine.
	StructField string

	// Value is the actual field's value.
	Value interface{}

	// Param is the param value.
	Param string

	// Kind is the Field's reflect Kind as string.
	Kind string

	// Type is the Field's reflect Type as string.
	Type string

	// OrigErr is the original error message.
	OrigErr string

	// Err is the translated error message.
	Err string
}

// Error returns a string representation of the error.
func (e *Error) Error() string {
	return e.Err
}

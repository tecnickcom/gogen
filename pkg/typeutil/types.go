package typeutil

// Int is a constraint for signed integer types.
type Int interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// UInt is a constraint for unsigned integer types.
type UInt interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Float is a constraint for float types.
type Float interface {
	~float32 | ~float64
}

// Number is a constraint for all integer and float numbers.
type Number interface {
	Int | UInt | Float
}

// Ordered is a constraint that permits any ordered type:
// any type that supports the operators < <= >= >.
type Ordered interface {
	Number | ~string
}

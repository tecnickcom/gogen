package validator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/errutil"
)

type testCustomTagStruct struct {
	E164NoPlus             string  `json:"e164_no_plus"             validate:"e164noplus"`
	EIN                    string  `json:"ein"                      validate:"ein"`
	USZIPCode              string  `json:"zip"                      validate:"zipcode"`
	USZIPCodeB             string  `json:"zip_b"                    validate:"zipcode"`
	Country                string  `json:"country"                  validate:"iso3166_1_alpha2"`
	State                  string  `json:"state"                    validate:"usstate"`
	StateB                 string  `json:"state_b"                  validate:"falseif=Country|usstate"`
	StateC                 string  `json:"state_c"                  validate:"falseif=Country US|usstate"`
	StateD                 string  `json:"state_d"                  validate:"falseif|usstate"`
	StateE                 string  `json:"state_e"                  validate:"falseif=Country US|usterritory"`
	FalseIfMissing         string  `json:"falseif_string"           validate:"falseif=MissingField"`
	FieldArray             []int   `json:"field_array"              validate:"required"`
	FieldInt               int     `json:"field_int"                validate:"required"`
	FieldUint              uint    `json:"field_uint"               validate:"required"`
	FieldFloat             float32 `json:"field_float"              validate:"required"`
	FieldBool              bool    `json:"field_bool"               validate:"required"`
	FieldInterface         any     `json:"field_interface"`
	FalseIfEmpty           string  `json:"falseif_empty"            validate:"falseif"`
	FalseIfArray           string  `json:"falseif_array"            validate:"falseif=FieldArray 3|alpha"`
	FalseIfInt             string  `json:"falseif_int"              validate:"falseif=FieldInt -123|alpha"`
	FalseIfUint            string  `json:"falseif_uint"             validate:"falseif=FieldUint 123|alpha"`
	FalseIfFloat           string  `json:"falseif_float"            validate:"falseif=FieldFloat 1.23|alpha"`
	FalseIfBool            string  `json:"falseif_bool"             validate:"falseif=FieldBool true|alpha"`
	FalseIfReqArray        string  `json:"falseif_req_array"        validate:"falseif=FieldArray|alpha"`
	FalseIfInterface       string  `json:"falseif_interface"        validate:"falseif=FieldInterface 1|alpha"`
	FieldOrTest            string  `json:"field_or_test"            validate:"max=3|alpha"`
	DatetimeRFC3339        string  `json:"datetime_rfc3339"         validate:"datetime_rfc3339"`
	DatetimeRFC3339Relaxed string  `json:"datetime_rfc3339_relaxed" validate:"datetime_rfc3339_relaxed"`
}

func getTestCustomTagData() testCustomTagStruct {
	return testCustomTagStruct{
		E164NoPlus:             "123456789012345",
		EIN:                    "12-3456789",
		USZIPCode:              "12345",
		USZIPCodeB:             "12345-1234",
		Country:                "US",
		State:                  "NY",
		StateB:                 "AL",
		StateC:                 "WI",
		StateD:                 "AK",
		StateE:                 "VI",
		FalseIfMissing:         "hello",
		FieldArray:             []int{1, 2, 3},
		FieldInt:               -123,
		FieldUint:              123,
		FieldFloat:             1.23,
		FieldBool:              true,
		FalseIfEmpty:           "X",
		FalseIfArray:           "A",
		FalseIfInt:             "B",
		FalseIfUint:            "C",
		FalseIfFloat:           "D",
		FalseIfBool:            "E",
		FalseIfReqArray:        "F",
		FalseIfInterface:       "G",
		FieldOrTest:            "123",
		DatetimeRFC3339:        "2023-07-08T12:34:56Z",
		DatetimeRFC3339Relaxed: "2023-07-08 12:34:56",
	}
}

func TestCustomTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fobj         func(obj testCustomTagStruct) testCustomTagStruct
		wantErr      bool
		wantErrCount int
	}{
		{
			name:         "success",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { return obj },
			wantErr:      false,
			wantErrCount: 0,
		},
		{
			name:         "fail with invalid e164noplus",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.E164NoPlus = "+123456789012345"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid ein",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.EIN = "12-345-56789"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid zip code",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.USZIPCode = "1234"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid US state",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.State = "XX"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid required US state",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.StateB = "XX"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid US state when country is not set",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.Country = ""; obj.StateB = "XX"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with invalid US territory",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.StateE = "XX"; return obj },
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name: "pass with non US state",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.Country = "GB"
				obj.StateC = "England"

				return obj
			},
			wantErr:      false,
			wantErrCount: 0,
		},
		{
			name: "pass with US state and non-US country",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.Country = "GB"
				obj.StateC = "NY"

				return obj
			},
			wantErr:      false,
			wantErrCount: 0,
		},
		{
			name:         "fail with or tags",
			fobj:         func(obj testCustomTagStruct) testCustomTagStruct { obj.FieldOrTest = "1234"; return obj },
			wantErr:      true,
			wantErrCount: 2,
		},
		{
			name: "fail with invalid RFC-3339 datetime",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.DatetimeRFC3339 = "2023-07-08 12:34:56"
				return obj
			},
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name: "fail with invalid RFC-3339 relaxed datetime",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.DatetimeRFC3339Relaxed = "2023-07-08_12:34:56"
				return obj
			},
			wantErr:      true,
			wantErrCount: 1,
		},
	}
	opts := []Option{
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags()),
		WithErrorTemplates(ErrorTemplates()),
	}
	v, err := New(opts...)
	require.NoError(t, err, "New() unexpected error = %v", err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := tt.fobj(getTestCustomTagData())
			err := v.ValidateStruct(s)

			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)

			errs := errutil.Errors(err)

			require.Len(t, errs, tt.wantErrCount, "errors: %+v", errs)
		})
	}
}

// TestCustomTags_falseifMultiWordValue ensures that a "falseif" comparison
// value containing spaces (e.g. "New York") is matched in full rather than
// truncated at the first space.
func TestCustomTags_falseifMultiWordValue(t *testing.T) {
	t.Parallel()

	// City is validated as a US state only when State equals the full
	// multi-word value "New York".
	type multiWordStruct struct {
		State string `json:"state"`
		City  string `json:"city"  validate:"falseif=State New York|usstate"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags()),
		WithErrorTemplates(ErrorTemplates()),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	// State fully matches "New York": the usstate check applies and fails.
	err = v.ValidateStruct(multiWordStruct{State: "New York", City: "XX"})
	require.Error(t, err, "usstate check must run when State equals the full value")

	// A value that only matches the truncated "New" must NOT trigger the
	// usstate check (proving the value is compared in full).
	err = v.ValidateStruct(multiWordStruct{State: "New", City: "XX"})
	require.NoError(t, err, "usstate check must be skipped when State differs from the full value")
}

// TestCustomTags_falseifNonComparable ensures that "falseif" referencing a struct field
// of a non-comparable type (containing a slice) does not panic and correctly detects
// zero values via IsZero.
func TestCustomTags_falseifNonComparable(t *testing.T) {
	t.Parallel()

	type meta struct {
		Tags []string
	}

	type ncStruct struct {
		Meta  meta   `json:"meta"`
		State string `json:"state" validate:"falseif=Meta|usstate"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags()),
		WithErrorTemplates(ErrorTemplates()),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	// Meta is zero: the usstate check is skipped, so an invalid state passes.
	err = v.ValidateStruct(ncStruct{State: "ZZ"})
	require.NoError(t, err, "usstate check must be skipped when Meta is zero")

	// Meta is set: the usstate check applies and fails for an invalid state.
	err = v.ValidateStruct(ncStruct{Meta: meta{Tags: []string{"x"}}, State: "ZZ"})
	require.Error(t, err, "usstate check must run when Meta is set")

	// Meta is set and the state is valid.
	err = v.ValidateStruct(ncStruct{Meta: meta{Tags: []string{"x"}}, State: "NY"})
	require.NoError(t, err, "valid state must pass when Meta is set")
}

// TestCustomTags_falseifFloat32Param ensures that "falseif" params for float32 fields
// are compared at float32 precision, so values such as "0.1" (not exactly representable
// in binary floating point) still match the stored field value.
func TestCustomTags_falseifFloat32Param(t *testing.T) {
	t.Parallel()

	type f32Struct struct {
		Ratio float32 `json:"ratio"`
		State string  `json:"state" validate:"falseif=Ratio 0.1|usstate"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags()),
		WithErrorTemplates(ErrorTemplates()),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	// Ratio equals the param at float32 precision: the usstate check applies and fails.
	err = v.ValidateStruct(f32Struct{Ratio: 0.1, State: "ZZ"})
	require.Error(t, err, "usstate check must run when Ratio matches the falseif param")

	// Ratio differs from the param: the usstate check is skipped.
	err = v.ValidateStruct(f32Struct{Ratio: 0.2, State: "ZZ"})
	require.NoError(t, err, "usstate check must be skipped when Ratio differs from the falseif param")

	// Ratio matches and the state is valid.
	err = v.ValidateStruct(f32Struct{Ratio: 0.1, State: "NY"})
	require.NoError(t, err, "valid state must pass when Ratio matches the falseif param")
}

func Test_hasDefaultValue_invalid(t *testing.T) {
	t.Parallel()

	var i any

	vi := reflect.ValueOf(i)
	got := hasDefaultValue(vi, vi.Kind())
	require.True(t, got, "Expecting true value")
}

func Test_hasNotValue_float32(t *testing.T) {
	t.Parallel()

	v := reflect.ValueOf(float32(0.1))

	require.False(t, hasNotValue(v, reflect.Float32, "0.1"))
	require.True(t, hasNotValue(v, reflect.Float32, "0.2"))
	require.True(t, hasNotValue(v, reflect.Float32, "not-a-number"))
}

func Test_hasNotValue_float64(t *testing.T) {
	t.Parallel()

	v := reflect.ValueOf(float64(0.1))

	require.False(t, hasNotValue(v, reflect.Float64, "0.1"))
	require.True(t, hasNotValue(v, reflect.Float64, "0.2"))
	require.True(t, hasNotValue(v, reflect.Float64, "not-a-number"))
}

// TestCustomTags_nonStringField ensures the string-based custom validators
// reject non-string fields instead of matching them or panicking.
func TestCustomTags_nonStringField(t *testing.T) {
	t.Parallel()

	type intFieldStruct struct {
		E164 int `json:"e164" validate:"e164noplus"`
		Zip  int `json:"zip"  validate:"zipcode"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags()),
		WithErrorTemplates(ErrorTemplates()),
	)
	require.NoError(t, err)

	err = v.ValidateStruct(intFieldStruct{E164: 123456789012345, Zip: 12345})
	require.Error(t, err, "non-string fields must fail the string-based custom validators")
	require.Len(t, errutil.Errors(err), 2, "both custom validators must reject the int fields")
}

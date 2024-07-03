package validator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

type testCustomTagStruct struct {
	E164NoPlus       string  `json:"e164_no_plus"      validate:"e164noplus"`
	EIN              string  `json:"ein"               validate:"ein"`
	EINB             string  `json:"ein_b"             validate:"ein"`
	USZIPCode        string  `json:"zip"               validate:"zipcode"`
	USZIPCodeB       string  `json:"zip_b"             validate:"zipcode"`
	Country          string  `json:"country"           validate:"iso3166_1_alpha2"`
	State            string  `json:"state"             validate:"usstate"`
	StateB           string  `json:"state_b"           validate:"falseif=Country|usstate"`
	StateC           string  `json:"state_c"           validate:"falseif=Country US|usstate"`
	StateD           string  `json:"state_d"           validate:"falseif|usstate"`
	StateE           string  `json:"state_e"           validate:"falseif=Country US|usterritory"`
	FalseIfMissing   string  `json:"falseif_string"    validate:"falseif=MissingField"`
	FieldArray       []int   `json:"field_array"       validate:"required"`
	FieldInt         int     `json:"field_int"         validate:"required"`
	FieldUint        uint    `json:"field_uint"        validate:"required"`
	FieldFloat       float32 `json:"field_float"       validate:"required"`
	FieldBool        bool    `json:"field_bool"        validate:"required"`
	FieldInterface   any     `json:"field_interface"`
	FalseIfEmpty     string  `json:"falseif_empty"     validate:"falseif"`
	FalseIfArray     string  `json:"falseif_array"     validate:"falseif=FieldArray 3|alpha"`
	FalseIfInt       string  `json:"falseif_int"       validate:"falseif=FieldInt -123|alpha"`
	FalseIfUint      string  `json:"falseif_uint"      validate:"falseif=FieldUint 123|alpha"`
	FalseIfFloat     string  `json:"falseif_float"     validate:"falseif=FieldFloat 1.23|alpha"`
	FalseIfBool      string  `json:"falseif_bool"      validate:"falseif=FieldBool true|alpha"`
	FalseIfReqArray  string  `json:"falseif_req_array" validate:"falseif=FieldArray|alpha"`
	FalseIfInterface string  `json:"falseif_interface" validate:"falseif=FieldInterface 1|alpha"`
	FieldOrTest      string  `json:"field_or_test"     validate:"max=3|alpha"`
}

func getTestCustomTagData() testCustomTagStruct {
	return testCustomTagStruct{
		E164NoPlus:       "123456789012345",
		EIN:              "12-3456789",
		EINB:             "123456789",
		USZIPCode:        "12345",
		USZIPCodeB:       "12345-1234",
		Country:          "US",
		State:            "NY",
		StateB:           "AL",
		StateC:           "WI",
		StateD:           "AK",
		StateE:           "VI",
		FalseIfMissing:   "hello",
		FieldArray:       []int{1, 2, 3},
		FieldInt:         -123,
		FieldUint:        123,
		FieldFloat:       1.23,
		FieldBool:        true,
		FalseIfEmpty:     "X",
		FalseIfArray:     "A",
		FalseIfInt:       "B",
		FalseIfUint:      "C",
		FalseIfFloat:     "D",
		FalseIfBool:      "E",
		FalseIfReqArray:  "F",
		FalseIfInterface: "G",
		FieldOrTest:      "123",
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

			errs := multierr.Errors(err)

			require.Len(t, errs, tt.wantErrCount, "errors: %+v", errs)
		})
	}
}

func Test_hasDefaultValue_invalid(t *testing.T) {
	t.Parallel()

	var i any

	vi := reflect.ValueOf(i)
	got := hasDefaultValue(vi, vi.Kind(), true)
	require.True(t, got, "Expecting true value")
}

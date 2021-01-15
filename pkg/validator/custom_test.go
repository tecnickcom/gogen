package validator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCustomTagStruct struct {
	E164             string      `json:"e164" validate:"e164"`
	E164B            string      `json:"e164_b" validate:"e164"`
	EIN              string      `json:"ein" validate:"ein"`
	EINB             string      `json:"ein_b" validate:"ein"`
	USZIPCode        string      `json:"zip" validate:"zipcode"`
	USZIPCodeB       string      `json:"zip_b" validate:"zipcode"`
	Country          string      `json:"country" validate:"iso3166_1_alpha2"`
	State            string      `json:"state" validate:"usstate"`
	StateB           string      `json:"state_b" validate:"falseif=Country|usstate"`
	StateC           string      `json:"state_c" validate:"falseif=Country US|usstate"`
	StateD           string      `json:"state_d" validate:"falseif|usstate"`
	FalseIfMissing   string      `json:"falseif_string" validate:"falseif=MissingField"`
	FieldArray       []int       `json:"field_array" validate:"required"`
	FieldInt         int         `json:"field_int" validate:"required"`
	FieldUint        uint        `json:"field_uint" validate:"required"`
	FieldFloat       float32     `json:"field_float" validate:"required"`
	FieldBool        bool        `json:"field_bool" validate:"required"`
	FieldInterface   interface{} `json:"field_interface"`
	FalseIfEmpty     string      `json:"falseif_empty" validate:"falseif"`
	FalseIfArray     string      `json:"falseif_array" validate:"falseif=FieldArray 3|alpha"`
	FalseIfInt       string      `json:"falseif_int" validate:"falseif=FieldInt -123|alpha"`
	FalseIfUint      string      `json:"falseif_uint" validate:"falseif=FieldUint 123|alpha"`
	FalseIfFloat     string      `json:"falseif_float" validate:"falseif=FieldFloat 1.23|alpha"`
	FalseIfBool      string      `json:"falseif_bool" validate:"falseif=FieldBool true|alpha"`
	FalseIfReqArray  string      `json:"falseif_req_array" validate:"falseif=FieldArray|alpha"`
	FalseIfInterface string      `json:"falseif_interface" validate:"falseif=FieldInterface 1|alpha"`
}

func getTestCustomTagData() testCustomTagStruct {
	return testCustomTagStruct{
		E164:             "+123456789012345",
		E164B:            "123456789012345",
		EIN:              "12-3456789",
		EINB:             "123456789",
		USZIPCode:        "12345",
		USZIPCodeB:       "12345-1234",
		Country:          "US",
		State:            "NY",
		StateB:           "AL",
		StateC:           "WI",
		StateD:           "AK",
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
	}
}

func TestCustomTags(t *testing.T) {
	tests := []struct {
		name    string
		fobj    func(obj testCustomTagStruct) testCustomTagStruct
		wantErr bool
	}{
		{
			name:    "success",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { return obj },
			wantErr: false,
		},
		{
			name:    "fail with invalid e164",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.E164 = "012345678"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid ein",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.EIN = "12-345-56789"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid zip code",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.USZIPCode = "1234"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid US state",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.State = "XX"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid US state when country is not set",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.Country = ""; obj.StateB = "XX"; return obj },
			wantErr: true,
		},
		{
			name: "pass with non US state",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.Country = "GB"
				obj.StateC = "England"
				return obj
			},
			wantErr: false,
		},
		{
			name: "pass with US state and non-US country",
			fobj: func(obj testCustomTagStruct) testCustomTagStruct {
				obj.Country = "GB"
				obj.StateC = "NY"
				return obj
			},
			wantErr: false,
		},
	}
	opts := []Option{
		WithFieldNameTag("json"),
		WithCustomValidationTags(CustomValidationTags),
		WithErrorTemplates(ErrorTemplates),
	}
	v, err := New(opts...)
	require.NoError(t, err, "New() unexpected error = %v", err)
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.fobj(getTestCustomTagData())
			err := v.ValidateStruct(s)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

func Test_hasDefaultValue_invalid(t *testing.T) {
	var i interface{}
	vi := reflect.ValueOf(i)
	t.Parallel()
	got := hasDefaultValue(vi, vi.Kind(), true)
	require.True(t, got, "Expecting true value")
}

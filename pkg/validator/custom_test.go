package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testCustomTagStruct struct {
	E164FieldA      string `json:"e164_field_a" validate:"e164"`
	E164FieldB      string `json:"e164_field_b" validate:"e164"`
	EINFieldA       string `json:"ein_field_a" validate:"ein"`
	EINFieldB       string `json:"ein_field_b" validate:"ein"`
	USZIPCodeFieldA string `json:"zip_field_a" validate:"zipcode"`
	USZIPCodeFieldB string `json:"zip_field_b" validate:"zipcode"`
	USStateField    string `json:"state_field" validate:"usstate"`
}

func getTestCustomTagData() testCustomTagStruct {
	return testCustomTagStruct{
		E164FieldA:      "+123456789012345",
		E164FieldB:      "123456789012345",
		EINFieldA:       "12-3456789",
		EINFieldB:       "123456789",
		USZIPCodeFieldA: "12345",
		USZIPCodeFieldB: "12345-1234",
		USStateField:    "NY",
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
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.E164FieldA = "012345678"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid ein",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.EINFieldA = "12-345-56789"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid zip code",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.USZIPCodeFieldA = "1234"; return obj },
			wantErr: true,
		},
		{
			name:    "fail with invalid US state",
			fobj:    func(obj testCustomTagStruct) testCustomTagStruct { obj.USStateField = "XX"; return obj },
			wantErr: true,
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
			require.Equal(t, tt.wantErr, err != nil, "ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	t.Parallel()
	want := "mock_error"
	e := &ValidationError{Err: "mock_error"}
	got := e.Error()
	require.Equal(t, want, got, "Error() = %v, want %v", got, want)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		want    bool
		wantErr bool
	}{
		{
			name:    "empty option success",
			opts:    nil,
			want:    true,
			wantErr: false,
		},
		{
			name: "applied opts returns error",
			opts: []Option{
				WithDefaultTranslations(),
				WithFieldNameTag("test_tag"),
				WithValidationTranslated("test_tag", nil, nil, nil),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "success with opts applied",
			opts: []Option{
				WithDefaultTranslations(),
				WithFieldNameTag("test_tag"),
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.opts...)

			if tt.wantErr {
				require.Nil(t, got, "New() returned Validator should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.NotNil(t, got, "New() returned Validator should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestValidator_ValidateStruct(t *testing.T) {
	type subStruct struct {
		URLField string `json:"sub_string" validate:"required,url"`
		IntField int    `json:"sub_int" validate:"required,min=2"`
	}
	type rootStruct struct {
		BoolField    bool       `json:"bool_field" validate:"required"`
		SubStruct    subStruct  `json:"sub_struct" validate:"required"`
		SubStructPtr *subStruct `json:"sub_struct_ptr" validate:"required"`
		StringField  string     `json:"string_field" validate:"required"`
		NoNameField  string     `json:"-" validate:"required"`
	}

	tests := []struct {
		name    string
		obj     interface{}
		wantErr bool
	}{
		{
			name: "success struct validated",
			obj: rootStruct{
				BoolField: true,
				SubStruct: subStruct{
					URLField: "http://first.test.invalid",
					IntField: 3,
				},
				SubStructPtr: &subStruct{
					URLField: "http://second.test.invalid",
					IntField: 123,
				},
				StringField: "hello world",
				NoNameField: "test",
			},
			wantErr: false,
		},
		{
			name:    "failed in validation with empty struct",
			obj:     rootStruct{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := []Option{
				WithFieldNameTag("json"),
				WithDefaultTranslations(),
			}
			v, _ := New(opts...)
			err := v.ValidateStruct(tt.obj)
			require.Equal(t, tt.wantErr, err != nil, "ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

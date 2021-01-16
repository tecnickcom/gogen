package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestError_Error(t *testing.T) {
	t.Parallel()
	want := "mock_error"
	e := &Error{Err: "mock_error"}
	got := e.Error()
	require.Equal(t, want, got, "Error() = %v, want %v", got, want)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "success with empty options",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "success with custom tag name option",
			opts: []Option{
				WithFieldNameTag("test_tag"),
			},
			wantErr: false,
		},
		{
			name: "success with custom tag name and error templates options",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr: false,
		},
		{
			name: "success with custom tag name, custom validation and error templates options",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithCustomValidationTags(CustomValidationTags),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr: false,
		},
		{
			name: "fail with invalid error template",
			opts: []Option{
				WithErrorTemplates(map[string]string{"error": "{{.ERROR} ---"}),
			},
			wantErr: true,
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
		BoolField    bool       `json:"bool_field"`
		SubStruct    subStruct  `json:"sub_struct" validate:"required"`
		SubStructPtr *subStruct `json:"sub_struct_ptr" validate:"required"`
		StringField  string     `json:"string_field" validate:"required"`
		NoNameField  string     `json:"-" validate:"required"`
	}

	validObj := rootStruct{
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
	}

	tests := []struct {
		name         string
		obj          interface{}
		opts         []Option
		wantErr      bool
		wantErrCount int
	}{
		{
			name: "success with custom tag",
			obj:  validObj,
			opts: []Option{
				WithFieldNameTag("json"),
			},
			wantErr:      false,
			wantErrCount: 0,
		},
		{
			name: "success with custom tag name and error templates options",
			obj:  validObj,
			opts: []Option{
				WithFieldNameTag("json"),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr:      false,
			wantErrCount: 0,
		},
		{
			name:         "fail with empty data and no options",
			obj:          rootStruct{},
			opts:         []Option{},
			wantErr:      true,
			wantErrCount: 5,
		},
		{
			name: "fail with empty data error templates",
			obj:  rootStruct{},
			opts: []Option{
				WithFieldNameTag("json"),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr:      true,
			wantErrCount: 5,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := New(tt.opts...)
			require.NoError(t, err, "New() unexpected error = %v", err)
			err = v.ValidateStruct(tt.obj)
			require.Equal(t, tt.wantErr, err != nil, "ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			errs := multierr.Errors(err)
			require.Equal(t, tt.wantErrCount, len(errs), "errors: %+v", errs)
		})
	}
}

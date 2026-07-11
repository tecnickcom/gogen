package validator

import (
	"context"
	"testing"

	vt "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/errutil"
)

func TestError_Error(t *testing.T) {
	t.Parallel()

	want := "mock_error"
	e := &Error{Err: "mock_error"}
	got := e.Error()
	require.Equal(t, want, got, "Error() = %v, want %v", got, want)
}

func TestNew(t *testing.T) {
	t.Parallel()

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
				WithErrorTemplates(ErrorTemplates()),
			},
			wantErr: false,
		},
		{
			name: "success with custom tag name, custom validation and error templates options",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithCustomValidationTags(CustomValidationTags()),
				WithErrorTemplates(ErrorTemplates()),
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
	t.Parallel()

	type subStruct struct {
		URLField string `json:"sub_string" validate:"required,url"`
		IntField int    `json:"sub_int"    validate:"required,min=2"`
	}

	type rootStruct struct {
		BoolField    bool       `json:"bool_field"`
		SubStruct    subStruct  `json:"sub_struct"     validate:"required"`
		SubStructPtr *subStruct `json:"sub_struct_ptr" validate:"required"`
		StringField  string     `json:"string_field"   validate:"required"`
		NoNameField  string     `json:"-"              validate:"required"`
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
		obj          any
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
				WithErrorTemplates(ErrorTemplates()),
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
				WithErrorTemplates(ErrorTemplates()),
			},
			wantErr:      true,
			wantErrCount: 5,
		},
		{
			// Non-validation errors (*vt.InvalidValidationError) must be surfaced.
			name:         "fail with nil object",
			obj:          nil,
			opts:         []Option{},
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with nil struct pointer",
			obj:          (*rootStruct)(nil),
			opts:         []Option{},
			wantErr:      true,
			wantErrCount: 1,
		},
		{
			name:         "fail with non-struct object",
			obj:          42,
			opts:         []Option{},
			wantErr:      true,
			wantErrCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := New(tt.opts...)
			require.NoError(t, err, "New() unexpected error = %v", err)

			err = v.ValidateStruct(tt.obj)
			require.Equal(t, tt.wantErr, err != nil, "ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)

			errs := errutil.Errors(err)
			require.Len(t, errs, tt.wantErrCount, "errors: %+v", errs)
		})
	}
}

// TestValidator_ValidateStruct_noHTMLEscape ensures rendered error messages keep
// special characters (such as '<' and '&') verbatim instead of HTML-escaping
// them, since the messages are returned as plain text (text/template, not
// html/template).
func TestValidator_ValidateStruct_noHTMLEscape(t *testing.T) {
	t.Parallel()

	type escapeStruct struct {
		Choice string `json:"choice" validate:"oneof=A&B C<D"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithErrorTemplates(ErrorTemplates()),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	err = v.ValidateStruct(escapeStruct{Choice: "X"})
	require.Error(t, err, "expected a validation error")

	msg := err.Error()
	require.Contains(t, msg, "A&B C<D", "message must keep raw special characters: %q", msg)
	require.NotContains(t, msg, "&amp;", "message must not be HTML-escaped: %q", msg)
	require.NotContains(t, msg, "&lt;", "message must not be HTML-escaped: %q", msg)
}

// TestValidator_ValidateStruct_falseifPrefix ensures that only the exact
// "falseif" tag is skipped and that look-alike tags such as "falseifoo" are
// still reported as errors.
func TestValidator_ValidateStruct_falseifPrefix(t *testing.T) {
	t.Parallel()

	type prefixStruct struct {
		Field string `json:"field" validate:"falseifoo"`
	}

	v, err := New(
		WithFieldNameTag("json"),
		WithCustomValidationTags(map[string]vt.FuncCtx{
			// always fails so the produced error must NOT be skipped
			"falseifoo": func(_ context.Context, _ vt.FieldLevel) bool { return false },
		}),
	)
	require.NoError(t, err, "New() unexpected error = %v", err)

	err = v.ValidateStruct(prefixStruct{Field: "anything"})
	require.Error(t, err, "falseifoo error must not be skipped")

	errs := errutil.Errors(err)
	require.Len(t, errs, 1, "errors: %+v", errs)
	require.Contains(t, errs[0].Error(), "falseifoo", "error must reference falseifoo: %q", errs[0].Error())
}

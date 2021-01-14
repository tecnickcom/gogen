package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		want    bool
		wantErr bool
	}{
		{
			name:    "success with empty options",
			opts:    nil,
			want:    true,
			wantErr: false,
		},
		{
			name: "applied opts returns error",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithDefaultTranslations(),
				WithValidationTranslated("test_tag", nil, nil, nil),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "success with default translations",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithDefaultTranslations(),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "success with error templates",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithErrorTemplates(ErrorTemplates),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "success with custom tags",
			opts: []Option{
				WithFieldNameTag("test_tag"),
				WithCustomValidationTags(CustomValidationTags),
				WithErrorTemplates(ErrorTemplates),
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
		name    string
		obj     interface{}
		opts    []Option
		wantErr bool
	}{
		{
			name: "success with no translations",
			obj:  validObj,
			opts: []Option{
				WithFieldNameTag("json"),
			},
			wantErr: false,
		},
		{
			name: "success with default error templates",
			obj:  validObj,
			opts: []Option{
				WithFieldNameTag("json"),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr: false,
		},
		{
			name: "success with default translator",
			obj:  validObj,
			opts: []Option{
				WithFieldNameTag("json"),
				WithDefaultTranslations(),
			},
			wantErr: false,
		},
		{
			name:    "fail with no options",
			obj:     rootStruct{},
			opts:    []Option{},
			wantErr: true,
		},
		{
			name: "fail with error templates",
			obj:  rootStruct{},
			opts: []Option{
				WithFieldNameTag("json"),
				WithErrorTemplates(ErrorTemplates),
			},
			wantErr: true,
		},
		{
			name: "fail with default translations",
			obj:  rootStruct{},
			opts: []Option{
				WithFieldNameTag("json"),
				WithDefaultTranslations(),
			},
			wantErr: true,
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
		})
	}
}

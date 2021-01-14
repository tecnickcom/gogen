package validator

import (
	"fmt"
	"testing"

	lc "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	vt "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestWithFieldNameTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{
			name: "tag is empty string",
			tag:  "",
		},
		{
			name: "success return name",
			tag:  "abcderfghijk",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{
				V: vt.New(),
			}
			err := WithFieldNameTag(tt.tag)(v)
			require.NoError(t, err)
		})
	}
}

func TestWithCustomValidationTags(t *testing.T) {
	tests := []struct {
		name    string
		arg     map[string]vt.Func
		wantErr bool
	}{
		{
			name:    "success with default custom tags",
			arg:     CustomValidationTags,
			wantErr: false,
		},
		{
			name:    "success with empty tags",
			arg:     map[string]vt.Func{},
			wantErr: false,
		},
		{
			name:    "error with invalid tag",
			arg:     map[string]vt.Func{"error": nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{V: vt.New()}
			err := WithCustomValidationTags(tt.arg)(v)
			if tt.wantErr {
				require.Error(t, err, "error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "unexpected error = %v", err)
			}
		})
	}
}

func TestWithErrorTemplates(t *testing.T) {
	tests := []struct {
		name    string
		arg     map[string]string
		wantErr bool
	}{
		{
			name:    "success with default templates",
			arg:     ErrorTemplates,
			wantErr: false,
		},
		{
			name:    "success with one templates",
			arg:     map[string]string{"test": "field {{.Tag}}"},
			wantErr: false,
		},
		{
			name:    "success with empty template",
			arg:     map[string]string{},
			wantErr: false,
		},
		{
			name:    "error with invalid template",
			arg:     map[string]string{"test": "{{.Something} missing closing curly brace"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{V: vt.New()}
			err := WithErrorTemplates(tt.arg)(v)
			if tt.wantErr {
				require.Error(t, err, "error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "unexpected error = %v", err)
				require.Equal(t, len(tt.arg), len(v.tpl), "Not all templates were imported")
			}
		})
	}
}

func TestWithDefaultTranslations(t *testing.T) {
	t.Parallel()
	v := &Validator{V: vt.New()}
	err := WithDefaultTranslations()(v)
	require.NoError(t, err)
}

func TestWithValidationTranslated(t *testing.T) {
	type args struct {
		tag           string
		fn            vt.Func
		registerFn    vt.RegisterTranslationsFunc
		translationFn vt.TranslationFunc
	}
	tests := []struct {
		name    string
		args    args
		want    Option
		nilT    bool
		wantErr bool
	}{
		{
			name: "validation registration failure",
			args: args{
				tag: "test_tag",
				fn:  nil,
			},
			nilT:    false,
			wantErr: true,
		},
		{
			name: "translation registration failure",
			args: args{
				tag: "test_tag",
				fn: func(fl vt.FieldLevel) bool {
					return true
				},
				registerFn: func(ut ut.Translator) error {
					return fmt.Errorf("registration error")
				},
			},
			nilT:    false,
			wantErr: true,
		},
		{
			name: "nil translator",
			args: args{
				tag: "test_tag",
				fn: func(fl vt.FieldLevel) bool {
					return true
				},
				registerFn: func(ut ut.Translator) error {
					return nil
				},
				translationFn: func(ut ut.Translator, fe vt.FieldError) string {
					return ""
				},
			},
			nilT:    true,
			wantErr: true,
		},
		{
			name: "success case",
			args: args{
				tag: "test_tag",
				fn: func(fl vt.FieldLevel) bool {
					return true
				},
				registerFn: func(ut ut.Translator) error {
					return nil
				},
				translationFn: func(ut ut.Translator, fe vt.FieldError) string {
					return ""
				},
			},
			nilT:    false,
			wantErr: false,
		},
	}
	en := lc.New()
	uni := ut.New(en, en)
	trans, ok := uni.GetTranslator("en")
	require.True(t, ok, "failed while creating the translator")
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{
				V: vt.New(),
				T: trans,
			}
			if tt.nilT {
				v.T = nil
			}
			err := WithValidationTranslated(tt.args.tag, tt.args.fn, tt.args.registerFn, tt.args.translationFn)(v)
			if tt.wantErr {
				require.Error(t, err, "error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "unexpected error = %v", err)
			}
		})
	}
}

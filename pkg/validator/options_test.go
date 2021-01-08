package validator

import (
	"fmt"
	"testing"

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
		wantErr bool
	}{
		{
			name: "validation registration failure",
			args: args{
				tag: "test_tag",
				fn:  nil,
			},
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := &Validator{
				V: vt.New(),
			}
			err := WithValidationTranslated(tt.args.tag, tt.args.fn, tt.args.registerFn, tt.args.translationFn)(v)
			if tt.wantErr {
				require.Error(t, err, "WithValidationTranslated() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "WithValidationTranslated() unexpected error = %v", err)
			}
		})
	}
}

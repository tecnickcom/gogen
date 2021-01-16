package validator

import (
	"testing"

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
			v := &Validator{v: vt.New()}
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
			v := &Validator{v: vt.New()}
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
			v := &Validator{v: vt.New()}
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

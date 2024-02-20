package validator

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"

	vt "github.com/go-playground/validator/v10"
)

// Option is the interface that allows to set configuration options.
type Option func(v *Validator) error

// WithFieldNameTag allows to use the field names specified by the tag instead of the original struct names.
func WithFieldNameTag(tag string) Option {
	return func(v *Validator) error {
		if tag == "" {
			return nil
		}

		v.v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}

			return name
		})

		return nil
	}
}

// WithCustomValidationTags register custom tags and validation functions.
func WithCustomValidationTags(t map[string]vt.FuncCtx) Option {
	return func(v *Validator) error {
		for tag, fn := range t {
			if err := v.v.RegisterValidationCtx(tag, fn); err != nil {
				return fmt.Errorf("failed registering custom tag: %w", err)
			}
		}

		return nil
	}
}

// WithCustomTypeFunc registers a CustomTypeFunc against a number of types.
func WithCustomTypeFunc(fn vt.CustomTypeFunc, types ...any) Option {
	return func(v *Validator) error {
		v.v.RegisterCustomTypeFunc(fn, types...)
		return nil
	}
}

// WithErrorTemplates sets basic template-based error message translations.
// The argument t maps tags to html templates that uses the Error data.
// These translations takes precedence over the parent library translation object.
func WithErrorTemplates(t map[string]string) Option {
	return func(v *Validator) error {
		if len(v.tpl) == 0 {
			v.tpl = make(map[string]*template.Template, len(t))
		}

		for tag, tpl := range t {
			t, err := template.New(tag).Parse(tpl)
			if err != nil {
				return fmt.Errorf("failed adding error template: %w", err)
			}

			v.tpl[tag] = t
		}

		return nil
	}
}

package validator

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"

	lc "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	vt "github.com/go-playground/validator/v10"
	tr "github.com/go-playground/validator/v10/translations/en"
)

// Option is the interface that allows to set options.
type Option func(v *Validator) error

// WithFieldNameTag allows to use the field names specified by the tag instead of the original struct names.
func WithFieldNameTag(tag string) Option {
	return func(v *Validator) error {
		if tag == "" {
			return nil
		}
		v.V.RegisterTagNameFunc(func(fld reflect.StructField) string {
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
func WithCustomValidationTags(t map[string]vt.Func) Option {
	return func(v *Validator) error {
		for tag, fn := range t {
			if err := v.V.RegisterValidation(tag, fn); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithErrorTemplates sets basic template-based error message translations.
// The argument t maps tags to html templates that uses the Error data.
// These translations takes precedence over the parent library translation object.
func WithErrorTemplates(t map[string]string) Option {
	return func(v *Validator) error {
		v.tpl = make(map[string]*template.Template, len(t))
		for tag, tpl := range t {
			t, err := template.New(tag).Parse(tpl)
			if err != nil {
				return err
			}
			v.tpl[tag] = t
		}
		return nil
	}
}

// WithDefaultTranslations sets the default English translations using the parent library translator.
func WithDefaultTranslations() Option {
	return func(v *Validator) error {
		en := lc.New()
		uni := ut.New(en, en)
		trans, ok := uni.GetTranslator("en")
		if ok {
			_ = tr.RegisterDefaultTranslations(v.V, trans)
			v.T = trans
		}
		return nil
	}
}

// WithValidationTranslated allows to register a validation func and a translation for the provided tag.
func WithValidationTranslated(tag string, fn vt.Func, registerFn vt.RegisterTranslationsFunc, translationFn vt.TranslationFunc) Option {
	return func(v *Validator) error {
		if err := v.V.RegisterValidation(tag, fn); err != nil {
			return err
		}
		if v.T == nil {
			return fmt.Errorf("the Translator object is nil")
		}
		if err := v.V.RegisterTranslation(tag, v.T, registerFn, translationFn); err != nil {
			return err
		}
		return nil
	}
}

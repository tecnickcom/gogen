// Package validator expose wrapper function for https://github.com/go-playground/validator
// to provide value validations for structs and individual fields based on tags.
package validator

import (
	"bytes"
	"html/template"
	"strings"

	ut "github.com/go-playground/universal-translator"
	vt "github.com/go-playground/validator/v10"
	"go.uber.org/multierr"
)

// Validator contains the validator object fields.
type Validator struct {
	// V is the validate object.
	V *vt.Validate

	// Trans is the translator object used by the parent library.
	T ut.Translator

	// tpl contains the map of basic translation templates indexed by tag.
	tpl map[string]*template.Template
}

// New returns a new validator with the specified options.
func New(opts ...Option) (*Validator, error) {
	v := &Validator{
		V: vt.New(),
	}
	for _, applyOpt := range opts {
		if err := applyOpt(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// ValidateStruct validates the structure fields tagged with "validate".
// nolint: gocognit,gocyclo
func (v *Validator) ValidateStruct(obj interface{}) (err error) {
	vErr := v.V.Struct(obj)
	if vErr == nil {
		return nil
	}
	for _, e := range vErr.(vt.ValidationErrors) {
		if e != nil {
			tags := strings.Split(e.Tag(), "|")
			for _, tag := range tags {
				if strings.HasPrefix(tag, "falseif") {
					continue
				}
				tagParts := strings.SplitN(tag, "=", 2)
				tagKey := tagParts[0]
				var tagParam string
				if len(tagParts) == 2 {
					tagParam = tagParts[1]
				} else {
					tagParam = e.Param()
				}
				ve := &Error{
					Tag:             tagKey,
					ActualTag:       e.ActualTag(),
					Namespace:       e.Namespace(),
					StructNamespace: e.StructNamespace(),
					Field:           e.Field(),
					StructField:     e.StructField(),
					Value:           e.Value(),
					Param:           tagParam,
					Kind:            e.Kind().String(),
					Type:            e.Type().String(),
					OrigErr:         e.Error(),
				}
				ve.Err = v.translate(e, ve)
				err = multierr.Append(err, ve)
			}
		}
	}
	return err
}

func (v *Validator) translate(fe vt.FieldError, ve *Error) string {
	t, ok := v.tpl[ve.Tag]
	if ok {
		if idx := strings.Index(ve.Namespace, "."); idx != -1 {
			ve.Namespace = ve.Namespace[idx+1:] // remove root struct name
		}
		var out bytes.Buffer
		if err := t.Execute(&out, ve); err == nil {
			return out.String()
		}
	}
	if v.T != nil {
		return fe.Translate(v.T)
	}
	return ve.OrigErr
}

// Package validator wraps https://github.com/go-playground/validator
// to provide value validations for structs and individual fields based on tags.
// It includes a simpler custom translation mechanism for errors.
package validator

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"

	vt "github.com/go-playground/validator/v10"
	"go.uber.org/multierr"
)

// Validator contains the validator object fields.
type Validator struct {
	// V is the validate object.
	v *vt.Validate

	// tpl contains the map of basic translation templates indexed by tag.
	tpl map[string]*template.Template
}

// New returns a new validator with the specified options.
func New(opts ...Option) (*Validator, error) {
	v := &Validator{v: vt.New()}
	for _, applyOpt := range opts {
		if err := applyOpt(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// ValidateStruct validates the structure fields tagged with "validate" and returns a multierror.
func (v *Validator) ValidateStruct(obj interface{}) (err error) {
	return v.ValidateStructCtx(context.Background(), obj)
}

// ValidateStructCtx validates the structure fields tagged with "validate" and returns a multierror.
func (v *Validator) ValidateStructCtx(ctx context.Context, obj interface{}) (err error) {
	vErr := v.v.StructCtx(ctx, obj)
	if vErr == nil {
		return nil
	}
	for _, fe := range vErr.(vt.ValidationErrors) {
		// separate tags grouped by OR
		tags := strings.Split(fe.Tag(), "|")
		for _, tag := range tags {
			if strings.HasPrefix(tag, "falseif") {
				// the "falseif" tag only works in combination with other tags
				continue
			}
			err = multierr.Append(err, v.tagError(fe, tag))
		}
	}
	return err
}

// tagError set the error message associated with the validation tag.
func (v *Validator) tagError(fe vt.FieldError, tag string) (err error) {
	tagParts := strings.SplitN(tag, "=", 2)
	tagKey := tagParts[0]
	tagParam := fe.Param()
	if len(tagParts) == 2 {
		tagParam = tagParts[1]
	}
	namespace := fe.Namespace()
	if idx := strings.Index(namespace, "."); idx != -1 {
		namespace = namespace[idx+1:] // remove root struct name
	}
	ve := &Error{
		Tag:             tagKey,
		Param:           tagParam,
		FullTag:         tag,
		Namespace:       namespace,
		StructNamespace: fe.StructNamespace(),
		Field:           fe.Field(),
		StructField:     fe.StructField(),
		Kind:            fe.Kind().String(),
		Value:           fe.Value(),
	}
	ve.Err = v.translate(ve)
	return ve
}

// translate returns the error message associated with the tag.
func (v *Validator) translate(ve *Error) string {
	if t, ok := v.tpl[ve.Tag]; ok {
		var out bytes.Buffer
		if err := t.Execute(&out, ve); err == nil {
			return out.String()
		}
	}
	return fmt.Sprintf("%s is invalid because fails the rule: '%s'", ve.Namespace, ve.FullTag)
}

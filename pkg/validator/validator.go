/*
Package validator solves two recurring problems with struct validation in Go
services: the gap in domain-specific validation rules missing from general
libraries, and the cost of turning raw validation errors into user-readable
messages. It wraps https://github.com/go-playground/validator — the de-facto
standard validation library — and layers a curated set of custom rules,
a template-based error translation engine, and a clean functional-options API
on top.

# Problem

go-playground/validator is comprehensive but generic. Production services
typically need a handful of domain rules (E.164 phone numbers, US EINs, ZIP
codes, RFC-3339 dates) that are not in the upstream library, and they need
validation errors that read as human sentences rather than raw tag names.
Wiring those two concerns together for every service is boilerplate this
package eliminates.

# How It Works

[New] creates a [Validator] using a variadic list of [Option] values:

 1. An upstream vt.Validate instance is created and held internally.
 2. Options register field-name tag aliases, custom validation functions,
    custom type functions, and error message templates on that instance.
 3. [Validator.ValidateStruct] (or its context-aware twin
    [Validator.ValidateStructCtx]) runs the upstream validator and transforms
    any [vt.ValidationErrors] into a [go.uber.org/multierr] aggregate of
    typed [Error] values, each carrying the failed tag, parameter, namespace,
    field name, kind, and the translated human-readable message.

Tag groups joined by "|" are iterated individually so each OR-branch failure
produces an independent, meaningful error.

# Key Features

  - Human-readable errors: [WithErrorTemplates] maps any validation tag to an
    html/template string. Templates receive an [Error] value, giving access to
    the namespace, field name, tag, parameter, kind, and actual value.
    [ErrorTemplates] returns a ready-to-use map covering every built-in
    go-playground/validator tag.
  - Custom domain rules: [CustomValidationTags] registers eight production-ready
    validators not in the upstream library:
  - falseif — conditional negation combinator
  - e164noplus — E.164 phone number without the leading '+'
  - ein — US Employer Identification Number (12-3456789 or 123456789)
  - zipcode — US ZIP code (12345 or 12345-6789)
  - usstate — two-letter US state code (including DC)
  - usterritory — two-letter US territory code (AS, GU, MP, PR, VI)
  - datetime_rfc3339 — strict RFC-3339 datetime
  - datetime_rfc3339_relaxed — RFC-3339 with space instead of 'T'
  - Field-name aliasing: [WithFieldNameTag] (commonly set to "json") makes
    error namespaces use the serialized field names that API consumers actually
    see, not the internal Go struct field names.
  - Extensible: [WithCustomValidationTags] and [WithCustomTypeFunc] expose the
    full upstream registration API for project-specific rules.
  - Structured errors: every failure is an [Error] struct, not a plain string,
    enabling programmatic error categorisation and localisation downstream.

# Usage

	v, err := validator.New(
	    validator.WithFieldNameTag("json"),
	    validator.WithCustomValidationTags(validator.CustomValidationTags()),
	    validator.WithErrorTemplates(validator.ErrorTemplates()),
	)
	if err != nil {
	    return err
	}

	type Address struct {
	    Phone   string `json:"phone"    validate:"required,e164noplus"`
	    ZIP     string `json:"zip"      validate:"required,zipcode"`
	    State   string `json:"state"    validate:"required,usstate"`
	}

	err = v.ValidateStruct(Address{Phone: "12345", ZIP: "bad", State: "XX"})
	// err is a multierr containing one *Error per failed field,
	// each with a message like:
	//   "phone must be a valid E.164 formatted phone number without the leading '+' symbol"
*/
package validator

import (
	"bytes"
	"context"
	"errors"
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

// New constructs a Validator with registered custom tags, field-name aliases, and error templates from the provided options.
func New(opts ...Option) (*Validator, error) {
	v := &Validator{v: vt.New()}

	for _, applyOpt := range opts {
		err := applyOpt(v)
		if err != nil {
			return nil, err
		}
	}

	return v, nil
}

// ValidateStruct validates struct fields against registered rules, returning a multierror of structured Errors if any fail.
func (v *Validator) ValidateStruct(obj any) error {
	return v.ValidateStructCtx(context.Background(), obj)
}

// ValidateStructCtx validates struct fields with context cancellation support, returning a multierror of structured Errors if any fail.
func (v *Validator) ValidateStructCtx(ctx context.Context, obj any) error {
	vErr := v.v.StructCtx(ctx, obj)

	var (
		valErr vt.ValidationErrors
		err    error
	)
	if errors.As(vErr, &valErr) {
		for _, fe := range valErr {
			// separate tags grouped by OR
			tags := strings.SplitSeq(fe.Tag(), "|")
			for tag := range tags {
				if strings.HasPrefix(tag, "falseif") {
					// the "falseif" tag only works in combination with other tags
					continue
				}

				err = multierr.Append(err, v.tagError(fe, tag))
			}
		}
	}

	//nolint:wrapcheck
	return err
}

// tagError set the error message associated with the validation tag.
func (v *Validator) tagError(fe vt.FieldError, tag string) error {
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

		err := t.Execute(&out, ve)
		if err == nil {
			return out.String()
		}
	}

	return fmt.Sprintf("%s is invalid because fails the rule: '%s'", ve.Namespace, ve.FullTag)
}

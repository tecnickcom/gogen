package validator_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/validator"
)

const (
	// fieldTagName is the name of the tag used for the validator rules.
	fieldTagName = "json"
)

// SubStruct is an example structure type used to test nested structures.
type SubStruct struct {
	URLField string `json:"sub_string" validate:"required,url"`
	IntField int    `json:"sub_int"    validate:"required,min=2"`
}

// RootStruct is an example structure type.
type RootStruct struct {
	BoolField   bool       `json:"bool_field"`
	SubStr      SubStruct  `json:"sub_struct"     validate:"required"`
	SubStrPtr   *SubStruct `json:"sub_struct_ptr" validate:"required"`
	StringField string     `json:"string_field"   validate:"required"`
	NoNameField string     `json:"-"              validate:"required"`
}

func ExampleValidator_ValidateStruct() {
	// data structure to check
	validObj := RootStruct{
		BoolField: true,
		SubStr: SubStruct{
			URLField: "http://first.test.invalid",
			IntField: 3,
		},
		SubStrPtr: &SubStruct{
			URLField: "http://second.test.invalid",
			IntField: 123,
		},
		StringField: "hello world",
		NoNameField: "test",
	}

	// instantiate the validator object
	v, err := validator.New(
		validator.WithFieldNameTag(fieldTagName),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// check the data structure
	err = v.ValidateStruct(validObj)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("OK")

	// Output:
	// OK
}

package validation

import (
	"errors"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
	enTranslations "gopkg.in/go-playground/validator.v9/translations/en"
)

type Validator interface {
	ValidateStruct(s interface{}) error
}

type ValidatorImpl struct {
	validator    *validator.Validate
	translations ut.Translator
}

func NewValidator() *ValidatorImpl {
	v := validator.New()
	registerValidations(v)
	t := setupTranslations(v)
	return &ValidatorImpl{v, t}
}

func (v *ValidatorImpl) ValidateStruct(s interface{}) error {
	err := v.validator.Struct(s)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if validationErrors != nil {
			return translateErrors(validationErrors, v.translations)
		}
	}
	return nil
}

func setupTranslations(v *validator.Validate) ut.Translator {
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	addTranslation("required", "{0} field is required.", trans, v)
	addTranslation("eth_addr", "{0} field is not a valid address.", trans, v)
	addTranslation("prefix", "{0} field does not starts with {1}.", trans, v)

	_ = enTranslations.RegisterDefaultTranslations(v, trans)

	return trans
}

func registerValidations(v *validator.Validate) {
	if err := v.RegisterValidation("prefix", validatePrefix); err != nil {
		logrus.Fatalf("Error registering validations: %s", err.Error())
	}
}

func addTranslation(tag string, messageTemplate string, trans ut.Translator, v *validator.Validate) {
	v.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
		return ut.Add(tag, messageTemplate, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field(), fe.Param())
		return t
	})
}

func validatePrefix(fl validator.FieldLevel) bool {
	return strings.HasPrefix(fl.Field().String(), fl.Param())
}

func translateErrors(errs []validator.FieldError, t ut.Translator) error {
	translations := []string{}
	for _, e := range errs {
		translations = append(translations, e.Translate(t))
	}
	err := errors.New(strings.Join(translations, ", "))
	logrus.Debugf("Validation error: %s", err.Error())
	return err
}

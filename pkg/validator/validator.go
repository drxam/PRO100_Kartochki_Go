package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	passwordRegex = regexp.MustCompile(`.*[a-zA-Z].*[0-9].*|.*[0-9].*[a-zA-Z].*`) // хотя бы одна буква и одна цифра
)

// Validator обёртка над go-playground/validator.
type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New()
	_ = v.RegisterValidation("email", validateEmail)
	_ = v.RegisterValidation("password", validatePassword)
	return &Validator{validate: v}
}

func validateEmail(fl validator.FieldLevel) bool {
	return emailRegex.MatchString(fl.Field().String())
}

func validatePassword(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return len(s) >= 8 && passwordRegex.MatchString(s)
}

// Validate проверяет структуру и возвращает ошибки в читаемом виде.
func (v *Validator) Validate(s interface{}) map[string]string {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}
	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field()[:1]) + e.Field()[1:]
		errors[field] = messageForTag(e.Tag(), e.Param())
	}
	return errors
}

func messageForTag(tag, param string) string {
	switch tag {
	case "required":
		return "обязательное поле"
	case "email":
		return "некорректный email"
	case "password":
		return "минимум 8 символов, буквы и цифры"
	case "min":
		return fmt.Sprintf("минимум %s символов", param)
	case "max":
		return fmt.Sprintf("максимум %s символов", param)
	case "len":
		return fmt.Sprintf("должно быть ровно %s символов", param)
	default:
		return "неверное значение"
	}
}

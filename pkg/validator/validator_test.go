package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type creds struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,password"`
}

func TestValidator_ValidEmailAndPassword(t *testing.T) {
	v := New()
	errs := v.Validate(&creds{Email: "user@example.com", Password: "abcd1234"})
	require.Nil(t, errs)
}

func TestValidator_InvalidEmail(t *testing.T) {
	v := New()
	cases := []string{"plainstring", "no-at.example.com", "@nope.com", "spaces in@x.com"}
	for _, e := range cases {
		errs := v.Validate(&creds{Email: e, Password: "abcd1234"})
		require.NotNil(t, errs, "должна быть ошибка для %q", e)
		require.Contains(t, errs, "email")
	}
}

func TestValidator_PasswordTooShort(t *testing.T) {
	v := New()
	errs := v.Validate(&creds{Email: "ok@x.com", Password: "abc1"})
	require.NotNil(t, errs)
	require.Contains(t, errs, "password")
}

func TestValidator_PasswordOnlyLetters(t *testing.T) {
	v := New()
	errs := v.Validate(&creds{Email: "ok@x.com", Password: "abcdefgh"})
	require.NotNil(t, errs)
	require.Contains(t, errs, "password")
}

func TestValidator_PasswordOnlyDigits(t *testing.T) {
	v := New()
	errs := v.Validate(&creds{Email: "ok@x.com", Password: "12345678"})
	require.NotNil(t, errs)
	require.Contains(t, errs, "password")
}

func TestValidator_RequiredFieldsMissing(t *testing.T) {
	v := New()
	errs := v.Validate(&creds{})
	require.Contains(t, errs, "email")
	require.Contains(t, errs, "password")
}

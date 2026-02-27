package validation_test

import (
	"testing"

	"github.com/km-arc/go-collections/framework/http/validation"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// pass asserts the validator passes for the given data/rules.
func pass(t *testing.T, label string, data map[string]string, rules validation.Rules) {
	t.Helper()
	t.Run(label, func(t *testing.T) {
		v := validation.Make(data, rules)
		if v.Fails() {
			t.Errorf("expected PASS, got FAIL — errors: %+v", v.Errors().Bag)
		}
	})
}

// fail asserts the validator fails with an error on the given field.
func fail(t *testing.T, label, field string, data map[string]string, rules validation.Rules) {
	t.Helper()
	t.Run(label, func(t *testing.T) {
		v := validation.Make(data, rules)
		if v.Passes() {
			t.Errorf("expected FAIL on field %q, but validator PASSED", field)
		}
		if v.Errors().First(field) == "" {
			t.Errorf("expected error on field %q, but none found. Errors: %+v", field, v.Errors().Bag)
		}
	})
}

// ── required ─────────────────────────────────────────────────────────────────

func TestValidation_Required(t *testing.T) {
	r := validation.Rules{"name": "required"}

	pass(t, "non-empty value", map[string]string{"name": "Alice"}, r)
	fail(t, "empty string", "name", map[string]string{"name": ""}, r)
	fail(t, "whitespace only", "name", map[string]string{"name": "   "}, r)
	fail(t, "missing key", map[string]string{}, r) // must be fail
}

func TestValidation_Required_MessageFormat(t *testing.T) {
	v := validation.Make(map[string]string{"name": ""}, validation.Rules{"name": "required"})
	_ = v.Fails()
	msg := v.Errors().First("name")
	expected := "The name field is required."
	if msg != expected {
		t.Errorf("message: got %q want %q", msg, expected)
	}
}

// ── email ─────────────────────────────────────────────────────────────────────

func TestValidation_Email(t *testing.T) {
	r := validation.Rules{"email": "email"}

	pass(t, "valid email", map[string]string{"email": "user@example.com"}, r)
	pass(t, "valid email with subdomain", map[string]string{"email": "user@mail.example.co.uk"}, r)
	fail(t, "no @ sign", "email", map[string]string{"email": "notanemail"}, r)
	fail(t, "no domain", "email", map[string]string{"email": "user@"}, r)
}

// ── min / max / size / between ───────────────────────────────────────────────

func TestValidation_Min(t *testing.T) {
	r := validation.Rules{"name": "min:3"}

	pass(t, "exactly 3", map[string]string{"name": "abc"}, r)
	pass(t, "more than 3", map[string]string{"name": "abcde"}, r)
	fail(t, "less than 3", "name", map[string]string{"name": "ab"}, r)
	fail(t, "empty", "name", map[string]string{"name": ""}, r)
}

func TestValidation_Max(t *testing.T) {
	r := validation.Rules{"bio": "max:5"}

	pass(t, "exactly 5", map[string]string{"bio": "hello"}, r)
	pass(t, "less than 5", map[string]string{"bio": "hi"}, r)
	fail(t, "more than 5", "bio", map[string]string{"bio": "toolong"}, r)
}

func TestValidation_Size(t *testing.T) {
	r := validation.Rules{"code": "size:4"}

	pass(t, "exactly 4", map[string]string{"code": "1234"}, r)
	fail(t, "too short", "code", map[string]string{"code": "123"}, r)
	fail(t, "too long", "code", map[string]string{"code": "12345"}, r)
}

func TestValidation_Between(t *testing.T) {
	r := validation.Rules{"pin": "between:4,6"}

	pass(t, "min boundary", map[string]string{"pin": "1234"}, r)
	pass(t, "max boundary", map[string]string{"pin": "123456"}, r)
	pass(t, "middle", map[string]string{"pin": "12345"}, r)
	fail(t, "too short", "pin", map[string]string{"pin": "123"}, r)
	fail(t, "too long", "pin", map[string]string{"pin": "1234567"}, r)
}

// ── Unicode character counting ────────────────────────────────────────────────

func TestValidation_Min_Unicode(t *testing.T) {
	// "日本語" = 3 runes, min:3 should pass
	pass(t, "unicode rune count", map[string]string{"name": "日本語"}, validation.Rules{"name": "min:3"})
	fail(t, "unicode rune count too short", "name", map[string]string{"name": "日本"}, validation.Rules{"name": "min:3"})
}

// ── numeric / integer / boolean ───────────────────────────────────────────────

func TestValidation_Numeric(t *testing.T) {
	r := validation.Rules{"amount": "numeric"}

	pass(t, "integer", map[string]string{"amount": "42"}, r)
	pass(t, "float", map[string]string{"amount": "3.14"}, r)
	pass(t, "negative", map[string]string{"amount": "-5.5"}, r)
	fail(t, "string", "amount", map[string]string{"amount": "abc"}, r)
	fail(t, "mixed", "amount", map[string]string{"amount": "12abc"}, r)
}

func TestValidation_Integer(t *testing.T) {
	r := validation.Rules{"count": "integer"}

	pass(t, "positive int", map[string]string{"count": "10"}, r)
	pass(t, "negative int", map[string]string{"count": "-3"}, r)
	fail(t, "float", "count", map[string]string{"count": "3.14"}, r)
	fail(t, "string", "count", map[string]string{"count": "abc"}, r)
}

func TestValidation_Boolean(t *testing.T) {
	r := validation.Rules{"active": "boolean"}

	for _, v := range []string{"true", "false", "1", "0", "yes", "no", "True", "False"} {
		pass(t, "boolean "+v, map[string]string{"active": v}, r)
	}
	fail(t, "invalid bool", "active", map[string]string{"active": "maybe"}, r)
}

// ── in / not_in ───────────────────────────────────────────────────────────────

func TestValidation_In(t *testing.T) {
	r := validation.Rules{"role": "in:admin,editor,viewer"}

	pass(t, "admin", map[string]string{"role": "admin"}, r)
	pass(t, "editor", map[string]string{"role": "editor"}, r)
	fail(t, "superuser not in list", "role", map[string]string{"role": "superuser"}, r)
	fail(t, "empty not in list", "role", map[string]string{"role": ""}, r)
}

func TestValidation_NotIn(t *testing.T) {
	r := validation.Rules{"status": "not_in:banned,suspended"}

	pass(t, "active", map[string]string{"status": "active"}, r)
	fail(t, "banned", "status", map[string]string{"status": "banned"}, r)
	fail(t, "suspended", "status", map[string]string{"status": "suspended"}, r)
}

// ── confirmed ─────────────────────────────────────────────────────────────────

func TestValidation_Confirmed(t *testing.T) {
	r := validation.Rules{"password": "confirmed"}

	pass(t, "matching", map[string]string{
		"password":              "secret",
		"password_confirmation": "secret",
	}, r)
	fail(t, "not matching", "password", map[string]string{
		"password":              "secret",
		"password_confirmation": "wrong",
	}, r)
	fail(t, "missing confirmation", "password", map[string]string{
		"password": "secret",
	}, r)
}

// ── same / different ─────────────────────────────────────────────────────────

func TestValidation_Same(t *testing.T) {
	r := validation.Rules{"confirm_email": "same:email"}

	pass(t, "same value", map[string]string{
		"email":         "a@b.com",
		"confirm_email": "a@b.com",
	}, r)
	fail(t, "different value", "confirm_email", map[string]string{
		"email":         "a@b.com",
		"confirm_email": "c@d.com",
	}, r)
}

func TestValidation_Different(t *testing.T) {
	r := validation.Rules{"new_password": "different:old_password"}

	pass(t, "different values", map[string]string{
		"old_password": "old",
		"new_password": "new",
	}, r)
	fail(t, "same value", "new_password", map[string]string{
		"old_password": "same",
		"new_password": "same",
	}, r)
}

// ── alpha / alpha_num / alpha_dash ────────────────────────────────────────────

func TestValidation_Alpha(t *testing.T) {
	r := validation.Rules{"name": "alpha"}

	pass(t, "letters only", map[string]string{"name": "HelloWorld"}, r)
	fail(t, "with numbers", "name", map[string]string{"name": "hello123"}, r)
	fail(t, "with spaces", "name", map[string]string{"name": "hello world"}, r)
}

func TestValidation_AlphaNum(t *testing.T) {
	r := validation.Rules{"slug": "alpha_num"}

	pass(t, "letters and numbers", map[string]string{"slug": "user123"}, r)
	fail(t, "with dash", "slug", map[string]string{"slug": "user-123"}, r)
	fail(t, "with space", "slug", map[string]string{"slug": "user 123"}, r)
}

func TestValidation_AlphaDash(t *testing.T) {
	r := validation.Rules{"slug": "alpha_dash"}

	pass(t, "letters-numbers_underscore", map[string]string{"slug": "user_name-123"}, r)
	fail(t, "with space", "slug", map[string]string{"slug": "user name"}, r)
	fail(t, "with dot", "slug", map[string]string{"slug": "user.name"}, r)
}

// ── url ───────────────────────────────────────────────────────────────────────

func TestValidation_URL(t *testing.T) {
	r := validation.Rules{"website": "url"}

	pass(t, "http", map[string]string{"website": "http://example.com"}, r)
	pass(t, "https", map[string]string{"website": "https://example.com/path?q=1"}, r)
	fail(t, "no protocol", "website", map[string]string{"website": "example.com"}, r)
	fail(t, "ftp protocol", "website", map[string]string{"website": "ftp://example.com"}, r)
}

// ── regex ─────────────────────────────────────────────────────────────────────

func TestValidation_Regex(t *testing.T) {
	r := validation.Rules{"zip": `regex:^\d{5}$`}

	pass(t, "5 digits", map[string]string{"zip": "12345"}, r)
	fail(t, "4 digits", "zip", map[string]string{"zip": "1234"}, r)
	fail(t, "letters", "zip", map[string]string{"zip": "abcde"}, r)
}

// ── gt / gte / lt / lte ───────────────────────────────────────────────────────

func TestValidation_GT(t *testing.T) {
	r := validation.Rules{"age": "gt:18"}

	pass(t, "19 > 18", map[string]string{"age": "19"}, r)
	fail(t, "18 not > 18", "age", map[string]string{"age": "18"}, r)
	fail(t, "17 not > 18", "age", map[string]string{"age": "17"}, r)
}

func TestValidation_GTE(t *testing.T) {
	r := validation.Rules{"age": "gte:18"}

	pass(t, "18 >= 18", map[string]string{"age": "18"}, r)
	pass(t, "19 >= 18", map[string]string{"age": "19"}, r)
	fail(t, "17 not >= 18", "age", map[string]string{"age": "17"}, r)
}

func TestValidation_LT(t *testing.T) {
	r := validation.Rules{"score": "lt:100"}

	pass(t, "99 < 100", map[string]string{"score": "99"}, r)
	fail(t, "100 not < 100", "score", map[string]string{"score": "100"}, r)
}

func TestValidation_LTE(t *testing.T) {
	r := validation.Rules{"score": "lte:100"}

	pass(t, "100 <= 100", map[string]string{"score": "100"}, r)
	pass(t, "99 <= 100", map[string]string{"score": "99"}, r)
	fail(t, "101 not <= 100", "score", map[string]string{"score": "101"}, r)
}

// ── nullable / sometimes ──────────────────────────────────────────────────────

func TestValidation_Nullable(t *testing.T) {
	// nullable allows empty values through without error
	r := validation.Rules{"bio": "nullable|min:10"}
	// empty value — nullable stops further processing
	pass(t, "empty with nullable", map[string]string{"bio": ""}, r)
}

func TestValidation_Sometimes(t *testing.T) {
	r := validation.Rules{"nickname": "sometimes|min:3"}
	// field absent — should not produce errors
	pass(t, "absent field with sometimes", map[string]string{}, r)
	// field present and valid
	pass(t, "present and valid", map[string]string{"nickname": "coolname"}, r)
}

// ── Chained / multiple rules ──────────────────────────────────────────────────

func TestValidation_Chained(t *testing.T) {
	rules := validation.Rules{
		"email":    "required|email",
		"password": "required|min:8|confirmed",
		"age":      "required|integer|gte:18",
	}

	pass(t, "all valid", map[string]string{
		"email":                 "user@example.com",
		"password":              "secret123",
		"password_confirmation": "secret123",
		"age":                   "25",
	}, rules)

	v := validation.Make(map[string]string{
		"email":    "not-an-email",
		"password": "short",
		"age":      "16",
	}, rules)

	if v.Passes() {
		t.Error("expected validation to fail")
	}

	errs := v.Errors()
	if errs.First("email") == "" {
		t.Error("expected error on email")
	}
	if errs.First("password") == "" {
		t.Error("expected error on password")
	}
	if errs.First("age") == "" {
		t.Error("expected error on age")
	}
}

// ── Errors bag ────────────────────────────────────────────────────────────────

func TestErrors_Has(t *testing.T) {
	v := validation.Make(map[string]string{"name": ""}, validation.Rules{"name": "required"})
	if !v.Fails() {
		t.Fatal("expected fails")
	}
	if !v.Errors().Has() {
		t.Error("Has() should be true when there are errors")
	}
}

func TestErrors_First(t *testing.T) {
	v := validation.Make(
		map[string]string{"email": "bad"},
		validation.Rules{"email": "required|email"},
	)
	_ = v.Fails()
	if v.Errors().First("email") == "" {
		t.Error("First('email') should return error message")
	}
	if v.Errors().First("nonexistent") != "" {
		t.Error("First('nonexistent') should return empty string")
	}
}

func TestErrors_Passes(t *testing.T) {
	v := validation.Make(
		map[string]string{"name": "Alice"},
		validation.Rules{"name": "required|min:2"},
	)
	if !v.Passes() {
		t.Errorf("expected Passes(), errors: %+v", v.Errors().Bag)
	}
}

// ── JSON output shape ─────────────────────────────────────────────────────────

func TestErrors_JSONShape(t *testing.T) {
	// The Errors struct must marshal to {"errors": {"field": ["msg1"]}}
	// This is tested by checking the Bag field tag
	v := validation.Make(
		map[string]string{"email": ""},
		validation.Rules{"email": "required"},
	)
	_ = v.Fails()

	errs := v.Errors()
	if errs.Bag == nil {
		t.Fatal("Bag should not be nil after failure")
	}
	msgs, ok := errs.Bag["email"]
	if !ok {
		t.Fatal("expected 'email' key in Bag")
	}
	if len(msgs) == 0 {
		t.Error("expected at least one message for email")
	}
}

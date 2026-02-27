package validation

import (
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ── Types ────────────────────────────────────────────────────────────────────

// Errors holds validation errors — mirrors Laravel's MessageBag.
// JSON output: {"errors": {"field": ["msg1", "msg2"]}}
type Errors struct {
	Bag map[string][]string `json:"errors"`
}

func (e *Errors) add(field, msg string) {
	if e.Bag == nil {
		e.Bag = make(map[string][]string)
	}
	e.Bag[field] = append(e.Bag[field], msg)
}

// Has returns true if there are any errors.
func (e *Errors) Has() bool { return len(e.Bag) > 0 }

// First returns the first error for a field.
func (e *Errors) First(field string) string {
	if msgs, ok := e.Bag[field]; ok && len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

// ── Validator ────────────────────────────────────────────────────────────────

// Rules is a map of field → pipe-separated rule string.
// e.g. Rules{"email": "required|email", "age": "required|numeric|min:18"}
type Rules map[string]string

// Validator validates a flat map of input values.
type Validator struct {
	data   map[string]string
	rules  Rules
	errors *Errors
}

// Make creates a new Validator — mirrors Validator::make($data, $rules).
func Make(data map[string]string, rules Rules) *Validator {
	return &Validator{
		data:   data,
		rules:  rules,
		errors: &Errors{},
	}
}

// Fails runs validation and returns true if any rule fails.
func (v *Validator) Fails() bool {
	v.validate()
	return v.errors.Has()
}

// Passes runs validation and returns true if all rules pass.
func (v *Validator) Passes() bool { return !v.Fails() }

// Errors returns the validation error bag.
func (v *Validator) Errors() *Errors { return v.errors }

// ── Core validation loop ─────────────────────────────────────────────────────

func (v *Validator) validate() {
	for field, ruleStr := range v.rules {
		value := v.data[field]
		rules := strings.Split(ruleStr, "|")

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			// Parse rule name and optional parameter: min:3 → name=min, param=3
			name, param, _ := strings.Cut(rule, ":")

			if !v.applyRule(field, value, name, param) {
				break // stop on first failure (like Laravel's bail behaviour)
			}
		}
	}
}

// applyRule returns true if the rule passes.
func (v *Validator) applyRule(field, value, rule, param string) bool {
	switch rule {
	case "required":
		if strings.TrimSpace(value) == "" {
			v.errors.add(field, fmt.Sprintf("The %s field is required.", field))
			return false
		}

	case "string":
		// In Go everything from the form is already a string; just ensure it's present.

	case "numeric":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			v.errors.add(field, fmt.Sprintf("The %s must be a number.", field))
			return false
		}

	case "integer":
		if _, err := strconv.Atoi(value); err != nil {
			v.errors.add(field, fmt.Sprintf("The %s must be an integer.", field))
			return false
		}

	case "boolean":
		lower := strings.ToLower(value)
		valid := map[string]bool{"true": true, "false": true, "1": true, "0": true, "yes": true, "no": true}
		if !valid[lower] {
			v.errors.add(field, fmt.Sprintf("The %s field must be true or false.", field))
			return false
		}

	case "email":
		if _, err := mail.ParseAddress(value); err != nil {
			v.errors.add(field, fmt.Sprintf("The %s must be a valid email address.", field))
			return false
		}

	case "url":
		if !regexp.MustCompile(`^https?://`).MatchString(value) {
			v.errors.add(field, fmt.Sprintf("The %s must be a valid URL.", field))
			return false
		}

	case "min":
		n, _ := strconv.Atoi(param)
		if utf8.RuneCountInString(value) < n {
			v.errors.add(field, fmt.Sprintf("The %s must be at least %d characters.", field, n))
			return false
		}

	case "max":
		n, _ := strconv.Atoi(param)
		if utf8.RuneCountInString(value) > n {
			v.errors.add(field, fmt.Sprintf("The %s may not be greater than %d characters.", field, n))
			return false
		}

	case "size":
		n, _ := strconv.Atoi(param)
		if utf8.RuneCountInString(value) != n {
			v.errors.add(field, fmt.Sprintf("The %s must be %d characters.", field, n))
			return false
		}

	case "between":
		parts := strings.SplitN(param, ",", 2)
		if len(parts) != 2 {
			break
		}
		min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		l := utf8.RuneCountInString(value)
		if l < min || l > max {
			v.errors.add(field, fmt.Sprintf("The %s must be between %d and %d characters.", field, min, max))
			return false
		}

	case "in":
		allowed := strings.Split(param, ",")
		found := false
		for _, a := range allowed {
			if strings.TrimSpace(a) == value {
				found = true
				break
			}
		}
		if !found {
			v.errors.add(field, fmt.Sprintf("The selected %s is invalid.", field))
			return false
		}

	case "not_in":
		disallowed := strings.Split(param, ",")
		for _, d := range disallowed {
			if strings.TrimSpace(d) == value {
				v.errors.add(field, fmt.Sprintf("The selected %s is invalid.", field))
				return false
			}
		}

	case "confirmed":
		// Expects data[field+"_confirmation"] to match
		if v.data[field+"_confirmation"] != value {
			v.errors.add(field, fmt.Sprintf("The %s confirmation does not match.", field))
			return false
		}

	case "same":
		if v.data[param] != value {
			v.errors.add(field, fmt.Sprintf("The %s and %s must match.", field, param))
			return false
		}

	case "different":
		if v.data[param] == value {
			v.errors.add(field, fmt.Sprintf("The %s and %s must be different.", field, param))
			return false
		}

	case "alpha":
		if !regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(value) {
			v.errors.add(field, fmt.Sprintf("The %s may only contain letters.", field))
			return false
		}

	case "alpha_num":
		if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(value) {
			v.errors.add(field, fmt.Sprintf("The %s may only contain letters and numbers.", field))
			return false
		}

	case "alpha_dash":
		if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(value) {
			v.errors.add(field, fmt.Sprintf("The %s may only contain letters, numbers, dashes and underscores.", field))
			return false
		}

	case "regex":
		re, err := regexp.Compile(param)
		if err != nil || !re.MatchString(value) {
			v.errors.add(field, fmt.Sprintf("The %s format is invalid.", field))
			return false
		}

	case "nullable":
		// Always passes; allows empty values through subsequent rules.

	case "sometimes":
		// Skip remaining rules if field is absent.
		if value == "" {
			return false // stop processing this field silently
		}

	case "gt":
		f, _ := strconv.ParseFloat(value, 64)
		t, _ := strconv.ParseFloat(param, 64)
		if f <= t {
			v.errors.add(field, fmt.Sprintf("The %s must be greater than %s.", field, param))
			return false
		}

	case "gte":
		f, _ := strconv.ParseFloat(value, 64)
		t, _ := strconv.ParseFloat(param, 64)
		if f < t {
			v.errors.add(field, fmt.Sprintf("The %s must be greater than or equal to %s.", field, param))
			return false
		}

	case "lt":
		f, _ := strconv.ParseFloat(value, 64)
		t, _ := strconv.ParseFloat(param, 64)
		if f >= t {
			v.errors.add(field, fmt.Sprintf("The %s must be less than %s.", field, param))
			return false
		}

	case "lte":
		f, _ := strconv.ParseFloat(value, 64)
		t, _ := strconv.ParseFloat(param, 64)
		if f > t {
			v.errors.add(field, fmt.Sprintf("The %s must be less than or equal to %s.", field, param))
			return false
		}
	}

	return true
}

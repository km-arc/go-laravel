// Package validation provides Laravel-compatible input validation.
//
// # Overview
//
// The validation package mirrors Laravel's Validator facade and its rule syntax.
// Rules are expressed as pipe-separated strings on a map of field names.
//
// # Basic Usage
//
//	v := validation.Make(map[string]string{
//	    "name":  "Alice",
//	    "email": "alice@example.com",
//	}, validation.Rules{
//	    "name":  "required|min:2|max:100",
//	    "email": "required|email",
//	})
//
//	if v.Fails() {
//	    // v.Errors() returns *Errors with Bag map[string][]string
//	    // JSON: {"errors": {"field": ["message1", "message2"]}}
//	}
//
// # Available Rules
//
// String rules:
//   - required — field must be present and non-empty
//   - string   — passes (all Go form values are strings)
//   - min:n    — minimum n UTF-8 characters
//   - max:n    — maximum n UTF-8 characters
//   - size:n   — exactly n UTF-8 characters
//   - between:min,max — length between min and max (inclusive)
//   - alpha    — letters only [a-zA-Z]
//   - alpha_num — letters and numbers [a-zA-Z0-9]
//   - alpha_dash — letters, numbers, dashes, underscores
//   - regex:pattern — must match regexp pattern
//
// Format rules:
//   - email — valid RFC 5322 email address
//   - url   — must start with http:// or https://
//
// Numeric rules:
//   - numeric — parseable as float64
//   - integer — parseable as int
//   - gt:n    — greater than n
//   - gte:n   — greater than or equal to n
//   - lt:n    — less than n
//   - lte:n   — less than or equal to n
//
// Comparison rules:
//   - confirmed       — field_confirmation must match field
//   - same:other      — must equal data[other]
//   - different:other — must not equal data[other]
//
// Type rules:
//   - boolean — true/false/1/0/yes/no (case-insensitive)
//   - in:a,b,c     — value must be in the comma-separated list
//   - not_in:a,b,c — value must NOT be in the comma-separated list
//
// Control rules:
//   - nullable  — allows empty/missing values; stops further rule processing
//   - sometimes — skips all rules silently if field is absent
//
// # Error Bag
//
// Errors are stored in a MessageBag that serialises to the same JSON structure
// as Laravel's validation errors:
//
//	{
//	  "errors": {
//	    "email": ["The email field is required.", "The email must be a valid email address."],
//	    "age":   ["The age must be greater than or equal to 18."]
//	  }
//	}
package validation
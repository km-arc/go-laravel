package http

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxMemory = 32 << 20 // 32 MB

// Request wraps *http.Request with Laravel-style helpers.
type Request struct {
	raw *http.Request
}

// NewRequest wraps a standard *http.Request.
func NewRequest(r *http.Request) *Request {
	return &Request{raw: r}
}

// Raw returns the underlying *http.Request.
func (req *Request) Raw() *http.Request { return req.raw }

// ── Binding ──────────────────────────────────────────────────────────────────

// Bind decodes the request body into v.
// Supports JSON and application/x-www-form-urlencoded / multipart.
// JSON fields map via `json:"name"`, form fields via `form:"name"`.
func (req *Request) Bind(v any) error {
	ct := req.ContentType()

	switch {
	case strings.Contains(ct, "application/json"):
		return req.bindJSON(v)
	case strings.Contains(ct, "multipart/form-data"):
		if err := req.raw.ParseMultipartForm(maxMemory); err != nil {
			return err
		}
		return bindForm(req.raw.MultipartForm.Value, v)
	default:
		if err := req.raw.ParseForm(); err != nil {
			return err
		}
		return bindForm(map[string][]string(req.raw.PostForm), v)
	}
}

func (req *Request) bindJSON(v any) error {
	defer req.raw.Body.Close()
	body, err := io.ReadAll(req.raw.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty request body")
	}
	return json.Unmarshal(body, v)
}

// bindForm maps form values onto a struct using `form:"field"` tags.
func bindForm(values map[string][]string, v any) error {
	// Use JSON round-trip: build map → marshal → unmarshal into struct
	// This keeps dependencies minimal while supporting nested structs via json tags.
	m := make(map[string]any, len(values))
	for k, vals := range values {
		if len(vals) == 1 {
			m[k] = vals[0]
		} else {
			m[k] = vals
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ── Input helpers ────────────────────────────────────────────────────────────

// Input returns a single input value (query string OR post body).
func (req *Request) Input(key string, fallback ...string) string {
	_ = req.raw.ParseForm()
	v := req.raw.FormValue(key)
	if v == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// Query returns a query-string value.
func (req *Request) Query(key string, fallback ...string) string {
	v := req.raw.URL.Query().Get(key)
	if v == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return v
}

// All returns all input as a flat map (query + post).
func (req *Request) All() map[string]string {
	_ = req.raw.ParseForm()
	out := make(map[string]string)
	for k, v := range req.raw.Form {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// Has returns true if the key is present and non-empty.
func (req *Request) Has(key string) bool {
	return req.Input(key) != ""
}

// RouteParam returns a URL route parameter (chi).
func (req *Request) RouteParam(key string) string {
	return chi.URLParam(req.raw, key)
}

// Header returns a request header value.
func (req *Request) Header(key string) string {
	return req.raw.Header.Get(key)
}

// BearerToken extracts the token from Authorization: Bearer <token>.
func (req *Request) BearerToken() string {
	auth := req.raw.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// IP returns the client IP (respects RealIP middleware).
func (req *Request) IP() string {
	return req.raw.RemoteAddr
}

// Method returns the HTTP method.
func (req *Request) Method() string { return req.raw.Method }

// Path returns the URL path.
func (req *Request) Path() string { return req.raw.URL.Path }

// ContentType returns the Content-Type header value.
func (req *Request) ContentType() string {
	return req.raw.Header.Get("Content-Type")
}

// IsJSON returns true when the request expects a JSON response.
func (req *Request) IsJSON() bool {
	return strings.Contains(req.raw.Header.Get("Accept"), "application/json") ||
		strings.Contains(req.ContentType(), "application/json")
}

// ── File uploads ─────────────────────────────────────────────────────────────

// File returns an uploaded file by field name.
func (req *Request) File(key string) (*multipart.FileHeader, error) {
	if err := req.raw.ParseMultipartForm(maxMemory); err != nil {
		return nil, err
	}
	_, fh, err := req.raw.FormFile(key)
	return fh, err
}

// Files returns all uploaded files for a field.
func (req *Request) Files(key string) ([]*multipart.FileHeader, error) {
	if err := req.raw.ParseMultipartForm(maxMemory); err != nil {
		return nil, err
	}
	if req.raw.MultipartForm == nil {
		return nil, errors.New("no multipart form")
	}
	return req.raw.MultipartForm.File[key], nil
}

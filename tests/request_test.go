package http_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gohttp "github.com/km-arc/go-laravel/http"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newJSONRequest(t *testing.T, body string) *gohttp.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return gohttp.NewRequest(req)
}

func newFormRequest(t *testing.T, values url.Values) *gohttp.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return gohttp.NewRequest(req)
}

func newGetRequest(t *testing.T, rawQuery string) *gohttp.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	return gohttp.NewRequest(req)
}

// ── Bind JSON ────────────────────────────────────────────────────────────────

func TestRequest_BindJSON(t *testing.T) {
	type user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	req := newJSONRequest(t, `{"name":"Alice","email":"alice@example.com"}`)

	var u user
	if err := req.Bind(&u); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("Name: got %q want %q", u.Name, "Alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email: got %q want %q", u.Email, "alice@example.com")
	}
}

func TestRequest_BindJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	r := gohttp.NewRequest(req)

	var v any
	err := r.Bind(&v)
	if err == nil {
		t.Error("expected error for empty body, got nil")
	}
}

func TestRequest_BindJSON_InvalidJSON(t *testing.T) {
	req := newJSONRequest(t, `{bad json}`)
	var v map[string]any
	if err := req.Bind(&v); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── Bind Form ────────────────────────────────────────────────────────────────

func TestRequest_BindForm(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	vals := url.Values{"name": {"Bob"}}
	req := newFormRequest(t, vals)

	var p payload
	if err := req.Bind(&p); err != nil {
		t.Fatalf("Bind form error: %v", err)
	}
	if p.Name != "Bob" {
		t.Errorf("Name: got %q want %q", p.Name, "Bob")
	}
}

// ── Input / Query ─────────────────────────────────────────────────────────────

func TestRequest_Input(t *testing.T) {
	vals := url.Values{"username": {"charlie"}}
	req := newFormRequest(t, vals)

	if got := req.Input("username"); got != "charlie" {
		t.Errorf("Input: got %q want %q", got, "charlie")
	}
}

func TestRequest_Input_Fallback(t *testing.T) {
	req := newGetRequest(t, "")
	if got := req.Input("missing", "default"); got != "default" {
		t.Errorf("Input fallback: got %q want %q", got, "default")
	}
}

func TestRequest_Query(t *testing.T) {
	req := newGetRequest(t, "page=2&limit=10")

	if got := req.Query("page"); got != "2" {
		t.Errorf("Query page: got %q want %q", got, "2")
	}
	if got := req.Query("limit"); got != "10" {
		t.Errorf("Query limit: got %q want %q", got, "10")
	}
}

func TestRequest_Query_Fallback(t *testing.T) {
	req := newGetRequest(t, "")
	if got := req.Query("missing", "1"); got != "1" {
		t.Errorf("Query fallback: got %q want %q", got, "1")
	}
}

func TestRequest_All(t *testing.T) {
	vals := url.Values{"a": {"1"}, "b": {"2"}}
	req := newFormRequest(t, vals)
	all := req.All()

	if all["a"] != "1" {
		t.Errorf("All[a]: got %q want %q", all["a"], "1")
	}
	if all["b"] != "2" {
		t.Errorf("All[b]: got %q want %q", all["b"], "2")
	}
}

func TestRequest_Has(t *testing.T) {
	vals := url.Values{"name": {"Alice"}, "empty": {""}}
	req := newFormRequest(t, vals)

	if !req.Has("name") {
		t.Error("Has('name') should be true")
	}
	if req.Has("empty") {
		t.Error("Has('empty') should be false for blank value")
	}
	if req.Has("missing") {
		t.Error("Has('missing') should be false")
	}
}

// ── Headers / Auth ────────────────────────────────────────────────────────────

func TestRequest_Header(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Custom", "value123")
	req := gohttp.NewRequest(r)

	if got := req.Header("X-Custom"); got != "value123" {
		t.Errorf("Header: got %q want %q", got, "value123")
	}
}

func TestRequest_BearerToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer my-secret-token")
	req := gohttp.NewRequest(r)

	if got := req.BearerToken(); got != "my-secret-token" {
		t.Errorf("BearerToken: got %q want %q", got, "my-secret-token")
	}
}

func TestRequest_BearerToken_Missing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	req := gohttp.NewRequest(r)

	if got := req.BearerToken(); got != "" {
		t.Errorf("BearerToken should be empty, got %q", got)
	}
}

// ── IsJSON ────────────────────────────────────────────────────────────────────

func TestRequest_IsJSON_ContentType(t *testing.T) {
	req := newJSONRequest(t, `{}`)
	if !req.IsJSON() {
		t.Error("IsJSON should be true when Content-Type is application/json")
	}
}

func TestRequest_IsJSON_Accept(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept", "application/json")
	req := gohttp.NewRequest(r)
	if !req.IsJSON() {
		t.Error("IsJSON should be true when Accept is application/json")
	}
}

// ── Method / Path ─────────────────────────────────────────────────────────────

func TestRequest_Method(t *testing.T) {
	r := httptest.NewRequest(http.MethodDelete, "/resource/1", nil)
	req := gohttp.NewRequest(r)
	if req.Method() != http.MethodDelete {
		t.Errorf("Method: got %q want DELETE", req.Method())
	}
}

func TestRequest_Path(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req := gohttp.NewRequest(r)
	if req.Path() != "/api/v1/users" {
		t.Errorf("Path: got %q want /api/v1/users", req.Path())
	}
}

// ── Multipart file upload ─────────────────────────────────────────────────────

func TestRequest_File(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("fake-image-data"))
	_ = w.Close()

	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Type", w.FormDataContentType())
	req := gohttp.NewRequest(r)

	fh, err := req.File("avatar")
	if err != nil {
		t.Fatalf("File error: %v", err)
	}
	if fh.Filename != "avatar.png" {
		t.Errorf("Filename: got %q want %q", fh.Filename, "avatar.png")
	}
}

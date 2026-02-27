package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gohttp "github.com/km-arc/go-collections/framework/http"
	"github.com/km-arc/go-collections/framework/http/validation"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newResponse(t *testing.T) (*gohttp.Response, *httptest.ResponseRecorder) {
	t.Helper()
	rr := httptest.NewRecorder()
	return gohttp.NewResponse(rr), rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
	return m
}

// ── JSON ──────────────────────────────────────────────────────────────────────

func TestResponse_JSON(t *testing.T) {
	res, rr := newResponse(t)
	res.JSON(http.StatusOK, map[string]any{"key": "val"})

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q want application/json", ct)
	}
	m := decodeJSON(t, rr)
	if m["key"] != "val" {
		t.Errorf("body key: got %v want val", m["key"])
	}
}

func TestResponse_Success(t *testing.T) {
	res, rr := newResponse(t)
	res.Success(map[string]any{"id": float64(1)})

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", rr.Code)
	}
	m := decodeJSON(t, rr)
	data, ok := m["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data envelope, got %T", m["data"])
	}
	if data["id"] != float64(1) {
		t.Errorf("data.id: got %v want 1", data["id"])
	}
}

func TestResponse_Created(t *testing.T) {
	res, rr := newResponse(t)
	res.Created(map[string]any{"name": "Alice"})

	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d want 201", rr.Code)
	}
	m := decodeJSON(t, rr)
	if _, ok := m["data"]; !ok {
		t.Error("expected 'data' key in response")
	}
}

func TestResponse_NoContent(t *testing.T) {
	res, rr := newResponse(t)
	res.NoContent()

	if rr.Code != http.StatusNoContent {
		t.Errorf("status: got %d want 204", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rr.Body.String())
	}
}

// ── Error helpers ─────────────────────────────────────────────────────────────

func TestResponse_Error(t *testing.T) {
	res, rr := newResponse(t)
	res.Error(http.StatusBadRequest, "bad input")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want 400", rr.Code)
	}
	m := decodeJSON(t, rr)
	if m["message"] != "bad input" {
		t.Errorf("message: got %v want 'bad input'", m["message"])
	}
}

func TestResponse_Unauthorized_DefaultMessage(t *testing.T) {
	res, rr := newResponse(t)
	res.Unauthorized()

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", rr.Code)
	}
	m := decodeJSON(t, rr)
	if m["message"] != "Unauthenticated." {
		t.Errorf("message: got %v", m["message"])
	}
}

func TestResponse_Unauthorized_CustomMessage(t *testing.T) {
	res, rr := newResponse(t)
	res.Unauthorized("Token expired.")

	m := decodeJSON(t, rr)
	if m["message"] != "Token expired." {
		t.Errorf("message: got %v", m["message"])
	}
}

func TestResponse_Forbidden(t *testing.T) {
	res, rr := newResponse(t)
	res.Forbidden()

	if rr.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", rr.Code)
	}
	m := decodeJSON(t, rr)
	if m["message"] != "This action is unauthorized." {
		t.Errorf("message: got %v", m["message"])
	}
}

func TestResponse_NotFound(t *testing.T) {
	res, rr := newResponse(t)
	res.NotFound()

	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d want 404", rr.Code)
	}
}

func TestResponse_ServerError(t *testing.T) {
	res, rr := newResponse(t)
	res.ServerError()

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d want 500", rr.Code)
	}
}

// ── ValidationError ───────────────────────────────────────────────────────────

func TestResponse_ValidationError(t *testing.T) {
	res, rr := newResponse(t)

	v := validation.Make(
		map[string]string{"email": ""},
		validation.Rules{"email": "required|email"},
	)
	_ = v.Fails()
	res.ValidationError(v.Errors())

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d want 422", rr.Code)
	}

	var body struct {
		Errors map[string][]string `json:"errors"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body.Errors["email"]; !ok {
		t.Error("expected 'email' key in errors")
	}
}

// ── Redirects ─────────────────────────────────────────────────────────────────

func TestResponse_RedirectTo(t *testing.T) {
	res, rr := newResponse(t)
	res.RedirectTo("/dashboard")

	if rr.Code != http.StatusFound {
		t.Errorf("status: got %d want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/dashboard" {
		t.Errorf("Location: got %q want /dashboard", loc)
	}
}

func TestResponse_RedirectBack_WithReferer(t *testing.T) {
	rr := httptest.NewRecorder()
	res := gohttp.NewResponse(rr)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Referer", "/previous")

	res.RedirectBack(r, "/home")

	if rr.Code != http.StatusFound {
		t.Errorf("status: got %d want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/previous" {
		t.Errorf("Location: got %q want /previous", loc)
	}
}

func TestResponse_RedirectBack_Fallback(t *testing.T) {
	rr := httptest.NewRecorder()
	res := gohttp.NewResponse(rr)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Referer header
	res.RedirectBack(r, "/home")

	if loc := rr.Header().Get("Location"); loc != "/home" {
		t.Errorf("Location fallback: got %q want /home", loc)
	}
}

// ── Raw() ─────────────────────────────────────────────────────────────────────

func TestResponse_Raw(t *testing.T) {
	_, rr := newResponse(t)
	res := gohttp.NewResponse(rr)
	if res.Raw() == nil {
		t.Error("Raw() should not be nil")
	}
}

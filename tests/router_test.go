package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/km-arc/go-collections/framework/routing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func do(t *testing.T, router *routing.Router, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ── HTTP verbs ────────────────────────────────────────────────────────────────

func TestRouter_Get(t *testing.T) {
	r := routing.New()
	r.Get("/hello", okHandler)

	rr := do(t, r, http.MethodGet, "/hello")
	if rr.Code != http.StatusOK {
		t.Errorf("GET /hello: got %d want 200", rr.Code)
	}
}

func TestRouter_Post(t *testing.T) {
	r := routing.New()
	r.Post("/users", okHandler)

	rr := do(t, r, http.MethodPost, "/users")
	if rr.Code != http.StatusOK {
		t.Errorf("POST /users: got %d want 200", rr.Code)
	}
}

func TestRouter_Put(t *testing.T) {
	r := routing.New()
	r.Put("/users/{id}", okHandler)

	rr := do(t, r, http.MethodPut, "/users/1")
	if rr.Code != http.StatusOK {
		t.Errorf("PUT /users/1: got %d want 200", rr.Code)
	}
}

func TestRouter_Patch(t *testing.T) {
	r := routing.New()
	r.Patch("/users/{id}", okHandler)

	rr := do(t, r, http.MethodPatch, "/users/1")
	if rr.Code != http.StatusOK {
		t.Errorf("PATCH /users/1: got %d want 200", rr.Code)
	}
}

func TestRouter_Delete(t *testing.T) {
	r := routing.New()
	r.Delete("/users/{id}", okHandler)

	rr := do(t, r, http.MethodDelete, "/users/1")
	if rr.Code != http.StatusOK {
		t.Errorf("DELETE /users/1: got %d want 200", rr.Code)
	}
}

func TestRouter_Any(t *testing.T) {
	r := routing.New()
	r.Any("/ping", okHandler)

	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		rr := do(t, r, method, "/ping")
		if rr.Code != http.StatusOK {
			t.Errorf("ANY %s /ping: got %d want 200", method, rr.Code)
		}
	}
}

// ── 404 for unregistered routes ──────────────────────────────────────────────

func TestRouter_NotFound(t *testing.T) {
	r := routing.New()
	rr := do(t, r, http.MethodGet, "/not-registered")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ── Route params ─────────────────────────────────────────────────────────────

func TestRouter_Param(t *testing.T) {
	r := routing.New()
	r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := routing.Param(req, "id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(id))
	})

	rr := do(t, r, http.MethodGet, "/users/42")
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d want 200", rr.Code)
	}
	if rr.Body.String() != "42" {
		t.Errorf("got body %q want %q", rr.Body.String(), "42")
	}
}

// ── Prefix / Group ───────────────────────────────────────────────────────────

func TestRouter_Prefix(t *testing.T) {
	r := routing.New()
	r.Prefix("/api/v1", func(api *routing.Router) {
		api.Get("/users", okHandler)
	})

	rr := do(t, r, http.MethodGet, "/api/v1/users")
	if rr.Code != http.StatusOK {
		t.Errorf("GET /api/v1/users: got %d want 200", rr.Code)
	}

	// Root must 404
	rr2 := do(t, r, http.MethodGet, "/users")
	if rr2.Code != http.StatusNotFound {
		t.Errorf("GET /users: expected 404, got %d", rr2.Code)
	}
}

func TestRouter_Group_Middleware(t *testing.T) {
	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	r := routing.New()
	r.Group(func(g *routing.Router) {
		g.Middleware(mw)
		g.Get("/protected", okHandler)
	})

	do(t, r, http.MethodGet, "/protected")
	if !called {
		t.Error("expected middleware to be called")
	}
}

// ── Resource routes ───────────────────────────────────────────────────────────

type stubController struct{}

func (s *stubController) Index(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(200) }
func (s *stubController) Store(w http.ResponseWriter, r *http.Request)   { w.WriteHeader(201) }
func (s *stubController) Show(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(200) }
func (s *stubController) Update(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(200) }
func (s *stubController) Destroy(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }

func TestRouter_Resource(t *testing.T) {
	r := routing.New()
	r.Resource("/photos", &stubController{})

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/photos", 200},
		{"POST", "/photos", 201},
		{"GET", "/photos/1", 200},
		{"PUT", "/photos/1", 200},
		{"PATCH", "/photos/1", 200},
		{"DELETE", "/photos/1", 204},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rr := do(t, r, tt.method, tt.path)
			if rr.Code != tt.want {
				t.Errorf("got %d want %d", rr.Code, tt.want)
			}
		})
	}
}

// ── Handler() returns http.Handler ───────────────────────────────────────────

func TestRouter_HandlerInterface(t *testing.T) {
	r := routing.New()
	r.Get("/ping", okHandler)
	var _ http.Handler = r.Handler()
}

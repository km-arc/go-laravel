package routing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router wraps chi.Router with Laravel-style helpers.
type Router struct {
	mux chi.Router
}

// New creates a Router with sane defaults (Logger, Recoverer).
func New() *Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	return &Router{mux: r}
}

// ── HTTP verbs ───────────────────────────────────────────────────────────────

func (r *Router) Get(pattern string, h http.HandlerFunc)    { r.mux.Get(pattern, h) }
func (r *Router) Post(pattern string, h http.HandlerFunc)   { r.mux.Post(pattern, h) }
func (r *Router) Put(pattern string, h http.HandlerFunc)    { r.mux.Put(pattern, h) }
func (r *Router) Patch(pattern string, h http.HandlerFunc)  { r.mux.Patch(pattern, h) }
func (r *Router) Delete(pattern string, h http.HandlerFunc) { r.mux.Delete(pattern, h) }

// Any registers a handler for all common HTTP methods.
func (r *Router) Any(pattern string, h http.HandlerFunc) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"} {
		r.mux.Method(m, pattern, h)
	}
}

// ── Groups & Prefixes ────────────────────────────────────────────────────────

// Group creates an inline group — Laravel: Route::group([], fn)
func (r *Router) Group(fn func(r *Router)) {
	r.mux.Group(func(mx chi.Router) {
		fn(&Router{mux: mx})
	})
}

// Prefix creates a sub-router with a URL prefix — Laravel: Route::prefix('/api')
func (r *Router) Prefix(pattern string, fn func(r *Router)) {
	r.mux.Route(pattern, func(mx chi.Router) {
		fn(&Router{mux: mx})
	})
}

// ── Middleware ───────────────────────────────────────────────────────────────

// Middleware adds one or more middleware to the router.
func (r *Router) Middleware(mw ...func(http.Handler) http.Handler) {
	r.mux.Use(mw...)
}

// ── Named / Resource routes ──────────────────────────────────────────────────

// Resource registers standard RESTful routes for a resource controller.
//
//	GET    /photos           → c.Index
//	POST   /photos           → c.Store
//	GET    /photos/{id}      → c.Show
//	PUT    /photos/{id}      → c.Update
//	DELETE /photos/{id}      → c.Destroy
type ResourceController interface {
	Index(w http.ResponseWriter, r *http.Request)
	Store(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Destroy(w http.ResponseWriter, r *http.Request)
}

func (r *Router) Resource(pattern string, c ResourceController) {
	r.mux.Get(pattern, c.Index)
	r.mux.Post(pattern, c.Store)
	r.mux.Get(pattern+"/{id}", c.Show)
	r.mux.Put(pattern+"/{id}", c.Update)
	r.mux.Patch(pattern+"/{id}", c.Update)
	r.mux.Delete(pattern+"/{id}", c.Destroy)
}

// ── Static files ─────────────────────────────────────────────────────────────

// Static serves a filesystem at the given prefix.
// e.g. router.Static("/public", "./public")
func (r *Router) Static(prefix, dir string) {
	fs := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))
	r.mux.Get(prefix+"/*", func(w http.ResponseWriter, req *http.Request) {
		fs.ServeHTTP(w, req)
	})
}

// ── Params ───────────────────────────────────────────────────────────────────

// Param extracts a URL param — equivalent to $request->route('id')
func Param(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// ── Serve ────────────────────────────────────────────────────────────────────

// ServeHTTP implements http.Handler so Router can be passed to http.ListenAndServe.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Handler returns the underlying http.Handler (for testing etc.).
func (r *Router) Handler() http.Handler {
	return r.mux
}

package main

import (
	"net/http"

	"github.com/km-arc/go-laravel/app"
	gohttp "github.com/km-arc/go-laravel/http"
	"github.com/km-arc/go-laravel/http/validation"
	"github.com/km-arc/go-laravel/routing"
)

func main() {
	application := app.New() // loads .env automatically

	r := application.Router

	// ── Basic routes ─────────────────────────────────────────────────────────

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		res := gohttp.NewResponse(w)
		res.Success(map[string]any{"message": "Welcome to Go-Laravel!"})
	})

	// ── Route prefix (like Route::prefix('api')) ──────────────────────────────

	r.Prefix("/api/v1", func(api *routing.Router) {

		// GET /api/v1/users
		api.Get("/users", func(w http.ResponseWriter, req *http.Request) {
			res := gohttp.NewResponse(w)
			res.Success([]map[string]any{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			})
		})

		// POST /api/v1/users
		api.Post("/users", func(w http.ResponseWriter, req *http.Request) {
			request := gohttp.NewRequest(req)
			res := gohttp.NewResponse(w)

			// 1. Bind JSON body into a struct
			var body struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Age   string `json:"age"`
			}
			if err := request.Bind(&body); err != nil {
				res.Error(http.StatusBadRequest, err.Error())
				return
			}

			// 2. Validate — Laravel-style rules
			v := validation.Make(map[string]string{
				"name":  body.Name,
				"email": body.Email,
				"age":   body.Age,
			}, validation.Rules{
				"name":  "required|min:2|max:100",
				"email": "required|email",
				"age":   "required|numeric|gte:18",
			})

			if v.Fails() {
				// 3. Return 422 {"errors": {"field": ["msg"]}}
				res.ValidationError(v.Errors())
				return
			}

			// 4. Return 201 created
			res.Created(map[string]any{
				"name":  body.Name,
				"email": body.Email,
			})
		})

		// GET /api/v1/users/{id}
		api.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			res := gohttp.NewResponse(w)
			id := routing.Param(req, "id")
			res.Success(map[string]any{"id": id})
		})

	})

	// ── Auth group with middleware ─────────────────────────────────────────────

	r.Group(func(protected *routing.Router) {
		protected.Middleware(AuthMiddleware)

		protected.Get("/profile", func(w http.ResponseWriter, req *http.Request) {
			res := gohttp.NewResponse(w)
			res.Success(map[string]any{"user": "authenticated"})
		})
	})

	application.Run()
}

// AuthMiddleware is an example JWT/token guard.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := gohttp.NewRequest(r)
		res := gohttp.NewResponse(w)

		if req.BearerToken() == "" {
			res.Unauthorized()
			return
		}
		next.ServeHTTP(w, r)
	})
}

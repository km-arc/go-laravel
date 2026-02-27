// Package http provides Laravel-compatible request and response helpers.
//
// # Request
//
// Request wraps *http.Request with a fluent API mirroring Laravel's
// Illuminate\Http\Request.
//
//	req := gohttp.NewRequest(r)
//
//	// Bind JSON / form body into a struct
//	var payload struct {
//	    Name string `json:"name"`
//	}
//	if err := req.Bind(&payload); err != nil { ... }
//
//	// Input retrieval (query string + POST body)
//	name  := req.Input("name", "default")
//	page  := req.Query("page", "1")
//	all   := req.All()          // map[string]string
//	ok    := req.Has("name")
//
//	// Route params (requires Chi router)
//	id := req.RouteParam("id")
//
//	// Headers and auth
//	token := req.BearerToken()
//	val   := req.Header("X-Custom")
//
//	// Type checks
//	req.IsJSON()   // Accept: application/json OR Content-Type: application/json
//	req.Method()   // "GET", "POST", ...
//	req.Path()     // "/api/v1/users"
//	req.IP()
//
//	// File uploads
//	fh, err := req.File("avatar")
//	files, err := req.Files("attachments")
//
// # Response
//
// Response wraps http.ResponseWriter with helpers matching Laravel's
// response() helper and JsonResponse.
//
//	res := gohttp.NewResponse(w)
//
//	// JSON
//	res.JSON(200, data)           // raw JSON with status
//	res.Success(data)             // 200 {"data": ...}
//	res.Created(data)             // 201 {"data": ...}
//	res.NoContent()               // 204
//
//	// Errors
//	res.Error(400, "bad input")   // {"message": "bad input"}
//	res.Unauthorized()            // 401 {"message": "Unauthenticated."}
//	res.Forbidden()               // 403 {"message": "This action is unauthorized."}
//	res.NotFound()                // 404 {"message": "Not found."}
//	res.ServerError()             // 500 {"message": "Server Error."}
//	res.ValidationError(errs)     // 422 {"errors": {"field": ["msg"]}}
//
//	// Redirects
//	res.RedirectTo("/dashboard")                // 302
//	res.RedirectBack(r, "/fallback")            // 302 to Referer
//	res.Redirect(http.StatusMovedPermanently, "/new") // custom code
//
// # ViewEngine
//
//	engine := gohttp.NewViewEngine("./views", ".html")
//	engine.View(w, "home", map[string]any{"title": "Home"})
//	engine.ViewWithLayout(w, "layouts/app", "home", data)
package http
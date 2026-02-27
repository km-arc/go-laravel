# Go-Laravel Framework

A Laravel-style HTTP framework for Go, built on top of [Chi](https://github.com/go-chi/chi) and compatible with the [go-collections](https://github.com/km-arc/go-collections) package.

> If you know Laravel, you already know this framework.

---

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Directory Structure](#directory-structure)
4. [Configuration](#configuration)
5. [Routing](#routing)
6. [Requests](#requests)
7. [Validation](#validation)
8. [Responses](#responses)
9. [Views / Templates](#views--templates)
10. [Controllers](#controllers)
11. [Running Tests](#running-tests)

---

## Installation

```bash
go get github.com/km-arc/go-collections/framework
go get github.com/go-chi/chi/v5
go get github.com/joho/godotenv
```

---

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/km-arc/go-collections/framework/app"
    gohttp "github.com/km-arc/go-collections/framework/http"
)

func main() {
    application := app.New() // loads .env automatically

    application.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
        gohttp.NewResponse(w).Success(map[string]any{
            "message": "Hello from Go-Laravel!",
        })
    })

    application.Run() // listens on APP_PORT (default :8000)
}
```

---

## Directory Structure

```
your-app/
├── main.go
├── .env
├── views/              ← HTML templates
├── public/             ← Static files
└── app/
    └── controllers/    ← Your controllers
```

---

## Configuration

### .env file

```dotenv
APP_NAME=MyApp
APP_ENV=local            # local | production | testing
APP_DEBUG=true
APP_URL=http://localhost
APP_PORT=8000
APP_KEY=base64:your-secret-key

DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=myapp
DB_USERNAME=root
DB_PASSWORD=

MAIL_DRIVER=smtp
MAIL_HOST=mailhog
MAIL_PORT=1025
MAIL_FROM_ADDRESS=hello@example.com
```

### Accessing Config Values

```go
cfg := config.Load()            // reads .env

cfg.App.Name                    // "MyApp"
cfg.App.Env                     // "local"
cfg.App.Debug                   // true
cfg.DB.Database                 // "myapp"

// Arbitrary keys
config.Get("CUSTOM_KEY", "default")
config.GetInt("WORKERS", 4)
config.GetBool("FEATURE_FLAG", false)
```

---

## Routing

> Compatible with [Laravel Routing docs](https://laravel.com/docs/routing).

### Basic Routes

```go
r := application.Router

r.Get("/users", handler)
r.Post("/users", handler)
r.Put("/users/{id}", handler)
r.Patch("/users/{id}", handler)
r.Delete("/users/{id}", handler)
r.Any("/ping", handler)         // all HTTP verbs
```

### Route Parameters

```go
// Laravel: $request->route('id')  →  Go: routing.Param(r, "id")
r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
    id := routing.Param(req, "id")
    // id = "42" for GET /users/42
})
```

### Route Groups

```go
// Laravel: Route::group(['middleware' => ['auth']], fn)
r.Group(func(g *routing.Router) {
    g.Middleware(AuthMiddleware)

    g.Get("/dashboard", dashboardHandler)
    g.Get("/profile", profileHandler)
})
```

### Route Prefixes

```go
// Laravel: Route::prefix('api/v1')->group(fn)
r.Prefix("/api/v1", func(api *routing.Router) {
    api.Get("/users", listUsersHandler)
    api.Post("/users", createUserHandler)
})
```

### Middleware

```go
// Apply globally
r.Middleware(middleware.Logger, middleware.Recoverer)

// Apply to a group
r.Group(func(g *routing.Router) {
    g.Middleware(AuthMiddleware, RateLimitMiddleware)
    g.Get("/admin", adminHandler)
})
```

### Resource Controllers

```go
// Laravel: Route::resource('photos', PhotoController::class)
r.Resource("/photos", &PhotoController{})

// Registers:
// GET    /photos           → Index
// POST   /photos           → Store
// GET    /photos/{id}      → Show
// PUT    /photos/{id}      → Update
// PATCH  /photos/{id}      → Update
// DELETE /photos/{id}      → Destroy
```

Implement the `ResourceController` interface:

```go
type PhotoController struct{}

func (c *PhotoController) Index(w http.ResponseWriter, r *http.Request)   { /* list */ }
func (c *PhotoController) Store(w http.ResponseWriter, r *http.Request)   { /* create */ }
func (c *PhotoController) Show(w http.ResponseWriter, r *http.Request)    { /* show one */ }
func (c *PhotoController) Update(w http.ResponseWriter, r *http.Request)  { /* update */ }
func (c *PhotoController) Destroy(w http.ResponseWriter, r *http.Request) { /* delete */ }
```

### Static Files

```go
// Laravel: Route::get('public/{file}', ...)  →  Go: router.Static(prefix, dir)
r.Static("/public", "./public")
```

---

## Requests

> Compatible with [Laravel Request docs](https://laravel.com/docs/requests).

### Creating a Request

```go
func handler(w http.ResponseWriter, r *http.Request) {
    req := gohttp.NewRequest(r)
    res := gohttp.NewResponse(w)
    // ...
}
```

### Binding Input to a Struct

```go
// Laravel: $request->validate([...])  →  Go: req.Bind(&struct)
// Works with JSON, form-urlencoded, and multipart.

var payload struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

if err := req.Bind(&payload); err != nil {
    res.Error(http.StatusBadRequest, err.Error())
    return
}
```

### Retrieving Input

```go
// Laravel: $request->input('name', 'default')
name := req.Input("name", "Guest")

// Laravel: $request->query('page', 1)
page := req.Query("page", "1")

// Laravel: $request->all()
all := req.All()  // map[string]string

// Laravel: $request->has('name')
if req.Has("name") { ... }
```

### Route Parameters

```go
// Laravel: $request->route('id')
id := req.RouteParam("id")
```

### Headers & Auth

```go
// Laravel: $request->header('X-Custom')
val := req.Header("X-Custom")

// Laravel: $request->bearerToken()
token := req.BearerToken()

// Laravel: $request->ip()
ip := req.IP()
```

### Request Type Checks

```go
// Laravel: $request->wantsJson()
if req.IsJSON() { ... }

// Laravel: $request->method()
method := req.Method()   // "GET", "POST", etc.

// Laravel: $request->path()
path := req.Path()       // "/api/v1/users"
```

### File Uploads

```go
// Laravel: $request->file('avatar')
fileHeader, err := req.File("avatar")
if err != nil {
    res.Error(http.StatusBadRequest, "No file uploaded")
    return
}
// fileHeader.Filename, fileHeader.Size, fileHeader.Open()

// Multiple files: $request->file('attachments')
files, err := req.Files("attachments")
```

---

## Validation

> Compatible with [Laravel Validation docs](https://laravel.com/docs/validation).

### Basic Usage

```go
// Laravel:
// $validated = $request->validate([
//     'name'  => 'required|string|max:255',
//     'email' => 'required|email',
// ]);

v := validation.Make(req.All(), validation.Rules{
    "name":  "required|string|max:255",
    "email": "required|email",
})

if v.Fails() {
    res.ValidationError(v.Errors())   // 422 {"errors": {"field": ["msg"]}}
    return
}
```

### Available Rules

| Rule | Laravel Equivalent | Example |
|------|-------------------|---------|
| `required` | `required` | `"required"` |
| `string` | `string` | `"required\|string"` |
| `email` | `email` | `"required\|email"` |
| `numeric` | `numeric` | `"numeric"` |
| `integer` | `integer` | `"integer"` |
| `boolean` | `boolean` | `"boolean"` |
| `url` | `url` | `"url"` |
| `min:n` | `min:n` | `"min:3"` |
| `max:n` | `max:n` | `"max:255"` |
| `size:n` | `size:n` | `"size:10"` |
| `between:min,max` | `between:min,max` | `"between:4,8"` |
| `in:a,b,c` | `in:a,b,c` | `"in:admin,editor"` |
| `not_in:a,b` | `not_in:a,b` | `"not_in:banned"` |
| `confirmed` | `confirmed` | `"confirmed"` (needs `field_confirmation`) |
| `same:other` | `same:other` | `"same:email"` |
| `different:other` | `different:other` | `"different:old_password"` |
| `alpha` | `alpha` | `"alpha"` |
| `alpha_num` | `alpha_num` | `"alpha_num"` |
| `alpha_dash` | `alpha_dash` | `"alpha_dash"` |
| `regex:pattern` | `regex:/pattern/` | `` "regex:^\d{4}$" `` |
| `gt:n` | `gt:n` | `"gt:0"` |
| `gte:n` | `gte:n` | `"gte:18"` |
| `lt:n` | `lt:n` | `"lt:100"` |
| `lte:n` | `lte:n` | `"lte:100"` |
| `nullable` | `nullable` | `"nullable\|min:10"` |
| `sometimes` | `sometimes` | `"sometimes\|required\|email"` |

### Working with Errors

```go
// Check if validation failed
if v.Fails() { ... }

// Check if validation passed
if v.Passes() { ... }

// Get all errors — map[string][]string
errs := v.Errors()

// Get first error for a field — like $errors->first('email')
msg := errs.First("email")

// Check if any errors exist — like $errors->any()
if errs.Has() { ... }
```

### Error Response Format

Mirrors Laravel's JSON validation error response exactly:

```json
{
  "errors": {
    "email": [
      "The email field is required.",
      "The email must be a valid email address."
    ],
    "password": [
      "The password must be at least 8 characters."
    ]
  }
}
```

---

## Responses

> Compatible with [Laravel Response docs](https://laravel.com/docs/responses).

### JSON Responses

```go
res := gohttp.NewResponse(w)

// Custom status + body
// Laravel: response()->json(['key' => 'val'], 200)
res.JSON(http.StatusOK, map[string]any{"key": "val"})

// 200 {"data": v}
// Laravel: response()->json(['data' => $resource])
res.Success(user)

// 201 {"data": v}
// Laravel: response()->json($resource, 201)
res.Created(newUser)

// 204 No Content
// Laravel: response()->noContent()
res.NoContent()
```

### Error Responses

```go
// Laravel: abort(400, 'Bad request')
res.Error(http.StatusBadRequest, "Bad request")

// Laravel: abort(401)
res.Unauthorized()
res.Unauthorized("Token expired.")

// Laravel: abort(403)
res.Forbidden()
res.Forbidden("Cannot access this resource.")

// Laravel: abort(404)
res.NotFound()
res.NotFound("User not found.")

// Laravel: abort(500)
res.ServerError()

// Laravel: validator->fails() → return response()->json($validator->errors(), 422)
res.ValidationError(v.Errors())
```

### Redirects

```go
// Laravel: redirect('/dashboard')
res.RedirectTo("/dashboard")

// Laravel: redirect()->back()
res.RedirectBack(r, "/fallback")

// Custom status code
// Laravel: redirect('/login', 301)
res.Redirect(http.StatusMovedPermanently, "/login")
```

---

## Views / Templates

```go
// Create engine (usually done in app bootstrap)
engine := gohttp.NewViewEngine("./views", ".html")

// Laravel: view('home', ['title' => 'Home'])
engine.View(w, "home", map[string]any{
    "title": "Welcome",
    "user":  currentUser,
})

// With a base layout
engine.ViewWithLayout(w, "layouts/app", "home", data)
```

Template (`views/home.html`):
```html
<!DOCTYPE html>
<html>
<head><title>{{ .title }}</title></head>
<body>
  <h1>Hello, {{ .user.Name }}!</h1>
</body>
</html>
```

---

## Controllers

Embed `app.Controller` to get `Request()` and `Response()` factory methods:

```go
type UserController struct {
    app.Controller
}

func (c *UserController) Store(w http.ResponseWriter, r *http.Request) {
    req := c.Request(r)
    res := c.Response(w)

    var input struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    if err := req.Bind(&input); err != nil {
        res.Error(http.StatusBadRequest, err.Error())
        return
    }

    v := validation.Make(req.All(), validation.Rules{
        "name":  "required|min:2",
        "email": "required|email",
    })
    if v.Fails() {
        res.ValidationError(v.Errors())
        return
    }

    // ... save to DB ...

    res.Created(map[string]any{
        "name":  input.Name,
        "email": input.Email,
    })
}
```

Register as a resource:

```go
r.Resource("/users", &UserController{})
```

---

## Running Tests

```bash
# All tests
go test ./...

# With verbose output
go test ./... -v

# Specific package
go test ./http/validation/... -v
go test ./routing/... -v
go test ./config/... -v
go test ./http/... -v

# With coverage
go test ./... -cover

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Roadmap

| Feature | Status |
|---------|--------|
| Collections | ✅ Done (go-collections) |
| .env Config | ✅ Done |
| Router (Chi) | ✅ Done |
| Request Binding | ✅ Done |
| Validation (25+ rules) | ✅ Done |
| Response Helpers | ✅ Done |
| Middleware | ✅ Done |
| Resource Controllers | ✅ Done |
| Views / Templates | ✅ Done |
| Database (GORM integration) | 🔜 Planned |
| Auth / JWT Middleware | 🔜 Planned |
| Migrations | 🔜 Planned |
| Events / Listeners | 🔜 Planned |
| Queue / Jobs | 🔜 Planned |
| Mail | 🔜 Planned |
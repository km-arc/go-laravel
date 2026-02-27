package http

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/km-arc/go-laravel/http/validation"
)

// ── Response ─────────────────────────────────────────────────────────────────

// Response wraps http.ResponseWriter with Laravel-style helpers.
type Response struct {
	w http.ResponseWriter
}

// NewResponse wraps a ResponseWriter.
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{w: w}
}

// Raw returns the underlying ResponseWriter.
func (res *Response) Raw() http.ResponseWriter { return res.w }

// ── JSON responses ────────────────────────────────────────────────────────────

// JSON sends a JSON response.
//
//	res.JSON(http.StatusOK, map[string]any{"message": "ok"})
func (res *Response) JSON(status int, data any) {
	res.w.Header().Set("Content-Type", "application/json")
	res.w.WriteHeader(status)
	_ = json.NewEncoder(res.w).Encode(data)
}

// Success sends 200 JSON: {"data": v}
func (res *Response) Success(v any) {
	res.JSON(http.StatusOK, envelope{"data": v})
}

// Created sends 201 JSON: {"data": v}
func (res *Response) Created(v any) {
	res.JSON(http.StatusCreated, envelope{"data": v})
}

// NoContent sends 204 with no body.
func (res *Response) NoContent() {
	res.w.WriteHeader(http.StatusNoContent)
}

// Error sends a JSON error response.
//
//	res.Error(http.StatusNotFound, "Resource not found")
func (res *Response) Error(status int, message string) {
	res.JSON(status, envelope{"message": message})
}

// Unauthorized sends 401.
func (res *Response) Unauthorized(message ...string) {
	msg := first(message, "Unauthenticated.")
	res.JSON(http.StatusUnauthorized, envelope{"message": msg})
}

// Forbidden sends 403.
func (res *Response) Forbidden(message ...string) {
	msg := first(message, "This action is unauthorized.")
	res.JSON(http.StatusForbidden, envelope{"message": msg})
}

// NotFound sends 404.
func (res *Response) NotFound(message ...string) {
	msg := first(message, "Not found.")
	res.JSON(http.StatusNotFound, envelope{"message": msg})
}

// ServerError sends 500.
func (res *Response) ServerError(message ...string) {
	msg := first(message, "Server Error.")
	res.JSON(http.StatusInternalServerError, envelope{"message": msg})
}

// ValidationError sends 422 with the standard Laravel error bag.
//
//	res.ValidationError(validator.Errors())
func (res *Response) ValidationError(errors *validation.Errors) {
	res.JSON(http.StatusUnprocessableEntity, errors)
}

// ── Redirects ────────────────────────────────────────────────────────────────

// Redirect performs an HTTP redirect.
//
//	res.Redirect(http.StatusFound, "/dashboard")
func (res *Response) Redirect(status int, url string) {
	http.Redirect(res.w, &http.Request{}, url, status)
}

// RedirectTo performs a 302 redirect.
func (res *Response) RedirectTo(url string) {
	res.w.Header().Set("Location", url)
	res.w.WriteHeader(http.StatusFound)
}

// RedirectBack redirects to the Referer header (or fallback URL).
func (res *Response) RedirectBack(r *http.Request, fallback string) {
	ref := r.Referer()
	if ref == "" {
		ref = fallback
	}
	res.w.Header().Set("Location", ref)
	res.w.WriteHeader(http.StatusFound)
}

// ── View / Templates ─────────────────────────────────────────────────────────

// ViewEngine holds a compiled template set.
type ViewEngine struct {
	dir string
	ext string
}

// NewViewEngine creates a ViewEngine.
// dir is the templates directory (e.g. "./views"), ext is the file extension (e.g. ".html").
func NewViewEngine(dir, ext string) *ViewEngine {
	return &ViewEngine{dir: dir, ext: ext}
}

// View renders a template file with data.
//
//	engine.View(res.Raw(), "home", map[string]any{"title": "Home"})
func (ve *ViewEngine) View(w http.ResponseWriter, name string, data any) {
	pattern := filepath.Join(ve.dir, name+ve.ext)
	tmpl, err := template.ParseFiles(pattern)
	if err != nil {
		http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
	}
}

// ViewWithLayout renders a template with a base layout.
func (ve *ViewEngine) ViewWithLayout(w http.ResponseWriter, layout, name string, data any) {
	layoutPath := filepath.Join(ve.dir, layout+ve.ext)
	viewPath := filepath.Join(ve.dir, name+ve.ext)
	tmpl, err := template.ParseFiles(layoutPath, viewPath)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, filepath.Base(layoutPath), data); err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

type envelope map[string]any

func first(ss []string, fallback string) string {
	if len(ss) > 0 && ss[0] != "" {
		return ss[0]
	}
	return fallback
}

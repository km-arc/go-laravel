package providers

import (
	"github.com/km-arc/go-laravel/framework/config"
	"github.com/km-arc/go-laravel/framework/container"
	gohttp "github.com/km-arc/go-laravel/framework/http"
	"github.com/km-arc/go-laravel/framework/routing"
)

// ── ConfigServiceProvider ─────────────────────────────────────────────────────

// ConfigServiceProvider loads the application configuration from .env and
// binds it into the container as "config".
//
// Bound abstracts:
//   - "config"  → *config.Config
//   - "app"     → *config.AppConfig  (alias shorthand)
//
// Laravel equivalent:
//
//	// Illuminate\Foundation\Bootstrap\LoadConfiguration
//	$app->singleton('config', fn() => new Repository($items));
type ConfigServiceProvider struct {
	container.BaseProvider
	EnvFiles []string
}

func (p *ConfigServiceProvider) Register(app *container.Container) {
	envFiles := p.EnvFiles
	app.Singleton("config", func(c *container.Container) any {
		return config.Load(envFiles...)
	})
	app.Alias("config", "configuration")
}

// ── RoutingServiceProvider ────────────────────────────────────────────────────

// RoutingServiceProvider registers the HTTP router.
//
// Bound abstracts:
//   - "router"  → *routing.Router
//
// Laravel equivalent:
//
//	// Illuminate\Routing\RoutingServiceProvider
//	$app->singleton('router', fn($app) => new Router($app['events'], $app));
type RoutingServiceProvider struct {
	container.BaseProvider
}

func (p *RoutingServiceProvider) Register(app *container.Container) {
	app.Singleton("router", func(c *container.Container) any {
		return routing.New()
	})
}

// ── ViewServiceProvider ───────────────────────────────────────────────────────

// ViewServiceProvider registers the template engine.
//
// Bound abstracts:
//   - "view"   → *gohttp.ViewEngine
//
// Configuration keys read from "config":
//   - view.dir (default: "./views")
//   - view.ext (default: ".html")
//
// Laravel equivalent:
//
//	// Illuminate\View\ViewServiceProvider
//	$app->singleton('view', fn($app) => new Factory(...));
type ViewServiceProvider struct {
	container.BaseProvider
	Dir string // template directory, default: "./views"
	Ext string // file extension,    default: ".html"
}

func (p *ViewServiceProvider) Register(app *container.Container) {
	dir := p.Dir
	if dir == "" {
		dir = "./views"
	}
	ext := p.Ext
	if ext == "" {
		ext = ".html"
	}

	app.Singleton("view", func(c *container.Container) any {
		return gohttp.NewViewEngine(dir, ext)
	})
}

package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/km-arc/go-laravel/framework/config"
	"github.com/km-arc/go-laravel/framework/container"
	gohttp "github.com/km-arc/go-laravel/framework/http"
	"github.com/km-arc/go-laravel/framework/providers"
	"github.com/km-arc/go-laravel/framework/routing"
)

// Application is the top-level application container.
// It embeds the IoC Container and ProviderRegistry so user code can
// call app.Bind(), app.Singleton(), app.Register() directly —
// exactly like $app in Laravel's bootstrap/app.php.
type Application struct {
	*container.Container
	Providers *container.ProviderRegistry
}

// New creates and bootstraps the application.
func New(envFiles ...string) *Application {
	c := container.New()
	registry := container.NewProviderRegistry(c)

	app := &Application{
		Container: c,
		Providers: registry,
	}

	// Register framework core providers (same order as Laravel)
	registry.Register(&providers.ConfigServiceProvider{EnvFiles: envFiles})
	registry.Register(&providers.RoutingServiceProvider{})
	registry.Register(&providers.ViewServiceProvider{})

	return app
}

// Register adds a ServiceProvider to the application.
func (a *Application) Register(provider container.ServiceProvider) {
	a.Providers.Register(provider)
}

// Boot runs the Boot() phase on all providers.
func (a *Application) Boot() {
	a.Providers.Boot()
}

// Config resolves *config.Config from the container.
func (a *Application) Config() *config.Config {
	return container.Resolve[*config.Config](a.Container, "config")
}

// Router resolves *routing.Router from the container.
func (a *Application) Router() *routing.Router {
	return container.Resolve[*routing.Router](a.Container, "router")
}

// Views resolves *gohttp.ViewEngine from the container.
func (a *Application) Views() *gohttp.ViewEngine {
	return container.Resolve[*gohttp.ViewEngine](a.Container, "view")
}

// Run boots the application (if needed) and starts the HTTP server.
func (a *Application) Run() {
	if !a.Providers.Booted() {
		a.Boot()
	}
	cfg := a.Config()
	router := a.Router()
	addr := ":" + cfg.App.Port
	fmt.Printf("🚀  %s running on http://localhost%s  [%s]\n",
		cfg.App.Name, addr, cfg.App.Env)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// Environment returns APP_ENV value.
func (a *Application) Environment() string { return a.Config().App.Env }
func (a *Application) IsLocal() bool       { return a.Environment() == "local" }
func (a *Application) IsProduction() bool  { return a.Environment() == "production" }
func (a *Application) IsTesting() bool     { return a.Environment() == "testing" }
func (a *Application) IsDebug() bool       { return a.Config().App.Debug }
func (a *Application) Version() string     { return "0.1.0" }

// Controller is an embeddable base for HTTP controllers.
type Controller struct{}

func (c *Controller) Request(r *http.Request) *gohttp.Request {
	return gohttp.NewRequest(r)
}
func (c *Controller) Response(w http.ResponseWriter) *gohttp.Response {
	return gohttp.NewResponse(w)
}
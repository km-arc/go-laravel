package app

import (
	"fmt"
	"log"
	"net/http"

	gohttp "github.com/km-arc/go-collections/framework/http"
	"github.com/km-arc/go-collections/framework/routing"
	"github.com/km-arc/go-laravel/config"
)

// Application is the top-level container — mirrors Laravel's Application.
type Application struct {
	Config *config.Config
	Router *routing.Router
	Views  *gohttp.ViewEngine
}

// New bootstraps the application.
//
//	app := app.New()
//	app.Router.Get("/", handler)
//	app.Run()
func New(envFiles ...string) *Application {
	cfg := config.Load(envFiles...)

	return &Application{
		Config: cfg,
		Router: routing.New(),
		Views:  gohttp.NewViewEngine("./views", ".html"),
	}
}

// Run starts the HTTP server on APP_PORT (default 8000).
func (a *Application) Run() {
	addr := ":" + a.Config.App.Port
	fmt.Printf("🚀  %s running on http://localhost%s  [%s]\n",
		a.Config.App.Name, addr, a.Config.App.Env)

	if err := http.ListenAndServe(addr, a.Router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// ── Controller base ───────────────────────────────────────────────────────────

// Controller is an embeddable base for all controllers,
// providing Req/Res factory methods.
type Controller struct{}

func (c *Controller) Request(r *http.Request) *gohttp.Request {
	return gohttp.NewRequest(r)
}

func (c *Controller) Response(w http.ResponseWriter) *gohttp.Response {
	return gohttp.NewResponse(w)
}

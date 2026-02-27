package main

import (
	"net/http"

	"github.com/km-arc/go-laravel/framework/app"
	"github.com/km-arc/go-laravel/framework/container"
	gohttp "github.com/km-arc/go-laravel/framework/http"
	"github.com/km-arc/go-laravel/framework/http/validation"
	"github.com/km-arc/go-laravel/framework/routing"
)

// ── Example: custom service provider ─────────────────────────────────────────

// Logger is a simple example service.
type Logger struct{ prefix string }

func (l *Logger) Info(msg string) { println("[" + l.prefix + "] " + msg) }

// LogServiceProvider mirrors Laravel's service provider pattern.
//
//	// Laravel:
//	// class LogServiceProvider extends ServiceProvider {
//	//     public function register(): void {
//	//         $this->app->singleton(Logger::class, fn() => new Logger);
//	//     }
//	// }
type LogServiceProvider struct{ container.BaseProvider }

func (p *LogServiceProvider) Register(app *container.Container) {
	app.Singleton("logger", func(c *container.Container) any {
		return &Logger{prefix: "APP"}
	})
}

func (p *LogServiceProvider) Boot(app *container.Container) {
	logger := container.Resolve[*Logger](app, "logger")
	logger.Info("Application booted ✅")
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	// 1. Create application (loads .env, registers core providers)
	//    Laravel: $app = require __DIR__.'/../bootstrap/app.php';
	application := app.New()

	// 2. Register your own providers
	//    Laravel: $app->register(new LogServiceProvider($app))
	application.Register(&LogServiceProvider{})

	// 3. Register routes (resolve router from container)
	//    Laravel: Route::get('/', ...)
	r := application.Router()

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		gohttp.NewResponse(w).Success(map[string]any{
			"app":     application.Config().App.Name,
			"version": application.Version(),
			"env":     application.Environment(),
		})
	})

	// 4. API prefix group
	r.Prefix("/api/v1", func(api *routing.Router) {

		api.Post("/users", func(w http.ResponseWriter, req *http.Request) {
			request := gohttp.NewRequest(req)
			res := gohttp.NewResponse(w)

			var body struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}
			if err := request.Bind(&body); err != nil {
				res.Error(http.StatusBadRequest, err.Error())
				return
			}

			v := validation.Make(map[string]string{
				"name":  body.Name,
				"email": body.Email,
			}, validation.Rules{
				"name":  "required|min:2|max:100",
				"email": "required|email",
			})

			if v.Fails() {
				res.ValidationError(v.Errors())
				return
			}

			// Resolve logger from container anywhere in your app
			logger := container.Resolve[*Logger](application.Container, "logger")
			logger.Info("Creating user: " + body.Name)

			res.Created(map[string]any{"name": body.Name, "email": body.Email})
		})

	})

	// 5. Boot + run
	//    Laravel: $kernel->handle(Request::capture())
	application.Run()
}
package container

// ── ServiceProvider interface ─────────────────────────────────────────────────

// ServiceProvider mirrors Laravel's Illuminate\Support\ServiceProvider.
//
// Every provider must implement at minimum Register().
// Boot() is called after ALL providers have been registered, making it safe
// to resolve other bindings inside Boot().
//
//	// Laravel:
//	// class AppServiceProvider extends ServiceProvider {
//	//     public function register(): void { $this->app->singleton(...); }
//	//     public function boot(): void     { /* use resolved services */ }
//	// }
//
//	type AppServiceProvider struct{ container.BaseProvider }
//
//	func (p *AppServiceProvider) Register(app *container.Container) {
//	    app.Singleton("logger", func(c *container.Container) any {
//	        return logging.New(container.Resolve[*config.Config](c, "config"))
//	    })
//	}
//
//	func (p *AppServiceProvider) Boot(app *container.Container) {
//	    logger := container.Resolve[*logging.Logger](app, "logger")
//	    logger.Info("Application booted")
//	}
type ServiceProvider interface {
	// Register binds services into the container.
	// Do NOT resolve other bindings here — use Boot() for that.
	Register(app *Container)

	// Boot is called after all providers are registered.
	// Safe to resolve and use any binding here.
	Boot(app *Container)

	// Provides returns the list of abstract keys this provider registers.
	// Used for deferred (lazy) provider loading.
	// Return nil / empty slice if the provider is always eager.
	//
	//	// Laravel: public function provides(): array { return [Cache::class]; }
	Provides() []string

	// IsDeferred returns true if this provider should be loaded lazily —
	// only when one of its Provides() abstracts is first resolved.
	//
	//	// Laravel: protected $defer = true;
	IsDeferred() bool
}

// ── BaseProvider ──────────────────────────────────────────────────────────────

// BaseProvider is an embeddable struct that provides no-op implementations
// of Boot(), Provides(), and IsDeferred().
// Embed it in your provider and only override what you need.
//
//	type MyProvider struct{ container.BaseProvider }
//	func (p *MyProvider) Register(app *container.Container) { ... }
type BaseProvider struct{}

func (p *BaseProvider) Boot(_ *Container)    {}
func (p *BaseProvider) Provides() []string   { return nil }
func (p *BaseProvider) IsDeferred() bool     { return false }

// ── ProviderRegistry ──────────────────────────────────────────────────────────

// ProviderRegistry manages registration and booting of ServiceProviders,
// including deferred (lazy) providers.
//
// It mirrors the behaviour of Laravel's Application::registerConfiguredProviders
// and Application::bootProviders.
type ProviderRegistry struct {
	app       *Container
	eager     []ServiceProvider
	deferred  map[string]ServiceProvider // abstract → provider
	booted    bool
	registered map[ServiceProvider]bool
}

// NewProviderRegistry creates a registry bound to app.
func NewProviderRegistry(app *Container) *ProviderRegistry {
	return &ProviderRegistry{
		app:        app,
		deferred:   make(map[string]ServiceProvider),
		registered: make(map[ServiceProvider]bool),
	}
}

// Register adds a provider and calls its Register() method (unless deferred).
//
//	// Laravel: $app->register(new AppServiceProvider($app))
func (r *ProviderRegistry) Register(provider ServiceProvider) {
	if r.registered[provider] {
		return
	}
	r.registered[provider] = true

	if provider.IsDeferred() {
		for _, abstract := range provider.Provides() {
			r.deferred[abstract] = provider
		}
		// Intercept Make() calls for deferred abstracts
		r.interceptDeferred(provider)
		return
	}

	provider.Register(r.app)
	r.eager = append(r.eager, provider)

	// If already booted, boot this provider immediately
	if r.booted {
		provider.Boot(r.app)
	}
}

// interceptDeferred registers a lazy binding for each deferred abstract.
// The first Make() call triggers real registration + boot.
func (r *ProviderRegistry) interceptDeferred(provider ServiceProvider) {
	for _, abstract := range provider.Provides() {
		abs := abstract // capture
		r.app.Bind(abs, func(c *Container) any {
			// Register for real on first use
			if !r.registered[provider] || r.deferred[abs] != nil {
				provider.Register(c)
				delete(r.deferred, abs)
				if r.booted {
					provider.Boot(c)
				}
			}
			return c.Make(abs)
		})
	}
}

// Boot calls Boot() on all eager providers.
// Must be called after ALL providers have been registered.
//
//	// Laravel: $app->boot()
func (r *ProviderRegistry) Boot() {
	if r.booted {
		return
	}
	r.booted = true
	for _, provider := range r.eager {
		provider.Boot(r.app)
	}
}

// Booted returns true if Boot() has been called.
func (r *ProviderRegistry) Booted() bool { return r.booted }

// Providers returns all registered eager providers.
func (r *ProviderRegistry) Providers() []ServiceProvider { return r.eager }

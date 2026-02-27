// Package container provides a Laravel-compatible IoC (Inversion of Control)
// container and Service Provider system for Go.
//
// # Overview
//
// The container manages the instantiation and lifecycle of your application's
// dependencies. It supports transient bindings, singletons, pre-built instances,
// aliases, tags, contextual bindings, and extension (decoration).
//
// It mirrors the public API of Laravel's Illuminate\Container\Container as
// closely as Go's type system allows. Because Go has no runtime constructor
// reflection, auto-wiring is replaced by explicit factory functions.
//
// # Container Lifecycle
//
//  1. Create: c := container.New()
//  2. Register providers: registry.Register(&MyProvider{})
//  3. Boot: registry.Boot()        — safe to resolve everything after this
//  4. Serve requests
//
// # Bindings
//
//	// Transient — new instance every Make()
//	// Laravel: $app->bind(Foo::class, fn($app) => new Foo)
//	c.Bind("Foo", func(c *container.Container) any { return &Foo{} })
//
//	// Singleton — created once, reused
//	// Laravel: $app->singleton(Cache::class, fn($app) => new RedisCache)
//	c.Singleton("cache", func(c *container.Container) any {
//	    cfg := container.Resolve[*Config](c, "config")
//	    return cache.NewRedis(cfg)
//	})
//
//	// Pre-built value
//	// Laravel: $app->instance(Config::class, $config)
//	c.Instance("config", myConfig)
//
//	// Alias
//	// Laravel: $app->alias(Cache::class, 'cache')
//	c.Alias("cache", "cacheManager")
//
// # Resolving
//
//	// Untyped
//	// Laravel: $app->make(Cache::class)
//	raw := c.Make("cache")
//
//	// Generic (preferred — no type assertion required)
//	cache := container.Resolve[*RedisCache](c, "cache")
//
// # Contextual Binding
//
//	// Laravel: $app->when(PhotoController::class)
//	//              ->needs(Filesystem::class)
//	//              ->give(fn() => new S3Filesystem)
//	c.When("PhotoController").
//	    Needs("Filesystem").
//	    Give(func(c *container.Container) any { return &S3Filesystem{} })
//
// # Tags
//
//	// Laravel: $app->tag([CpuReport::class, MemReport::class], 'reports')
//	c.Tag([]string{"CpuReport", "MemReport"}, "reports")
//	reports := c.Tagged("reports")  // []any
//
// # Extend / Decorate
//
//	// Laravel: $app->extend(Logger::class, fn($logger, $app) => new TimestampLogger($logger))
//	c.Extend("logger", func(instance any, c *container.Container) any {
//	    return &TimestampLogger{Inner: instance.(*Logger)}
//	})
//
// # Service Providers
//
//	type AppServiceProvider struct{ container.BaseProvider }
//
//	func (p *AppServiceProvider) Register(app *container.Container) {
//	    app.Singleton("mailer", func(c *container.Container) any {
//	        cfg := container.Resolve[*config.Config](c, "config")
//	        return mail.NewSMTP(cfg.Mail)
//	    })
//	}
//
//	func (p *AppServiceProvider) Boot(app *container.Container) {
//	    // safe to resolve other bindings here
//	}
//
//	registry := container.NewProviderRegistry(c)
//	registry.Register(&AppServiceProvider{})
//	registry.Boot()
//
// # Deferred Providers
//
//	type HeavyProvider struct{ container.BaseProvider }
//
//	func (p *HeavyProvider) IsDeferred() bool     { return true }
//	func (p *HeavyProvider) Provides() []string   { return []string{"heavy"} }
//	func (p *HeavyProvider) Register(app *container.Container) {
//	    app.Singleton("heavy", func(c *container.Container) any {
//	        return heavySetup() // only called on first app.Make("heavy")
//	    })
//	}
package container

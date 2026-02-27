# IoC Container & Service Providers

> Full documentation for `github.com/km-arc/go-laravel/framework/container`
>
> Mirrors [Laravel's Service Container](https://laravel.com/docs/container) and
> [Service Providers](https://laravel.com/docs/providers) as closely as Go allows.

---

## Table of Contents

1. [Introduction](#introduction)
2. [Basic Bindings](#basic-bindings)
3. [Resolving](#resolving)
4. [Singletons](#singletons)
5. [Instances](#instances)
6. [Aliases](#aliases)
7. [Contextual Binding](#contextual-binding)
8. [Tags](#tags)
9. [Extending Bindings](#extending-bindings)
10. [Callbacks & Events](#callbacks--events)
11. [Service Providers](#service-providers)
12. [Deferred Providers](#deferred-providers)
13. [Wiring into Application](#wiring-into-application)
14. [Laravel → Go Cheatsheet](#laravel--go-cheatsheet)

---

## Introduction

The IoC container is a tool for managing class dependencies and performing
dependency injection. In Laravel you write:

```php
// Laravel
$app->singleton(UserRepository::class, fn($app) => new EloquentUserRepository($app));
$repo = $app->make(UserRepository::class);
```

In Go, because there is no runtime constructor reflection, factories are
explicit closures — but the API is otherwise identical:

```go
// Go-Laravel
c.Singleton("UserRepository", func(c *container.Container) any {
    return &EloquentUserRepository{DB: container.Resolve[*gorm.DB](c, "db")}
})
repo := container.Resolve[*EloquentUserRepository](c, "UserRepository")
```

---

## Basic Bindings

### Bind (transient)

A new instance is created on every `Make()` call.

```go
// Laravel: $app->bind(Transistor::class, fn($app) => new Transistor($app->make(PodcastParser::class)))
c.Bind("Transistor", func(c *container.Container) any {
    parser := container.Resolve[*PodcastParser](c, "PodcastParser")
    return &Transistor{Parser: parser}
})
```

### Bind If Not Already Bound

```go
// Laravel: $app->bindIf(Transistor::class, fn($app) => new Transistor)
if !c.Bound("Transistor") {
    c.Bind("Transistor", func(c *container.Container) any {
        return &Transistor{}
    })
}
```

---

## Resolving

### Make

```go
// Laravel: $app->make(Transistor::class)
transistor := c.Make("Transistor")
```

### Resolve with generics (preferred)

Avoids type assertions:

```go
// Instead of: repo := c.Make("repo").(*UserRepository)
// Write:
repo := container.Resolve[*UserRepository](c, "repo")

// Safe version — returns (T, bool) without panic
repo, ok := container.MustResolve[*UserRepository](c, "repo")
```

---

## Singletons

The factory is called once; subsequent `Make()` calls return the cached instance.

```go
// Laravel: $app->singleton(Transistor::class, fn($app) => new Transistor)
c.Singleton("cache", func(c *container.Container) any {
    cfg := container.Resolve[*config.Config](c, "config")
    return cache.NewRedis(cfg.Cache.Host)
})
```

### Singleton If

```go
// Laravel: $app->singletonIf(Transistor::class, fn($app) => new Transistor)
if !c.Bound("cache") {
    c.Singleton("cache", factory)
}
```

---

## Instances

Register a pre-built object as a singleton.

```go
// Laravel: $app->instance(Config::class, $config)
cfg := config.Load()
c.Instance("config", cfg)
```

---

## Aliases

Register alternative names for an abstract.

```go
// Laravel: $app->alias(Cache::class, 'cache')
c.Alias("cache", "cacheManager")

// Both resolve to the same binding:
c.Make("cache")        // ✅
c.Make("cacheManager") // ✅
```

---

## Contextual Binding

Give different implementations to different consumers.

```go
// Laravel:
// $app->when(PhotoController::class)
//     ->needs(Filesystem::class)
//     ->give(fn() => new S3Filesystem);
//
// $app->when(VideoController::class)
//     ->needs(Filesystem::class)
//     ->give(fn() => new LocalFilesystem);

c.When("PhotoController").
    Needs("Filesystem").
    Give(func(c *container.Container) any {
        return filesystem.NewS3(container.Resolve[*config.Config](c, "config"))
    })

c.When("VideoController").
    Needs("Filesystem").
    Give(func(c *container.Container) any {
        return filesystem.NewLocal("/var/videos")
    })
```

### GiveValue — inject a scalar

```go
// Laravel: ->give('/tmp/photos')
c.When("ReportAggregator").
    Needs("storagePath").
    GiveValue("/tmp/reports")
```

---

## Tags

Group related bindings under a tag and resolve them all at once.

```go
// Laravel:
// $app->tag([CpuReport::class, MemoryReport::class], 'reports');
// $reports = $app->tagged('reports');

c.Tag([]string{"CpuReport", "MemReport", "DiskReport"}, "reports")

// Resolve all at once
for _, report := range c.Tagged("reports") {
    report.(Reporter).Generate()
}
```

---

## Extending Bindings

Decorate / wrap an already-resolved service.

```go
// Laravel:
// $app->extend(Service::class, fn($service, $app) => new DecoratedService($service));

c.Extend("logger", func(instance any, c *container.Container) any {
    inner := instance.(*Logger)
    return &TimestampLogger{Inner: inner}
})
```

`Extend` can be chained — each extender receives the output of the previous one.

---

## Callbacks & Events

### AfterResolving

Called every time any abstract is resolved from the container.

```go
// Laravel: $app->afterResolving(fn($obj, $app) => ...)
c.AfterResolving(func(abstract string, instance any) {
    log.Printf("resolved: %s", abstract)
})
```

### Rebinding

Called when an abstract is re-bound (useful for updating dependent singletons).

```go
// Laravel: $app->rebinding(UserRepository::class, fn($app, $repo) => ...)
c.Rebinding("db", func(newDB any) {
    // Update any service that holds a reference to "db"
    container.Resolve[*UserRepo](c, "userRepo").SetDB(newDB.(*gorm.DB))
})
```

---

## Service Providers

Service Providers are the central place to bootstrap your application services.
They have two lifecycle methods: `Register` and `Boot`.

```go
// Laravel:
// class AppServiceProvider extends ServiceProvider {
//     public function register(): void { ... }
//     public function boot(): void     { ... }
// }

type AppServiceProvider struct {
    container.BaseProvider // provides no-op Boot, Provides, IsDeferred
}

func (p *AppServiceProvider) Register(app *container.Container) {
    // ✅ ONLY bind things here — do NOT resolve other bindings
    app.Singleton("mailer", func(c *container.Container) any {
        cfg := container.Resolve[*config.Config](c, "config")
        return mail.NewSMTP(cfg.Mail)
    })
}

func (p *AppServiceProvider) Boot(app *container.Container) {
    // ✅ Safe to resolve and use any binding here
    // All providers have been registered before Boot() is called
    mailer := container.Resolve[*mail.SMTPMailer](app, "mailer")
    mailer.SetFromName(container.Resolve[*config.Config](app, "config").App.Name)
}
```

### Register the provider

```go
// Laravel: // bootstrap/app.php — Application::configure()->withProviders([...])
application := app.New()
application.Register(&AppServiceProvider{})
application.Register(&DatabaseServiceProvider{})
application.Boot()
application.Run()
```

---

## Deferred Providers

Mark a provider as deferred so it is only loaded when one of its abstracts
is first resolved. Good for heavy services (DB, queues, search).

```go
// Laravel:
// class HeavyServiceProvider extends ServiceProvider {
//     protected $defer = true;
//     public function provides(): array { return [HeavyService::class]; }
//     public function register(): void  { $this->app->singleton(HeavyService::class, ...); }
// }

type HeavyServiceProvider struct {
    container.BaseProvider
}

func (p *HeavyServiceProvider) IsDeferred() bool   { return true }
func (p *HeavyServiceProvider) Provides() []string { return []string{"heavy"} }

func (p *HeavyServiceProvider) Register(app *container.Container) {
    // Only called on first app.Make("heavy")
    app.Singleton("heavy", func(c *container.Container) any {
        return heavySetup() // expensive initialization
    })
}
```

---

## Wiring into Application

The `app.Application` embeds the container and provider registry, so you use
the same API from your bootstrap file:

```go
application := app.New()  // loads .env, registers Config/Router/View providers

// Register your providers
application.Register(&DatabaseServiceProvider{})
application.Register(&AuthServiceProvider{})
application.Register(&EventServiceProvider{})

// Define routes
r := application.Router()
r.Get("/", homeHandler)

// Boot all providers, start server
application.Run()
```

Resolve anything from the container at any point:

```go
// In a controller or middleware:
db     := container.Resolve[*gorm.DB](application.Container, "db")
mailer := container.Resolve[*mail.Mailer](application.Container, "mailer")
logger := container.Resolve[*Logger](application.Container, "logger")
```

---

## Laravel → Go Cheatsheet

| Laravel | Go-Laravel |
|---------|-----------|
| `$app->bind(Foo::class, fn($app) => new Foo)` | `c.Bind("Foo", func(c *container.Container) any { return &Foo{} })` |
| `$app->singleton(Foo::class, fn($app) => new Foo)` | `c.Singleton("Foo", func(c *container.Container) any { return &Foo{} })` |
| `$app->instance(Foo::class, $foo)` | `c.Instance("Foo", foo)` |
| `$app->make(Foo::class)` | `c.Make("Foo")` |
| `app(Foo::class)` | `container.Resolve[*Foo](c, "Foo")` |
| `$app->bound(Foo::class)` | `c.Bound("Foo")` |
| `$app->resolved(Foo::class)` | `c.Resolved("Foo")` |
| `$app->alias(Foo::class, 'foo')` | `c.Alias("Foo", "foo")` |
| `$app->tag([A::class, B::class], 'tag')` | `c.Tag([]string{"A", "B"}, "tag")` |
| `$app->tagged('tag')` | `c.Tagged("tag")` |
| `$app->extend(Foo::class, fn($foo,$app) => ...)` | `c.Extend("Foo", func(i any, c *container.Container) any { ... })` |
| `$app->when(A::class)->needs(B::class)->give(...)` | `c.When("A").Needs("B").Give(...)` |
| `$app->when(A::class)->needs(B::class)->give('/path')` | `c.When("A").Needs("B").GiveValue("/path")` |
| `$app->afterResolving(fn($obj,$app) => ...)` | `c.AfterResolving(func(abs string, inst any) { ... })` |
| `$app->rebinding(Foo::class, fn($app,$foo) => ...)` | `c.Rebinding("Foo", func(inst any) { ... })` |
| `$app->forgetInstance(Foo::class)` | `c.Forget("Foo")` |
| `$app->flush()` | `c.Flush()` |
| `ServiceProvider::register()` | `func (p *MyProvider) Register(app *container.Container)` |
| `ServiceProvider::boot()` | `func (p *MyProvider) Boot(app *container.Container)` |
| `protected $defer = true` | `func (p *MyProvider) IsDeferred() bool { return true }` |
| `public function provides()` | `func (p *MyProvider) Provides() []string { return []string{"foo"} }` |
| `$app->register(new MyProvider($app))` | `application.Register(&MyProvider{})` |
| `$app->boot()` | `application.Boot()` |
| `$app->booted()` | `application.Providers.Booted()` |
| `$app->environment()` | `application.Environment()` |
| `$app->isLocal()` | `application.IsLocal()` |
| `$app->isProduction()` | `application.IsProduction()` |
| `app()->version()` | `application.Version()` |

---

## Why No Auto-Wiring?

Laravel uses PHP's reflection API to introspect constructor parameters and
resolve them automatically. Go's `reflect` package can read struct fields and
method signatures at runtime, but there is no way to call a function with
automatically resolved arguments without either code generation or unsafe tricks.

The closest Go equivalent is [google/wire](https://github.com/google/wire),
which generates wiring code at compile time. This framework takes the explicit
factory approach (like Pimple, PHP-DI in manual mode), which is:

- ✅ Transparent — no magic
- ✅ Type-safe — generics catch mistakes at compile time  
- ✅ Fast — no reflection at resolve time
- ✅ Testable — swap any factory in tests with `c.Instance("db", mockDB)`

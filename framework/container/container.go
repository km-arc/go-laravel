package container

import (
	"fmt"
	"reflect"
	"sync"
)

// ── Binding types ─────────────────────────────────────────────────────────────

// Factory is a function that builds a concrete value from the container.
type Factory func(c *Container) any

// binding holds a registered factory and whether it is a singleton.
type binding struct {
	factory   Factory
	singleton bool
}

// extender wraps an already-resolved instance with decorator logic.
type extender func(instance any, c *Container) any

// ── Container ─────────────────────────────────────────────────────────────────

// Container is the IoC container — mirrors Laravel's Illuminate\Container\Container.
//
// It supports:
//   - Bind / Singleton / Instance / Alias
//   - Make / Resolve (generic)
//   - Tags (group multiple abstractions under one tag)
//   - Extend (decorate / wrap resolved instances)
//   - Contextual binding (when A needs B, give it C)
//   - Rebound callbacks
//   - Resolved event callbacks
type Container struct {
	mu sync.RWMutex

	// abstract → binding
	bindings map[string]*binding

	// abstract → resolved singleton instance
	instances map[string]any

	// alias → abstract (canonical key)
	aliases map[string]string

	// abstract → extender funcs
	extenders map[string][]extender

	// tag → []abstract
	tags map[string][]string

	// contextual: when[concrete][abstract] = factory
	contextual map[string]map[string]Factory

	// rebound callbacks: abstract → []func(any)
	reboundCallbacks map[string][]func(any)

	// resolved callbacks: []func(abstract, instance)
	afterResolving []func(string, any)

	// stack of abstracts currently being resolved (for contextual lookup)
	buildStack []string
}

// New creates an empty container.
func New() *Container {
	c := &Container{
		bindings:         make(map[string]*binding),
		instances:        make(map[string]any),
		aliases:          make(map[string]string),
		extenders:        make(map[string][]extender),
		tags:             make(map[string][]string),
		contextual:       make(map[string]map[string]Factory),
		reboundCallbacks: make(map[string][]func(any)),
	}
	// Bind the container to itself — like Laravel's $app->instance()
	c.Instance("container", c)
	return c
}

// ── Registration ──────────────────────────────────────────────────────────────

// Bind registers a transient (new instance each Make) factory.
//
//	// Laravel: $app->bind(UserRepository::class, fn($app) => new EloquentUserRepository($app))
//	c.Bind("UserRepository", func(c *container.Container) any {
//	    return &EloquentUserRepository{DB: Resolve[*gorm.DB](c, "db")}
//	})
func (c *Container) Bind(abstract string, factory Factory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bind(abstract, factory, false)
}

// Singleton registers a factory whose result is cached after first resolution.
//
//	// Laravel: $app->singleton(Cache::class, fn($app) => new RedisCache($app))
//	c.Singleton("cache", func(c *container.Container) any {
//	    return cache.NewRedisCache(Resolve[*config.Config](c, "config"))
//	})
func (c *Container) Singleton(abstract string, factory Factory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bind(abstract, factory, true)
}

// Instance registers a pre-built value as a singleton.
//
//	// Laravel: $app->instance(Config::class, $config)
//	c.Instance("config", myConfig)
func (c *Container) Instance(abstract string, instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.bindings, c.canonical(abstract))
	key := c.canonical(abstract)
	c.instances[key] = instance
	c.fireRebound(abstract, instance)
}

// bind is the internal registration helper (must hold mu.Lock).
func (c *Container) bind(abstract string, factory Factory, singleton bool) {
	key := c.canonical(abstract)

	// Drop existing singleton instance so it's rebuilt with the new factory
	wasBound := c.instances[key] != nil
	delete(c.instances, key)

	c.bindings[key] = &binding{factory: factory, singleton: singleton}

	if wasBound {
		c.mu.Unlock()
		c.fireRebound(abstract, c.make(abstract))
		c.mu.Lock()
	}
}

// Alias registers an alternative name for an abstract.
//
//	// Laravel: $app->alias(Cache::class, 'cache')
//	c.Alias("cache", "cacheManager")
func (c *Container) Alias(abstract, alias string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if abstract == alias {
		panic(fmt.Sprintf("container: [%s] is aliased to itself", abstract))
	}
	c.aliases[alias] = c.canonical(abstract)
}

// ── Contextual Binding ────────────────────────────────────────────────────────

// When starts a contextual binding chain.
//
//	// Laravel: $app->when(PhotoController::class)->needs(Filesystem::class)->give(fn() => new S3)
//	c.When("PhotoController").Needs("Filesystem").Give(func(c *container.Container) any {
//	    return filesystem.NewS3(...)
//	})
func (c *Container) When(concrete string) *ContextualBuilder {
	return &ContextualBuilder{container: c, concrete: concrete}
}

// getContextual returns the contextual factory for (concrete, abstract), or nil.
func (c *Container) getContextual(concrete, abstract string) Factory {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if m, ok := c.contextual[concrete]; ok {
		if f, ok := m[abstract]; ok {
			return f
		}
	}
	return nil
}

// ── Extend ────────────────────────────────────────────────────────────────────

// Extend decorates the resolved instance of an abstract.
//
//	// Laravel: $app->extend(Logger::class, fn($logger, $app) => new TimestampLogger($logger))
//	c.Extend("logger", func(instance any, c *container.Container) any {
//	    return logging.NewTimestampWrapper(instance.(*Logger))
//	})
func (c *Container) Extend(abstract string, fn extender) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.canonical(abstract)
	c.extenders[key] = append(c.extenders[key], fn)

	// If already resolved as singleton, re-apply extenders and refire rebound
	if inst, ok := c.instances[key]; ok {
		extended := c.applyExtenders(key, inst)
		c.instances[key] = extended
		c.mu.Unlock()
		c.fireRebound(abstract, extended)
		c.mu.Lock()
	}
}

// ── Tags ──────────────────────────────────────────────────────────────────────

// Tag associates multiple abstracts under a named group.
//
//	// Laravel: $app->tag([CpuReport::class, MemoryReport::class], 'reports')
//	c.Tag([]string{"CpuReport", "MemoryReport"}, "reports")
func (c *Container) Tag(abstracts []string, tag string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tags[tag] = append(c.tags[tag], abstracts...)
}

// Tagged resolves all abstracts registered under a tag.
//
//	// Laravel: $app->tagged('reports')
//	reports := c.Tagged("reports")  // []any
func (c *Container) Tagged(tag string) []any {
	c.mu.RLock()
	abstracts := c.tags[tag]
	c.mu.RUnlock()

	result := make([]any, 0, len(abstracts))
	for _, abs := range abstracts {
		result = append(result, c.make(abs))
	}
	return result
}

// ── Resolution ────────────────────────────────────────────────────────────────

// Make resolves an abstract from the container.
//
//	// Laravel: $app->make(UserRepository::class)
//	repo := c.Make("UserRepository")
func (c *Container) Make(abstract string) any {
	return c.make(abstract)
}

// make is the internal resolver (no outer lock — individual ops lock as needed).
func (c *Container) make(abstract string) any {
	key := c.canonical(abstract)

	// Check singleton instance cache
	c.mu.RLock()
	if inst, ok := c.instances[key]; ok {
		c.mu.RUnlock()
		return inst
	}
	c.mu.RUnlock()

	// Check contextual binding (look at current build stack top)
	if len(c.buildStack) > 0 {
		caller := c.buildStack[len(c.buildStack)-1]
		if f := c.getContextual(caller, abstract); f != nil {
			return c.runFactory(key, f, false)
		}
	}

	// Look up binding
	c.mu.RLock()
	b, ok := c.bindings[key]
	c.mu.RUnlock()

	if !ok {
		// No binding — try to return nil gracefully
		panic(fmt.Sprintf("container: no binding registered for [%s]", abstract))
	}

	return c.runFactory(key, b.factory, b.singleton)
}

// runFactory executes a factory, optionally caching the result.
func (c *Container) runFactory(key string, f Factory, singleton bool) any {
	c.buildStack = append(c.buildStack, key)

	instance := f(c)

	c.buildStack = c.buildStack[:len(c.buildStack)-1]

	// Apply extenders
	c.mu.RLock()
	exts := c.extenders[key]
	c.mu.RUnlock()
	if len(exts) > 0 {
		instance = c.applyExtenders(key, instance)
	}

	if singleton {
		c.mu.Lock()
		c.instances[key] = instance
		c.mu.Unlock()
	}

	c.fireAfterResolving(key, instance)
	return instance
}

func (c *Container) applyExtenders(key string, instance any) any {
	for _, ext := range c.extenders[key] {
		instance = ext(instance, c)
	}
	return instance
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// Bound returns true if an abstract has been registered.
//
//	// Laravel: $app->bound(UserRepository::class)
func (c *Container) Bound(abstract string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := c.canonical(abstract)
	_, hasBinding := c.bindings[key]
	_, hasInstance := c.instances[key]
	return hasBinding || hasInstance
}

// Resolved returns true if the abstract has been resolved at least once.
//
//	// Laravel: $app->resolved(Cache::class)
func (c *Container) Resolved(abstract string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := c.canonical(abstract)
	_, ok := c.instances[key]
	return ok
}

// Forget removes all registrations for an abstract (binding + instance).
//
//	// Laravel: $app->forgetInstance(Cache::class)
func (c *Container) Forget(abstract string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.canonical(abstract)
	delete(c.bindings, key)
	delete(c.instances, key)
}

// Flush resets the entire container.
func (c *Container) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bindings = make(map[string]*binding)
	c.instances = make(map[string]any)
	c.aliases = make(map[string]string)
	c.extenders = make(map[string][]extender)
	c.tags = make(map[string][]string)
	c.contextual = make(map[string]map[string]Factory)
}

// Bindings returns a copy of all registered abstract keys (for debugging).
func (c *Container) Bindings() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.bindings)+len(c.instances))
	for k := range c.bindings {
		out = append(out, k)
	}
	for k := range c.instances {
		if _, already := c.bindings[k]; !already {
			out = append(out, k)
		}
	}
	return out
}

// canonical resolves an alias to its canonical key.
func (c *Container) canonical(abstract string) string {
	if target, ok := c.aliases[abstract]; ok {
		return target
	}
	return abstract
}

// ── Callbacks ─────────────────────────────────────────────────────────────────

// Rebinding registers a callback to be called whenever an abstract is re-bound.
//
//	// Laravel: $app->rebinding(UserRepository::class, fn($app, $repo) => ...)
func (c *Container) Rebinding(abstract string, cb func(any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reboundCallbacks[abstract] = append(c.reboundCallbacks[abstract], cb)
}

// AfterResolving registers a callback fired after any abstract is resolved.
//
//	// Laravel: $app->afterResolving(fn($object, $app) => ...)
func (c *Container) AfterResolving(cb func(abstract string, instance any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.afterResolving = append(c.afterResolving, cb)
}

func (c *Container) fireRebound(abstract string, instance any) {
	c.mu.RLock()
	cbs := c.reboundCallbacks[abstract]
	c.mu.RUnlock()
	for _, cb := range cbs {
		cb(instance)
	}
}

func (c *Container) fireAfterResolving(abstract string, instance any) {
	c.mu.RLock()
	cbs := c.afterResolving
	c.mu.RUnlock()
	for _, cb := range cbs {
		cb(abstract, instance)
	}
}

// ── Reflect helpers ───────────────────────────────────────────────────────────

// TypeKey returns the package-qualified type name of v, useful as a stable
// abstract key when working with interfaces.
//
//	key := container.TypeKey((*UserRepository)(nil))  // "main.UserRepository"
//	c.Singleton(key, factory)
//	repo := container.Resolve[UserRepository](c, key)
func TypeKey(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// ── Generics helper ───────────────────────────────────────────────────────────

// Resolve is a generic helper that calls Make and type-asserts the result.
//
//	// Instead of: db := c.Make("db").(*gorm.DB)
//	// Write:      db := container.Resolve[*gorm.DB](c, "db")
func Resolve[T any](c *Container, abstract string) T {
	instance := c.Make(abstract)
	typed, ok := instance.(T)
	if !ok {
		panic(fmt.Sprintf("container: Resolve[%T]: [%s] resolved to %T", *new(T), abstract, instance))
	}
	return typed
}

// MustResolve is like Resolve but returns (T, bool) without panicking.
func MustResolve[T any](c *Container, abstract string) (T, bool) {
	instance := c.Make(abstract)
	typed, ok := instance.(T)
	return typed, ok
}
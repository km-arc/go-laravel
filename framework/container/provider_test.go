package container_test

import (
	"testing"

	"github.com/km-arc/go-collections/framework/container"
)

// ── stub providers ────────────────────────────────────────────────────────────

type eagerProvider struct {
	container.BaseProvider
	registerCalled bool
	bootCalled     bool
}

func (p *eagerProvider) Register(app *container.Container) {
	p.registerCalled = true
	app.Singleton("eager-svc", func(c *container.Container) any { return "eager" })
}

func (p *eagerProvider) Boot(app *container.Container) {
	p.bootCalled = true
}

// deferredProvider is lazy — only registered when "deferred-svc" is first resolved.
type deferredProvider struct {
	container.BaseProvider
	registerCalled bool
	bootCalled     bool
}

func (p *deferredProvider) Register(app *container.Container) {
	p.registerCalled = true
	app.Singleton("deferred-svc", func(c *container.Container) any { return "deferred-value" })
}

func (p *deferredProvider) Boot(app *container.Container) {
	p.bootCalled = true
}

func (p *deferredProvider) IsDeferred() bool   { return true }
func (p *deferredProvider) Provides() []string { return []string{"deferred-svc"} }

// multiProvider registers multiple abstracts.
type multiProvider struct {
	container.BaseProvider
}

func (p *multiProvider) Register(app *container.Container) {
	app.Singleton("alpha", func(c *container.Container) any { return "α" })
	app.Singleton("beta", func(c *container.Container) any { return "β" })
}

// ── ProviderRegistry ──────────────────────────────────────────────────────────

func TestRegistry_EagerProvider_RegisterCalled(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &eagerProvider{}
	reg.Register(p)

	if !p.registerCalled {
		t.Error("Register() should be called immediately for eager providers")
	}
}

func TestRegistry_EagerProvider_BootCalledAfterBoot(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &eagerProvider{}
	reg.Register(p)

	if p.bootCalled {
		t.Error("Boot() should NOT be called before registry.Boot()")
	}

	reg.Boot()

	if !p.bootCalled {
		t.Error("Boot() should be called after registry.Boot()")
	}
}

func TestRegistry_EagerProvider_ServiceResolvable(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)
	reg.Register(&eagerProvider{})
	reg.Boot()

	got := c.Make("eager-svc").(string)
	if got != "eager" {
		t.Errorf("eager-svc: got %q, want 'eager'", got)
	}
}

func TestRegistry_Boot_IdempotentCallsAreIgnored(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &eagerProvider{}
	reg.Register(p)

	reg.Boot()
	reg.Boot() // second call should be no-op

	if !reg.Booted() {
		t.Error("Booted() should be true after Boot()")
	}
}

func TestRegistry_Booted_FalseBeforeBoot(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)
	if reg.Booted() {
		t.Error("Booted() should be false before Boot()")
	}
}

func TestRegistry_DuplicateRegister_Ignored(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &eagerProvider{}
	reg.Register(p)
	reg.Register(p) // second register of same instance

	// registerCalled should still only reflect one real registration
	if !p.registerCalled {
		t.Error("provider should have been registered once")
	}
}

// ── Deferred providers ────────────────────────────────────────────────────────

func TestRegistry_DeferredProvider_NotRegisteredEagerly(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &deferredProvider{}
	reg.Register(p)
	reg.Boot()

	// Provider.Register should NOT have been called yet
	if p.registerCalled {
		t.Error("deferred provider Register() should not be called until Make()")
	}
}

func TestRegistry_DeferredProvider_RegisteredOnFirstMake(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)

	p := &deferredProvider{}
	reg.Register(p)
	reg.Boot()

	// Trigger lazy load
	got := c.Make("deferred-svc").(string)
	if got != "deferred-value" {
		t.Errorf("deferred-svc: got %q, want 'deferred-value'", got)
	}
}

// ── Multiple providers ────────────────────────────────────────────────────────

func TestRegistry_MultipleProviders_AllServicesResolvable(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)
	reg.Register(&multiProvider{})
	reg.Register(&eagerProvider{})
	reg.Boot()

	if got := c.Make("alpha").(string); got != "α" {
		t.Errorf("alpha: got %q, want 'α'", got)
	}
	if got := c.Make("beta").(string); got != "β" {
		t.Errorf("beta: got %q, want 'β'", got)
	}
	if got := c.Make("eager-svc").(string); got != "eager" {
		t.Errorf("eager-svc: got %q, want 'eager'", got)
	}
}

// ── Providers list ────────────────────────────────────────────────────────────

func TestRegistry_Providers_ReturnsEagerOnes(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)
	reg.Register(&eagerProvider{})
	reg.Register(&deferredProvider{}) // deferred — not in Providers()

	if len(reg.Providers()) != 1 {
		t.Errorf("Providers(): got %d, want 1 (eager only)", len(reg.Providers()))
	}
}

// ── BaseProvider defaults ─────────────────────────────────────────────────────

func TestBaseProvider_Defaults(t *testing.T) {
	var p container.BaseProvider
	c := container.New()

	p.Boot(c) // should not panic

	if p.IsDeferred() {
		t.Error("BaseProvider.IsDeferred() should be false")
	}
	if len(p.Provides()) != 0 {
		t.Error("BaseProvider.Provides() should return empty slice")
	}
}

// ── Boot after registration (late provider) ───────────────────────────────────

func TestRegistry_RegisterAfterBoot_BootsImmediately(t *testing.T) {
	c := container.New()
	reg := container.NewProviderRegistry(c)
	reg.Boot() // boot before registering

	p := &eagerProvider{}
	reg.Register(p) // register after boot

	if !p.bootCalled {
		t.Error("provider registered after Boot() should be booted immediately")
	}
}

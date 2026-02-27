package container

// ContextualBuilder implements the fluent contextual binding API.
//
//	// Laravel: $app->when(PhotoController::class)->needs(Filesystem::class)->give(...)
//	c.When("PhotoController").Needs("Filesystem").Give(func(c *container.Container) any {
//	    return filesystem.NewS3(...)
//	})
type ContextualBuilder struct {
	container *Container
	concrete  string
	needs     string
}

// Needs specifies which abstract the concrete type depends on.
func (b *ContextualBuilder) Needs(abstract string) *ContextualBuilder {
	b.needs = abstract
	return b
}

// Give provides the factory that should be used when the concrete type
// resolves the specified abstract.
func (b *ContextualBuilder) Give(factory Factory) {
	b.container.mu.Lock()
	defer b.container.mu.Unlock()

	if _, ok := b.container.contextual[b.concrete]; !ok {
		b.container.contextual[b.concrete] = make(map[string]Factory)
	}
	b.container.contextual[b.concrete][b.needs] = factory
}

// GiveValue is a shorthand for Give when the value is a simple scalar or
// pre-built instance (no factory logic needed).
//
//	// Laravel: ->give('/tmp/photos')
//	c.When("PhotoController").Needs("storagePath").GiveValue("/tmp/photos")
func (b *ContextualBuilder) GiveValue(value any) {
	b.Give(func(_ *Container) any { return value })
}
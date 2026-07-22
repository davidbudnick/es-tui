package service

// Container holds all service dependencies for the application.
type Container struct {
	Config ConfigService
	ES     ESService
}

// NewContainer creates a new service container with the provided services.
func NewContainer(config ConfigService, es ESService) *Container {
	return &Container{
		Config: config,
		ES:     es,
	}
}

// Close closes all services in the container.
func (c *Container) Close() error {
	var lastErr error

	if c.Config != nil {
		if err := c.Config.Close(); err != nil {
			lastErr = err
		}
	}

	if c.ES != nil {
		if err := c.ES.Disconnect(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

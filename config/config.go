package config

// Key uniquely identifies a configuration object in the configuration store.
type Key string

// IsNil returns true if the key is empty.
func (k Key) IsNil() bool {
	return k == ""
}

// Config provides an interface for configuration objects.
type Config interface {
	// Key returns the configuration key for this object.
	Key() Key
}

// Store provides a configuration store interface. Components can retrieve
// their configuration using their component keys.
type Store interface {
	// Open creates the resources like db connections or files required by the store.
	Open() error

	// Close releases any underlying resources used by the store.
	Close()

	// Get returns the configuration for the specified component or error if the
	// configuration is not found in the store.
	Get(Config) error
}

// BaseConfig provides a default implementation for Config interface.
type BaseConfig struct {
	ConfigKey Key
}

// Key returns the configuration key.
func (b *BaseConfig) Key() Key {
	return b.ConfigKey
}

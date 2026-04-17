package config

type Config struct {
}

func (c *Config) getPropertyConfiguration(name string) (map[string]string, error)

type Loader interface {
	Load() (Config, error)
}

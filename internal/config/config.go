package config

import (
	"github.com/BurntSushi/toml"
	"path"
)

type StubRouterConfig struct {
	Server struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
	}

	Session struct {
		Duration    string `toml:"duration"`
		IdleTimeout string `toml:"idle_timeout"`
		CookieName  string `toml:"cookie_name"`
		TokenSecret string `toml:"token_secret"`
		UseridField string `toml:"userid_field"`
	}

	Targets map[string]string `toml:"targets"`

	Stubs struct {
		Storage struct {
			Type  string `toml:"type"`
			Path  string `toml:"path"`
			Cache struct {
				Enabled            bool   `toml:"enabled"`
				ExpirationInterval string `toml:"expiration_interval"`
				CleanupInterval    string `toml:"cleanup_interval"`
			} `toml:"cache"`
		}
	}
}

var cfg *StubRouterConfig

// normalize fix params and set defaults
func normalize(cfg *StubRouterConfig) error {
	fixedTargets := make(map[string]string)

	for k, v := range cfg.Targets {
		fixedTargets[path.Clean("/"+k)] = v
	}
	cfg.Targets = fixedTargets

	return nil
}

func ParseConfig() (*StubRouterConfig, error) {
	_, err := toml.DecodeFile("config.toml", &cfg)
	if err != nil {
		return cfg, err
	}

	_ = normalize(cfg)
	return cfg, nil
}

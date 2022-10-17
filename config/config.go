package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type StubRouterConfig struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	}

	Session struct {
		Duration    string `yaml:"duration"`
		IdleTimeout string `yaml:"idle_timeout"`
		CookieName  string `yaml:"cookie_name"`
		TokenSecret string `yaml:"token_secret"`
		UseridField string `yaml:"userid_field"`
	}

	Targets map[string]string `yaml:"targets"`

	Stubs struct {
		Storage struct {
			Type  string `yaml:"type"`
			Path  string `yaml:"path"`
			Cache struct {
				Enabled            bool   `yaml:"enabled"`
				ExpirationInterval string `yaml:"expiration_interval"`
				CleanupInterval    string `yaml:"cleanup_interval"`
			} `yaml:"cache"`
		}
	}
}

var cfg *StubRouterConfig

func ParseConfig() (*StubRouterConfig, error) {
	configFile, err := os.Open("config.yml")
	if err != nil {
		return cfg, err
	}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

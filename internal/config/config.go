package config

import (
	"github.com/jessevdk/go-flags"
	"path"
)

type StubRouterConfig struct {
	Server struct {
		Host string `short:"h" long:"host" default:"0.0.0.0" description:"Listen host address"`
		Port int    `short:"p" long:"port" default:"3333" description:"Listen host port"`
	} `group:"server" namespace:"server"`

	Session struct {
		Duration    string `long:"duration" default:"24h" description:"Session duration in time.Duration format"`
		IdleTimeout string `long:"idle-timeout" default:"0h" description:"Session idle in time.Duration format"`
		CookieName  string `long:"cookie-name" default:"sessid" description:"Session cookie name"`
	} `group:"session" namespace:"session"`

	Auth struct {
		Enabled     bool   `long:"enabled" description:"Enable auth"`
		UseridField string `long:"user-field" description:"Auth user field in JWT token"`
	} `group:"auth" namespace:"auth"`

	Targets map[string]string `short:"t" long:"target" description:"Target pair target_path:target_host"`

	StubsStorage struct {
		Type  string `long:"type" default:"file" description:"Stub storage type: file, redis"`
		Path  string `long:"path" default:"." description:"Stub storage path: FS path, redis connect string"`
		Cache struct {
			Enabled            bool   `long:"enabled" description:"Cache stub in memory"`
			ExpirationInterval string `long:"expiration-interval" default:"30m" description:"Stub lifetime in cache"`
			CleanupInterval    string `long:"cleanup-interval" default:"60m" description:"Remove stub from cache after"`
		} `group:"cache" namespace:"cache"`
	} `group:"stubs" namespace:"stubs"`
}

// normalize fix params and set defaults
func normalize(cfg *StubRouterConfig) error {
	fixedTargets := make(map[string]string)

	for k, v := range cfg.Targets {
		fixedTargets[path.Clean("/"+k)] = v
	}
	cfg.Targets = fixedTargets

	return nil
}

func ParseConfig() (StubRouterConfig, error) {
	var cfg StubRouterConfig

	_, err := flags.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	_ = normalize(&cfg)
	return cfg, nil
}

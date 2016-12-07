package main

import "fmt"

// Env 環境參數
type Env struct {
	version     string `env:"-"`
	path        string `env:"-"`
	Debug       bool   `env:"DEBUG"`
	RedisServer string `env:"REDIS_SERVER"`
	AuthDSN     string `env:"AUTH_DSN"`
	EventDSN    string `env:"EVENT_DSN"`
	Addr        string `env:"ADDR"`
	GCDuration  string `env:"GC_DURATION"`
}

func (env *Env) String() string {
	return fmt.Sprintf(`events-driver %s
	path=%s
	DEBUG=%t
	AUTH_DSN=%s
	EVENT_DSN=%s
	ADDR=%s
	GC_DURATION=%s
	`,
		env.version,
		env.path,
		env.Debug,
		env.AuthDSN,
		env.EventDSN,
		env.Addr,
		env.GCDuration)
}

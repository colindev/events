package main

import "fmt"

// Env 環境參數
type Env struct {
	version string `env:"-"`
	path    string `env:"-"`
	// 除錯模式
	Debug bool `env:"DEBUG"`
	// 跟隨其他 events server
	Follow string `env:"FOLLOW"`
	// 存放登入連線 sqlite資料庫
	AuthDSN string `env:"AUTH_DSN"`
	// 存放事件流 sqlite資料庫
	EventDSN string `env:"EVENT_DSN"`
	// pub/sub 服務端口
	Addr string `env:"ADDR"`
	// 資料保留時數
	GCDuration string `env:"GC_DURATION"`
}

func (env *Env) String() string {
	return fmt.Sprintf(`events-driver %s
	path=%s
	DEBUG=%t
	FOLLOW=%s
	AUTH_DSN=%s
	EVENT_DSN=%s
	ADDR=%s
	GC_DURATION=%s
	`,
		env.version,
		env.path,
		env.Debug,
		env.Follow,
		env.AuthDSN,
		env.EventDSN,
		env.Addr,
		env.GCDuration)
}

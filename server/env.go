package main

import "github.com/colindev/osenv"

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
	return "\n" + osenv.ToString(env)
}

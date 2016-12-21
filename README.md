# Event Driver

Depend on Redis

[![Go Report Card](https://goreportcard.com/badge/github.com/colindev/events)](https://goreportcard.com/report/github.com/colindev/events)
[![Build Status](https://travis-ci.org/colindev/events.svg?branch=master)](https://travis-ci.org/colindev/events)

- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/event) event
- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/launcher) launcher 
- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/listener) listener 
- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/client) client
- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/server) server 
- [![GoDoc](https://godoc.org/github.com/colindev/events?status.svg)](https://godoc.org/github.com/colindev/events/cli) cli

### Server

```golang
./events-server -env [env file]
```

### Listaner

```golang

lis := listener.New(func()(client.Conn, error){ return client.Dial("[APP NAME]", "[HOST:PORT]")}).
    On(event.Event("prefix.*"), func(ev event.Event, rd event.RawData){
        // do some thing...
    }).
    On(event.Event("another.name"), func(ev event.Event, rd event.RawData){
        // do some thing for "another.name"
    })
    
lis.Run("prefix.*", "another.*")


```

### Launcher

```golang

lau := launcher.New(client.NewPool(func()(client.Conn, error){ return client.Dial("[APP NAME]", "[HOST:PORT]")}, maxIdleConn))
lau.Fire(event.Event("prefix.a"), event.RawData("......"))

```

### App name 行為

- `""` 空字串代表匿名連線, server 不會紀錄此連線的歷程
- 非空字串, 記名連線, server 會接收 client conn 的 recover 訊號重新發送上次斷線時間點的全部事件
  * TODO #29, #4


### 其他部份

https://github.com/colindev/events/milestone/1

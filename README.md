# Event Driver

Depend on Redis

### Listaner

```golang

lis := listener.New(redisPool).
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

lau := launcher.New(redisPool)
lau.Fire(event.Event("prefix.a"), event.RawData("......"))

```

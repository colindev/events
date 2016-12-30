### Change log

- 新增連線事件通知
    * Join store.Auth{...}
    * Leave store.Auth{...}

- events-cli 支援 -i 參數, 可取得server 當前所有連線狀態 
    * [ConnStatus](./server/conn.go)

- 重構: event 提供資料封箱與拆箱函式, 用以簡化 server/hub.publish server/conn.SendEvent 的複雜度 

- 修正: server/hub.handler 移除 for 循環內的 auth 處理


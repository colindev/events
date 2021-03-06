package store

import (
	"log"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	// 暫不開放自選 DSN
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Config 資料庫設定
type Config struct {
	AuthDSN      string
	EventDSN     string
	MaxIdleConns int
	MaxOpenConns int
	Debug        bool
	GCDuration   string
}

// Store 包含 登入者 跟 事件
type Store struct {
	wg     *sync.WaitGroup
	tk     *time.Ticker
	auth   *gorm.DB
	events *gorm.DB
	Events chan *Event
	quit   chan struct{}
}

// New return Store instance
func New(c Config) (*Store, error) {

	gcDuration, err := time.ParseDuration(c.GCDuration)
	if err != nil {
		return nil, err
	}

	auth, err := gorm.Open("sqlite3", c.AuthDSN)
	if err != nil {
		return nil, err
	}

	events, err := gorm.Open("sqlite3", c.EventDSN)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		auth = auth.Debug()
		events = events.Debug()
	}
	auth.AutoMigrate(Auth{})
	events.AutoMigrate(Event{})

	if c.MaxIdleConns > 0 {
		auth.DB().SetMaxIdleConns(c.MaxIdleConns)
		events.DB().SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.MaxOpenConns > 0 {
		auth.DB().SetMaxOpenConns(c.MaxOpenConns)
		events.DB().SetMaxOpenConns(c.MaxOpenConns)
	}

	eventChan := make(chan *Event, 100)
	store := &Store{
		wg:     &sync.WaitGroup{},
		auth:   auth.Model(Auth{}),
		events: events.Model(Event{}),
		Events: eventChan,
		quit:   make(chan struct{}),
		tk:     time.NewTicker(gcDuration),
	}

	store.wg.Add(2)
	go func() {
		defer store.wg.Done()
		for ev := range eventChan {
			if err := store.newEvent(ev); err != nil {
				log.Println("store event fail:", err)
			}
		}
	}()

	// GC
	go func() {
		defer store.wg.Done()
		for {
			select {
			case t := <-store.tk.C:
				until := t.Add(-gcDuration)
				timestamp := until.Unix()

				log.Printf("[store]: run gc until %s (%d)\n", until, timestamp)

				auth.Delete(Auth{}, "disconnected_at < ?", timestamp)
				events.Delete(Event{}, "received_at < ?", timestamp)
			case <-store.quit:
				return
			}
		}
	}()

	return store, nil
}

// Close db conn
func (s *Store) Close() {
	close(s.Events)
	s.tk.Stop()
	s.quit <- struct{}{}
	s.wg.Wait()
	s.auth.Close()
	s.events.Close()
}

// GetLast auth record
func (s *Store) GetLast(name string) (*Auth, error) {

	auth := Auth{
		Name: name,
	}

	if err := s.auth.Where(auth).Order("disconnected_at DESC").Limit(1).FirstOrInit(&auth).Error; err != nil {
		return nil, err
	}
	s.EachAuth(func(au *Auth) bool {
		auth = *au
		if auth.RecoverSince > 0 {
			return false
		}
		return true
	}, name)

	return &auth, nil
}

// NewAuth insert auth record
func (s *Store) NewAuth(auth *Auth) error {
	s.wg.Add(1)
	defer s.wg.Done()
	return s.auth.Create(auth).Error
}

// UpdateAuth record
func (s *Store) UpdateAuth(auth *Auth) error {
	s.wg.Add(1)
	defer s.wg.Done()
	return s.auth.Where(map[string]interface{}{
		"name":         auth.Name,
		"connected_at": auth.ConnectedAt,
	}).Update(auth).Error
}

// EachAuth callback
func (s *Store) EachAuth(f func(*Auth) bool, name string) error {
	offset := 0
	limit := 10

	db := s.auth.Where("name = ?", name).Limit(limit).Order("connected_at DESC")
	for {
		var list []*Auth
		ret := db.Offset(offset).Find(&list)
		if ret.Error != nil {
			return ret.Error
		}
		for _, au := range list {
			if !f(au) {
				return nil
			}
		}
		if len(list) < limit {
			return nil
		}
		offset += limit
	}
}

func (s *Store) newEvent(ev *Event) error {
	return s.events.Create(ev).Error
}

// EachEvents callback
func (s *Store) EachEvents(f func(*Event) error, prefix []string, since, until int64) error {

	offset := 0
	limit := 100

	db := s.events.Where("received_at >= ?", since).Limit(limit).Order("received_at ASC")
	if len(prefix) > 0 {
		db = db.Where("prefix IN (?)", prefix)
	}

	if until > 0 {
		db = db.Where("received_at <= ?", until)
	}

	defer func() {
		if v := recover(); v != nil {
			log.Println(v)
		}
	}()

	var err error
	for {
		var list []*Event
		ret := db.Offset(offset).Find(&list)
		if ret.Error != nil {
			return ret.Error
		}

		for _, ev := range list {
			if err = f(ev); err != nil {
				return err
			}
		}

		if len(list) < limit {
			break
		}
		offset += limit
	}

	return err
}

// Auth table struct
type Auth struct {
	Name           string `gorm:"column:name;index;size:40"`
	IP             string `gorm:"column:ip;size:30"`
	ConnectedAt    int64  `gorm:"column:connected_at"`
	DisconnectedAt int64  `gorm:"column:disconnected_at"`
	RecoverSince   int64  `gorm:"column:recover_since"`
	RecoverUntil   int64  `gorm:"column:recover_until"`
}

// TableName ...
func (Auth) TableName() string {
	return "auth"
}

// Event table struct
type Event struct {
	// hash(name:json)
	Hash string `gorm:"column:hash;primary_key;size:40"`
	// name = group.xxxx
	Name       string `gorm:"column:name;index;size:30"`
	Prefix     string `gorm:"column:prefix;index;size:15"`
	Length     int    `gorm:"column:length"`
	Raw        string `gorm:"column:raw;type:longtext"`
	ReceivedAt int64  `gorm:"column:received_at"`
}

// TableName ...
func (Event) TableName() string {
	return "events"
}

package store

import (
	"log"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Config struct {
	AuthDSN      string
	EventDSN     string
	MaxIdleConns int
	MaxOpenConns int
	Debug        bool
}

type Store struct {
	wg     *sync.WaitGroup
	auth   *gorm.DB
	events *gorm.DB
}

func New(c Config) (*Store, error) {

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

	return &Store{
		wg:     &sync.WaitGroup{},
		auth:   auth.Model(Auth{}),
		events: events.Model(Event{}),
	}, nil
}

func (s *Store) Close() {
	s.wg.Wait()
	s.auth.Close()
	s.events.Close()
}

func (s *Store) GetLast(name string) (*Auth, error) {
	s.wg.Add(1)
	defer s.wg.Done()

	auth := Auth{
		Name: name,
	}

	if err := s.auth.Where(auth).Order("disconnected_at DESC").Limit(1).FirstOrInit(&auth).Error; err != nil {
		return nil, err
	}

	return &auth, nil
}

func (s *Store) NewAuth(auth *Auth) error {
	s.wg.Add(1)
	defer s.wg.Done()
	return s.auth.Create(auth).Error
}

func (s *Store) UpdateAuth(auth *Auth) error {
	s.wg.Add(1)
	defer s.wg.Done()
	return s.auth.Where("name", auth.Name).Where("connected_at", auth.ConnectedAt).Update(auth).Error
}

func (s *Store) NewEvent(ev *Event) error {
	s.wg.Add(1)
	defer s.wg.Done()
	return s.events.Create(ev).Error
}

func (s *Store) EachEvents(f func(*Event), since int64, prefix []string) error {
	s.wg.Add(1)
	defer s.wg.Done()

	offset := 0
	limit := 100

	db := s.events.Limit(limit).Order("received_at ASC")
	if len(prefix) > 0 {
		db = db.Where("prefix IN (?)", prefix)
	}

	defer func() {
		if v := recover(); v != nil {
			log.Println(v)
		}
	}()

	for {
		var list []*Event
		ret := db.Where("received_at >= ?", since).Offset(offset).Find(&list)
		if ret.Error != nil {
			return ret.Error
		}

		for _, ev := range list {
			f(ev)
		}

		if len(list) < limit {
			break
		}
		offset += limit
	}

	return nil
}

type Auth struct {
	Name           string `gorm:"column:name;index;size:40"`
	IP             string `gorm:"column:ip;size:15"`
	ConnectedAt    int64  `gorm:"column:connected_at"`
	DisconnectedAt int64  `gorm:"column:disconnected_at"`
}

func (Auth) TableName() string {
	return "auth"
}

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

func (Event) TableName() string {
	return "events"
}

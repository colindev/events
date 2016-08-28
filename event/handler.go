package event

type (
	// Handler handle data from redis channel
	Handler func(Event, RawData)
)

package event

type (
	// Handler handle data from pub/sub channel
	Handler func(Event, RawData)
)

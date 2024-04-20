package event

import "time"

const (
	TypeConcert          string = "CONCERT"
	ShowTypeLive         string = "LIVE"
	ShowTypeHologramLive string = "HOLOGRAM_LIVE"
	ShowTypeOnline       string = "ONLINE"
)

type ShowLocation struct {
	Country          string
	City             string
	FormattedAddress string
	Latitude         float64
	Longitude        float64
}

type Show struct {
	ID               int64
	Venue            string
	Type             string
	TicketAllocation int64
	Location         *ShowLocation
	Time             time.Time
}

type Promotor struct {
	Name  string
	Email string
	Phone string
}

type Event struct {
	ID                    int64
	Name                  string
	Type                  string
	Promotors             []Promotor
	Artists               []string
	TotalTicketAllocation int64
	Shows                 []Show
	Description           string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

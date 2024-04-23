package ticket

import "time"

type TicketStock struct {
	EventID         string
	ShowID          string
	ID              string
	OnlineFor       *string
	Tier            string
	Allocation      int64
	Price           float64
	Acquired        int64
	LastStockUpdate time.Time
}

type AcquiredTicket struct {
	ID                   int64
	Number               string
	EventID              string
	ShowID               string
	Tier                 string
	TicketStockID        string
	EventName            string
	ShowVenue            string
	ShowType             string
	ShowCountry          string
	ShowCity             string
	ShowFormattedAddress string
	ShowTime             time.Time
	CustomerName         string
	CustomerEmail        string
	CustomerID           int64
	CreatedAt            time.Time
	OrderID              string
}

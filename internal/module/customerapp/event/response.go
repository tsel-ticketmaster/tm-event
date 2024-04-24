package event

import "time"

type PromotorResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type LocationResponse struct {
	Country          string  `json:"country"`
	City             string  `json:"city"`
	FormattedAddress string  `json:"formatted_address"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
}

type ShowResponse struct {
	ID       string            `json:"id"`
	Venue    string            `json:"venue"`
	Type     string            `json:"type"`
	Location *LocationResponse `json:"location"`
	Time     time.Time         `json:"time"`
	Status   string            `json:"status"`
}

type EventResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Status      string             `json:"status"`
	Promotors   []PromotorResponse `json:"promotors"`
	Artists     []string           `json:"artists"`
	Shows       []ShowResponse     `json:"shows,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func (r *EventResponse) PopulateFromEntity(e Event) {
	r.ID = e.ID
	r.Name = e.Name
	r.Description = e.Description
	r.Status = e.Status

	for _, v := range e.Promotors {
		r.Promotors = append(r.Promotors, PromotorResponse{
			Name:  v.Name,
			Email: v.Email,
			Phone: v.Phone,
		})
	}

	for _, v := range e.Artists {
		r.Artists = append(r.Artists, v.Name)
	}

	for _, v := range e.Shows {
		var location *LocationResponse
		if v.Location != nil {
			location = &LocationResponse{
				Country:          v.Location.Country,
				City:             v.Location.City,
				FormattedAddress: v.Location.FormattedAddress,
				Latitude:         v.Location.Latitude,
				Longitude:        v.Location.Longitude,
			}
		}
		r.Shows = append(r.Shows, ShowResponse{
			ID:       v.ID,
			Venue:    v.Venue,
			Type:     v.Type,
			Time:     v.Time,
			Status:   v.Status,
			Location: location,
		})
	}

	r.CreatedAt = e.CreatedAt
	r.UpdatedAt = e.UpdatedAt
}

type GetManyEventResponse struct {
	Total  int64           `json:"total"`
	Events []EventResponse `json:"events"`
}

type GetManyShowResponse struct {
	Shows []ShowResponse `json:"shows"`
}

type ShowTicketResponse struct {
	ID    string  `json:"id"`
	Tier  string  `json:"tier"`
	Stock int64   `json:"stock"`
	Price float64 `json:"price"`
}

type GetManyShowTicketsResponse struct {
	ShowTickets []ShowTicketResponse `json:"show_tickets"`
}

type AcquiredTicketResponse struct {
	ID                   int64     `json:"id"`
	Number               string    `json:"number"`
	EventID              string    `json:"event_id"`
	ShowID               string    `json:"show_id"`
	Tier                 string    `json:"tier"`
	TicketStockID        string    `json:"ticket_stock_id"`
	EventName            string    `json:"event_name"`
	ShowVenue            string    `json:"show_venue"`
	ShowType             string    `json:"show_type"`
	ShowCountry          string    `json:"show_country"`
	ShowCity             string    `json:"show_city"`
	ShowFormattedAddress string    `json:"show_formatted_address"`
	ShowTime             time.Time `json:"show_time"`
	CustomerName         string    `json:"customer_name"`
	CustomerEmail        string    `json:"customer_email"`
	CustomerID           int64     `json:"customer_id"`
	CreatedAt            time.Time `json:"created_at"`
	OrderID              string    `json:"order_id"`
}

type GetManyAcquiredTicketResponse struct {
	Total           int64                    `json:"total"`
	AcquiredTickets []AcquiredTicketResponse `json:"acquired_tickets"`
}

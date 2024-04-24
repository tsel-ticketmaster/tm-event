package event

type GetManyEventRequest struct {
	Page int `validate:"required"`
	Size int `validate:"required"`
}

type GetManyShowRequest struct {
	EventID string
}

type GetManyShowTicketsRequest struct {
	EventID string
	ShowID  string
}

type GetManyAcquiredTicketRequest struct {
	Page int
	Size int
}

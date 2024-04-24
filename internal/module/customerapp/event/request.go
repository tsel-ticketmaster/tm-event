package event

type GetManyEventRequest struct {
	Page int `validate:"required"`
	Size int `validate:"required"`
}

package event

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/middleware"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	publicMiddleware "github.com/tsel-ticketmaster/tm-event/pkg/middleware"
	"github.com/tsel-ticketmaster/tm-event/pkg/response"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
)

type HTTPHandler struct {
	SessionMiddleware *middleware.CustomerSession
	Validate          *validator.Validate
	EventUseCase      EventUseCase
}

func InitHTTPHandler(router *mux.Router, customerSession *middleware.CustomerSession, validate *validator.Validate, eventUsecase EventUseCase) {
	handler := &HTTPHandler{
		Validate:     validate,
		EventUseCase: eventUsecase,
	}

	router.HandleFunc("/tm-event/v1/customerapp/events", publicMiddleware.SetRouteChain(handler.GetManyEvent, customerSession.Verify)).Methods(http.MethodGet)
	router.HandleFunc("/tm-event/v1/customerapp/events/{eventID}/shows", publicMiddleware.SetRouteChain(handler.GetManyShow, customerSession.Verify)).Methods(http.MethodGet)
}

func (handler HTTPHandler) validate(ctx context.Context, payload interface{}) error {
	err := handler.Validate.StructCtx(ctx, payload)
	if err == nil {
		return nil
	}

	errorFields := err.(validator.ValidationErrors)

	errMessages := make([]string, len(errorFields))

	for k, errorField := range errorFields {
		errMessages[k] = fmt.Sprintf("invalid '%s' with value '%v'", errorField.Field(), errorField.Value())
	}

	errorMessage := strings.Join(errMessages, ", ")

	return fmt.Errorf(errorMessage)

}

func (handler HTTPHandler) GetManyEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := GetManyEventRequest{}

	qs := r.URL.Query()

	req.Page, _ = strconv.Atoi(qs.Get("page"))
	req.Size, _ = strconv.Atoi(qs.Get("size"))

	if err := handler.validate(ctx, req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.RESTEnvelope{
			Status:  status.BAD_REQUEST,
			Message: err.Error(),
		})

		return
	}

	resp, err := handler.EventUseCase.GetManyEvent(ctx, req)
	if err != nil {
		ae := errors.Destruct(err)
		response.JSON(w, ae.HTTPStatusCode, response.RESTEnvelope{
			Status:  ae.Status,
			Message: ae.Message,
		})

		return
	}
	response.JSON(w, http.StatusOK, response.RESTEnvelope{
		Status:  status.OK,
		Message: "list of event",
		Data:    resp,
		Meta:    nil,
	})

}

func (handler HTTPHandler) GetManyShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	req := GetManyShowRequest{
		EventID: vars["eventID"],
	}

	resp, err := handler.EventUseCase.GetManyShow(ctx, req)
	if err != nil {
		ae := errors.Destruct(err)
		response.JSON(w, ae.HTTPStatusCode, response.RESTEnvelope{
			Status:  ae.Status,
			Message: ae.Message,
		})

		return
	}
	response.JSON(w, http.StatusOK, response.RESTEnvelope{
		Status:  status.OK,
		Message: "list of show",
		Data:    resp,
		Meta:    nil,
	})
}

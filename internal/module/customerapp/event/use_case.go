package event

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tsel-ticketmaster/tm-event/internal/module/customerapp/ticket"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/util"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	"github.com/tsel-ticketmaster/tm-event/pkg/pubsub"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
	"golang.org/x/sync/errgroup"
)

type EventUseCase interface {
	OnOrderPaid(ctx context.Context, e OrderPaidEvent) error
	GetManyEvent(ctx context.Context, req GetManyEventRequest) (GetManyEventResponse, error)
	GetManyShow(ctx context.Context, req GetManyShowRequest) (GetManyShowResponse, error)
}

type eventUseCase struct {
	logger                   *logrus.Logger
	location                 *time.Location
	timeout                  time.Duration
	eventRepository          EventRepository
	artistRepository         ArtistRepository
	promotorRepository       PromotorRepository
	showRepository           ShowRepository
	locationRepository       LocationRepository
	ticketStockRepository    ticket.TicketStockRepository
	acquiredTicketRepository ticket.AcquiredTicketRepository
	publisher                pubsub.Publisher
}

type EventUseCaseProperty struct {
	Logger                   *logrus.Logger
	Location                 *time.Location
	Timeout                  time.Duration
	EventRepository          EventRepository
	ArtistRepository         ArtistRepository
	PromotorRepository       PromotorRepository
	ShowRepository           ShowRepository
	LocationRepository       LocationRepository
	TicketStockRepository    ticket.TicketStockRepository
	AcquiredTicketRepository ticket.AcquiredTicketRepository
	Publisher                pubsub.Publisher
}

func NewEventUseCase(props EventUseCaseProperty) EventUseCase {
	return &eventUseCase{
		logger:                   props.Logger,
		location:                 props.Location,
		timeout:                  props.Timeout,
		eventRepository:          props.EventRepository,
		artistRepository:         props.ArtistRepository,
		promotorRepository:       props.PromotorRepository,
		showRepository:           props.ShowRepository,
		locationRepository:       props.LocationRepository,
		ticketStockRepository:    props.TicketStockRepository,
		acquiredTicketRepository: props.AcquiredTicketRepository,
		publisher:                props.Publisher,
	}
}

// GetManyEvent implements EventUseCase.
func (u *eventUseCase) GetManyEvent(ctx context.Context, req GetManyEventRequest) (GetManyEventResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	offset := (req.Page - 1) * req.Size
	limit := req.Size

	var bunchOfEvents []Event
	var total int64

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		count, err := u.eventRepository.Count(gctx, nil)
		if err != nil {
			return err
		}
		total = count
		return nil
	})
	g.Go(func() error {
		events, err := u.eventRepository.FindMany(ctx, offset, limit, nil)
		if err != nil {
			return err
		}
		bunchOfEvents = events
		return nil
	})

	if err := g.Wait(); err != nil {
		return GetManyEventResponse{}, err
	}

	resp := GetManyEventResponse{
		Total:  total,
		Events: make([]EventResponse, len(bunchOfEvents)),
	}

	for k, v := range bunchOfEvents {
		bunchOfArtist, err := u.artistRepository.FindManyByEventID(ctx, v.ID, nil)
		if err != nil {
			return GetManyEventResponse{}, nil
		}

		bunchOfPromotors, err := u.promotorRepository.FindManyByEventID(ctx, v.ID, nil)
		if err != nil {
			return GetManyEventResponse{}, nil
		}

		v.Artists = bunchOfArtist
		v.Promotors = bunchOfPromotors

		e := EventResponse{}
		e.PopulateFromEntity(v)
		resp.Events[k] = e
	}

	return resp, nil
}

// GetManyShow implements EventUseCase.
func (u *eventUseCase) GetManyShow(ctx context.Context, req GetManyShowRequest) (GetManyShowResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	bunchOfShows, err := u.showRepository.FindManyByEventID(ctx, req.EventID, nil)
	if err != nil {
		return GetManyShowResponse{}, err
	}

	resp := GetManyShowResponse{
		Shows: make([]ShowResponse, len(bunchOfShows)),
	}

	for k, v := range bunchOfShows {
		location, err := u.locationRepository.FindByShowID(ctx, v.ID, nil)
		if err != nil {
			return GetManyShowResponse{}, err
		}

		lr := &LocationResponse{
			Country:          location.Country,
			City:             location.City,
			FormattedAddress: location.FormattedAddress,
			Latitude:         location.Latitude,
			Longitude:        location.Longitude,
		}
		sr := ShowResponse{
			ID:       v.ID,
			Venue:    v.Venue,
			Type:     v.Type,
			Location: lr,
			Time:     v.Time,
			Status:   v.Status,
		}

		resp.Shows[k] = sr
	}

	return resp, nil
}

// OnOrderPaid implements EventUseCase.
func (u *eventUseCase) OnOrderPaid(ctx context.Context, oe OrderPaidEvent) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	tx, err := u.eventRepository.BeginTx(ctx)
	if err != nil {
		return err
	}
	if len(oe.Items) < 1 {
		return errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "invalid items")
	}
	orderItem := oe.Items[0]

	e, err := u.eventRepository.FindByID(ctx, orderItem.EventID, tx)
	if err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	s, err := u.showRepository.FindByID(ctx, orderItem.ShowID, tx)
	if err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	loc, err := u.locationRepository.FindByShowID(ctx, orderItem.ShowID, tx)
	if err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	ts, err := u.ticketStockRepository.FindByIDForUpdate(ctx, orderItem.TicketStockID, tx)
	if err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	now := time.Now()
	ts.Acquired = ts.Acquired + orderItem.Quantity
	ts.LastStockUpdate = now

	if err := u.ticketStockRepository.Update(ctx, orderItem.TicketStockID, ts, tx); err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	aq := ticket.AcquiredTicket{
		Number:               util.GenerateUniqueID(util.UppercaseNumeric, 20),
		EventID:              e.ID,
		ShowID:               s.ID,
		Tier:                 ts.Tier,
		TicketStockID:        ts.ID,
		EventName:            e.Name,
		ShowVenue:            s.Venue,
		ShowType:             s.Type,
		ShowCountry:          loc.Country,
		ShowCity:             loc.City,
		ShowFormattedAddress: loc.FormattedAddress,
		ShowTime:             s.Time,
		CustomerName:         oe.CustomerName,
		CustomerEmail:        oe.CustomerEmail,
		CustomerID:           oe.CustomerID,
		OrderID:              oe.ID,
		CreatedAt:            now,
	}

	aqID, err := u.acquiredTicketRepository.Save(ctx, aq, tx)
	if err != nil {
		u.eventRepository.Rollback(ctx, tx)
		return err
	}

	u.eventRepository.CommitTx(ctx, tx)

	aq.ID = aqID

	aqBuff, _ := json.Marshal(aq)

	u.publisher.Publish(ctx, "acquire-ticket", aq.Number, nil, aqBuff)

	return nil
}

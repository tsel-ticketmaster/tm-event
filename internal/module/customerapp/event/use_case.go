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
)

type EventUseCase interface {
	OnOrderPaid(ctx context.Context, e OrderPaidEvent) error
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

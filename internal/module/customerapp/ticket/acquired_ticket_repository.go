package ticket

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
)

type AcquiredTicketRepository interface {
	Save(ctx context.Context, aq AcquiredTicket, tx *sql.Tx) (int64, error)
	FindManyByCustomerID(ctx context.Context, customerID int64, tx *sql.Tx) ([]AcquiredTicket, error)
}

type acquiredTicketRepository struct {
	logger *logrus.Logger
	db     *sql.DB
}

func NewAcquiredTicketRepository(logger *logrus.Logger, db *sql.DB) AcquiredTicketRepository {
	return &acquiredTicketRepository{
		logger: logger,
		db:     db,
	}
}

// FindByCustomerID implements AcquiredTicketRepository.
func (r *acquiredTicketRepository) FindManyByCustomerID(ctx context.Context, customerID int64, tx *sql.Tx) ([]AcquiredTicket, error) {
	panic("unimplemented")
}

// Save implements AcquiredTicketRepository.
func (r *acquiredTicketRepository) Save(ctx context.Context, aq AcquiredTicket, tx *sql.Tx) (int64, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		INSERT INTO acquired_ticket
		(
			"number", event_id, show_id, tier, ticket_stock_id, event_name, show_venue, show_type, show_country, show_city,
			show_formatted_address, show_time, customer_name, customer_email, customer_id, created_at, order_id
		)
		VALUES
		(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while saving acquired ticket's prorperties")
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, aq.Number, aq.EventID, aq.ShowID, aq.Tier, aq.TicketStockID,
		aq.EventName, aq.ShowVenue, aq.ShowType, aq.ShowCountry, aq.ShowCity, aq.ShowFormattedAddress,
		aq.ShowTime, aq.CustomerName, aq.CustomerEmail, aq.CustomerID, aq.CreatedAt, aq.OrderID,
	)
	var ID int64
	err = row.Scan(&ID)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while saving acquired ticket's prorperties")
	}

	return ID, nil
}

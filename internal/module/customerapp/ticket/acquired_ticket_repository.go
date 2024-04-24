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
	CountByCustomerID(ctx context.Context, customerID int64, tx *sql.Tx) (int64, error)
	FindManyByCustomerID(ctx context.Context, customerID int64, offset, limit int, tx *sql.Tx) ([]AcquiredTicket, error)
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
func (r *acquiredTicketRepository) CountByCustomerID(ctx context.Context, customerID int64, tx *sql.Tx) (int64, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `SELECT count(id) FROM acquired_ticket WHERE customer_id = $1`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while counting acquired ticket's prorperties")
	}
	defer stmt.Close()

	var count int64
	row := stmt.QueryRowContext(ctx, customerID)

	err = row.Scan(&count)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while counting acquired ticket's prorperties for update")
	}
	return count, nil
}

// FindByCustomerID implements AcquiredTicketRepository.
func (r *acquiredTicketRepository) FindManyByCustomerID(ctx context.Context, customerID int64, offset, limit int, tx *sql.Tx) ([]AcquiredTicket, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		SELECT 
			id, "number", event_id, show_id, tier, ticket_stock_id, event_name, show_venue, show_type, show_country, show_city,
			show_formatted_address, show_time, customer_name, customer_email, customer_id, created_at, order_id
		FROM acquired_ticket
		WHERE
			customer_id = $1
		ORDER BY id DESC
		OFFSET $2
		LIMIT $3
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of acquired ticket's prorperties")
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, customerID, offset, limit)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of acquired ticket's prorperties")
	}

	defer rows.Close()

	var data = make([]AcquiredTicket, 0)
	for rows.Next() {
		var aq AcquiredTicket
		err := rows.Scan(
			&aq.ID, &aq.Number, &aq.EventID, &aq.ShowID, &aq.Tier, &aq.TicketStockID,
			&aq.EventName, &aq.ShowVenue, &aq.ShowType, &aq.ShowCountry, &aq.ShowCity, &aq.ShowFormattedAddress,
			&aq.ShowTime, &aq.CustomerName, &aq.CustomerEmail, &aq.CustomerID, &aq.CreatedAt, &aq.OrderID,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error()
			return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of order rule day's prorperties")
		}

		data = append(data, aq)
	}

	return data, nil
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

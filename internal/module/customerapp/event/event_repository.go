package event

import (
	"context"
	"database/sql"
	"fmt"

	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
)

type EventRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CommitTx(ctx context.Context, tx *sql.Tx) error
	Rollback(ctx context.Context, tx *sql.Tx) error

	FindByID(ctx context.Context, ID string, tx *sql.Tx) (Event, error)
	FindMany(ctx context.Context, offset, limit int, tx *sql.Tx) ([]Event, error)
	Count(ctx context.Context, tx *sql.Tx) (int64, error)
}

type sqlCommand interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type eventRepository struct {
	logger *logrus.Logger
	db     *sql.DB
}

func NewEventRepository(logger *logrus.Logger, db *sql.DB) EventRepository {
	return &eventRepository{
		logger: logger,
		db:     db,
	}
}

// BeginTx implements EventRepository.
func (r *eventRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred trying to begin transaction")
	}

	return tx, nil
}

// CommitTx implements EventRepository.
func (r *eventRepository) CommitTx(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred trying to commit transaction")
	}

	return nil
}

// Rollback implements EventRepository.
func (r *eventRepository) Rollback(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred trying to rollback transaction")
	}

	return nil
}

// Count implements EventRepository.
func (r *eventRepository) Count(ctx context.Context, tx *sql.Tx) (int64, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `SELECT count(id) FROM event`
	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while counting bunch of event's prorperties")
	}
	defer stmt.Close()

	var count int64
	row := stmt.QueryRowContext(ctx)
	if err := row.Scan(&count); err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while counting bunch of event's prorperties")
	}

	return count, nil
}

// FindMany implements EventRepository.
func (r *eventRepository) FindMany(ctx context.Context, offset int, limit int, tx *sql.Tx) ([]Event, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		SELECT 
			id, name, description, status, created_at, updated_at
		FROM event
		ORDER BY id DESC
		OFFSET $1
		LIMIT $2
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of event's prorperties")
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, offset, limit)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of event's prorperties")
	}

	defer rows.Close()

	var bunchOfEvents = make([]Event, 0)
	for rows.Next() {
		var data Event
		err := rows.Scan(
			&data.ID, &data.Name, &data.Description, &data.Status, &data.CreatedAt, &data.UpdatedAt,
		)
		if err != nil {
			r.logger.WithContext(ctx).WithError(err).Error()
			return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of event's prorperties")
		}

		bunchOfEvents = append(bunchOfEvents, data)
	}

	return bunchOfEvents, nil
}

// FindByID implements EventRepository.
func (r *eventRepository) FindByID(ctx context.Context, ID string, tx *sql.Tx) (Event, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		SELECT 
			id, name, description, status, created_at, updated_at
		FROM event
		WHERE
			id = $1
		LIMIT 1
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return Event{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting event's prorperties")
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, ID)

	var data Event
	err = row.Scan(
		&data.ID, &data.Name, &data.Description, &data.Status, &data.CreatedAt, &data.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Event{}, errors.New(http.StatusNotFound, status.NOT_FOUND, fmt.Sprintf("event's properties with id '%s' is not found", ID))
		}
		r.logger.WithContext(ctx).WithError(err).Error()
		return Event{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting event's prorperties")
	}

	return data, nil
}

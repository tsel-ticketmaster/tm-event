package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/sirupsen/logrus"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
)

type CreatedAtFilter struct {
	From time.Time
	To   time.Time
}

type AdminRepositoryFilter struct {
	Status    *string
	Search    *string
	CreatedAt *CreatedAtFilter
	Offset    *int
	Limit     *int
	OrderBy   map[string]string
}

func NewAdminRepositoryFilter() *AdminRepositoryFilter {
	defaultOffset := 0
	defaultLimit := 10
	return &AdminRepositoryFilter{
		Offset: &defaultOffset,
		Limit:  &defaultLimit,
	}
}

func (f *AdminRepositoryFilter) SetStatus(status string) {
	if status == "" {
		return
	}
	f.Status = &status
}

func (f *AdminRepositoryFilter) SetSearch(search string) {
	f.Search = &search
}

func (f *AdminRepositoryFilter) SetOffset(offset int) {
	f.Offset = &offset
}

func (f *AdminRepositoryFilter) SetLimit(limit int) {
	f.Limit = &limit
}

func (f *AdminRepositoryFilter) SetRangeByCreatedAt(from, to time.Time) {
	f.CreatedAt = &CreatedAtFilter{
		From: from,
		To:   to,
	}
}

func (f *AdminRepositoryFilter) ToSQL() (string, []interface{}, error) {
	/**
	Example query with full where clauses:

	SELECT
		id, name, email, role, password, password_salt, status, created_at, updated_at
	FROM admin
	WHERE
		status = "ACTIVE"
	AND
		created_at >= '2024-01-01T00:00:00+07.00'
	AND
		created_at < '2024-01-31T23:59:59.999+07.00'
	ORDER BY
		created_at DESC
	OFFSET 0
	LIMIT 10
	*/

	builder := squirrel.Select("id, name, email, password, password_salt, status, created_at, updated_at").From("admin")
	if f.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": *f.Status})
	}

	if f.Search != nil {
		builder = builder.Where("to_tsvector(name) @@ to_tsquery(?)", *f.Search)
	}

	if f.CreatedAt != nil && f.Search == nil {
		builder = builder.
			Where(squirrel.GtOrEq{"created_at": f.CreatedAt.From}).
			Where(squirrel.Lt{"created_at": f.CreatedAt.To})
	}

	builder = builder.Offset(uint64(*f.Offset)).Limit(uint64(*f.Limit))

	if f.Search == nil {
		builder = builder.OrderBy("created_at", "DESC")
	}

	return builder.PlaceholderFormat(squirrel.Dollar).ToSql()
}

// AdminRepository is a set collection of behavior to store, update, and view admin's properties.
type AdminRepository interface {
	Save(context.Context, Administrator, *sql.Tx) (int64, error)
	Update(context.Context, int64, Administrator, *sql.Tx) error
	FindByID(context.Context, int64, *sql.Tx) (Administrator, error)
	FindByEmail(context.Context, string, *sql.Tx) (Administrator, error)
	FindMany(context.Context, AdminRepositoryFilter, *sql.Tx) ([]Administrator, error)
}

type sqlCommand interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

type adminRepository struct {
	logger *logrus.Logger
	db     *sql.DB
}

// NewAdminRepository acts like the constructor of AdminRepository. It returns collection of behaviors that implements the AdminRepository interface.
func NewAdminRepository(logger *logrus.Logger, db *sql.DB) AdminRepository {
	return &adminRepository{
		logger: logger,
		db:     db,
	}
}

// FindByID returns admin's properties and error. If the error returns in the tuppels, the admin's properties still exist with empty data or default value.
func (r *adminRepository) FindByID(ctx context.Context, ID int64, tx *sql.Tx) (Administrator, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		SELECT 
			id, name, email, password, password_salt, status, created_at, updated_at
		FROM admin
		WHERE
			id = $1
		LIMIT 1
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return Administrator{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting admin's prorperties")
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, ID)

	var data Administrator
	err = row.Scan(
		&data.ID, &data.Name, &data.Email, &data.Password, &data.PasswordSalt, &data.Status, &data.CreatedAt, &data.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Administrator{}, errors.New(http.StatusNotFound, status.NOT_FOUND, fmt.Sprintf("admin's properties with id '%d' is not found", ID))
		}
		r.logger.WithContext(ctx).WithError(err).Error()
		return Administrator{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting admin's prorperties")
	}

	return data, nil
}

// FindByEmail implements AdminRepository.
func (r *adminRepository) FindByEmail(ctx context.Context, email string, tx *sql.Tx) (Administrator, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		SELECT 
			id, name, email, password, password_salt, status, created_at, updated_at
		FROM admin
		WHERE
			email = $1
		LIMIT 1
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return Administrator{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting admin's prorperties")
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, email)

	var data Administrator
	err = row.Scan(
		&data.ID, &data.Name, &data.Email, &data.Password, &data.PasswordSalt, &data.Status, &data.CreatedAt, &data.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Administrator{}, errors.New(http.StatusNotFound, status.NOT_FOUND, fmt.Sprintf("admin's properties with email '%s' is not found", email))
		}
		r.logger.WithContext(ctx).WithError(err).Error()
		return Administrator{}, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting admin's prorperties")
	}

	return data, nil
}

// FindMany returns collection of admin's slice and error. If the list of the slice is empty, the error is still be nil or no error will be returned.
func (r *adminRepository) FindMany(ctx context.Context, filter AdminRepositoryFilter, tx *sql.Tx) ([]Administrator, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query, args, _ := filter.ToSQL()

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of admins' prorperties")
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of admins' prorperties")
	}
	defer rows.Close()

	var bunchOfDatas []Administrator
	for rows.Next() {
		var data Administrator
		if err := rows.Scan(
			&data.ID, &data.Name, &data.Password, &data.PasswordSalt, &data.Status, &data.CreatedAt, &data.UpdatedAt,
		); err != nil {
			r.logger.WithContext(ctx).WithError(err).Error()
			return nil, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while getting bunch of admins' prorperties")
		}

		bunchOfDatas = append(bunchOfDatas, data)
	}

	return bunchOfDatas, nil
}

// Save creates new admin's property. It also returns error if any problems are occured.
func (r *adminRepository) Save(ctx context.Context, data Administrator, tx *sql.Tx) (int64, error) {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		INSERT INTO admin
		(
			name, email, password, password_salt, status, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		RETURNING id
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while saving admin's prorperties")
	}

	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, data.Name, data.Email, data.Password, data.PasswordSalt, data.Status, data.CreatedAt, data.UpdatedAt)

	var id int64

	err = row.Scan(&id)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return 0, errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while saving admin's prorperties")
	}

	return id, nil
}

// Update modifies the existing admin's property. It also returns error if any problems are occured.
func (r *adminRepository) Update(ctx context.Context, ID int64, data Administrator, tx *sql.Tx) error {
	var cmd sqlCommand = r.db

	if tx != nil {
		cmd = tx
	}

	query := `
		UPDATE admin
		SET
			name = $1,
			email = $2,
			password = $3,
			password_salt = $4,
			status = $5,
			updated_at = $6
		WHERE id = $7
	`

	stmt, err := cmd.PrepareContext(ctx, query)
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while updating admin's prorperties")
	}

	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, query, data.Name, data.Email, data.Password, data.PasswordSalt, data.Status, data.UpdatedAt, ID); err != nil {
		r.logger.WithContext(ctx).WithError(err).Error()
		return errors.New(http.StatusInternalServerError, status.INTERNAL_SERVER_ERROR, "an error occurred while updating admin's prorperties")
	}

	return nil
}

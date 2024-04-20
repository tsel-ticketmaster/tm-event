package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/jwt"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/session"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/util"
	"github.com/tsel-ticketmaster/tm-event/pkg/errors"
	"github.com/tsel-ticketmaster/tm-event/pkg/status"
)

type AdminUseCase interface {
	SignIn(context.Context, SignInRequest) (SignInResponse, error)
	Create(context.Context, CreateRequest) (CreateResponse, error)
	SignOut(context.Context) error
	// GetByID(context.Context, GetByIDRequest) (GetByIDResponse, error)
	// GetMany(context.Context, GetManyRequest) (GetManyResponse, error)
	// ChangeEmail(context.Context, ChangeEmailRequest) (ChangeEmailResponse, error)
	// ChangePassword(context.Context, ChangePasswordRequest) (ChangePassowrdResponse, error)
	// ChangeProfile(context.Context, ChangeProfileRequest) (ChangeProfileRequest, error)
}

type adminUseCase struct {
	logger          *logrus.Logger
	timeout         time.Duration
	jsonWebToken    *jwt.JSONWebToken
	session         session.Session
	adminRepository AdminRepository
}

type AdminUseCaseProperty struct {
	Logger          *logrus.Logger
	Timeout         time.Duration
	JSONWebToken    *jwt.JSONWebToken
	Session         session.Session
	AdminRepository AdminRepository
}

func NewAdminUseCase(props AdminUseCaseProperty) AdminUseCase {
	return adminUseCase{
		logger:          props.Logger,
		timeout:         props.Timeout,
		jsonWebToken:    props.JSONWebToken,
		session:         props.Session,
		adminRepository: props.AdminRepository,
	}
}

// Create will creates new administrator.
func (a adminUseCase) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	_, err := a.adminRepository.FindByEmail(ctx, req.Email, nil)
	if err == nil {
		return CreateResponse{}, errors.New(http.StatusConflict, status.ALREADY_EXIST, fmt.Sprintf("admin with email %s is already exist", req.Email))
	}

	if !errors.MatchStatus(err, status.NOT_FOUND) {
		return CreateResponse{}, err
	}

	now := time.Now()
	passwordSalt := util.GenerateRandomHEX(16)
	defaultPassword := "P@ssw0rd"
	hashedPassword := util.GenerateSecret(defaultPassword, passwordSalt, 32)

	newAdmin := Administrator{
		Name:         req.Name,
		Email:        req.Email,
		Password:     hashedPassword,
		PasswordSalt: passwordSalt,
		Status:       StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	id, err := a.adminRepository.Save(ctx, newAdmin, nil)
	if err != nil {
		return CreateResponse{}, err
	}

	resp := CreateResponse{
		ID: id,
	}

	return resp, nil
}

// SignIn will sign in the administrator and got token for the session.
func (a adminUseCase) SignIn(ctx context.Context, req SignInRequest) (SignInResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	admin, err := a.adminRepository.FindByEmail(ctx, req.Email, nil)
	if err != nil {
		if errors.MatchStatus(err, status.NOT_FOUND) {
			return SignInResponse{}, errors.New(http.StatusBadRequest, status.BAD_REQUEST, "invalid admin email or password")
		}
		return SignInResponse{}, err
	}

	hashPassword := util.GenerateSecret(req.Password, admin.PasswordSalt, 32)

	if hashPassword != admin.Password {
		return SignInResponse{}, errors.New(http.StatusBadRequest, status.BAD_REQUEST, "invalid admin email or password")
	}

	now := time.Now()
	expiresIn := time.Hour * 1
	expiresAt := now.Add(expiresIn)
	subject := fmt.Sprintf("admin:%d", admin.ID)
	userType := "ADMIN"

	claim := jwt.Claim{}
	claim.Subject = subject
	claim.IssuedAt = now.Unix()
	claim.ExpiresAt = expiresAt.Unix()
	claim.Name = admin.Name
	claim.Email = admin.Email
	claim.Type = userType
	claim.Issuer = "ticket-master"

	idToken, err := a.jsonWebToken.Sign(ctx, claim)
	if err != nil {
		a.logger.WithContext(ctx).WithError(err).Error()
		return SignInResponse{}, err
	}

	if err := a.session.Set(ctx, fmt.Sprintf("%s:%d", "admin", admin.ID), session.Account{
		ID:   admin.ID,
		Name: admin.Name,
		Type: userType,
	}, expiresIn); err != nil {
		return SignInResponse{}, err
	}

	resp := SignInResponse{
		Token:     idToken,
		ExpiresAt: expiresAt,
	}

	return resp, nil
}

// SignOut will sign out the administrator and kill the existing session.
func (a adminUseCase) SignOut(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	acc, err := session.GetAccountFromCtx(ctx)
	if err != nil {
		return err
	}

	_, err = a.adminRepository.FindByID(ctx, acc.ID, nil)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("admin:%d", acc.ID)
	if err := a.session.Delete(ctx, key); err != nil {
		return err
	}

	return nil
}

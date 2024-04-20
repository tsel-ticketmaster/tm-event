package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/tsel-ticketmaster/tm-event/config"
	"github.com/tsel-ticketmaster/tm-event/internal/module/adminapp/admin"
	"github.com/tsel-ticketmaster/tm-event/internal/module/customerapp/customer"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/jwt"
	internalMiddleare "github.com/tsel-ticketmaster/tm-event/internal/pkg/middleware"
	"github.com/tsel-ticketmaster/tm-event/internal/pkg/session"
	"github.com/tsel-ticketmaster/tm-event/pkg/applogger"
	"github.com/tsel-ticketmaster/tm-event/pkg/kafka"
	"github.com/tsel-ticketmaster/tm-event/pkg/middleware"
	"github.com/tsel-ticketmaster/tm-event/pkg/monitoring"
	"github.com/tsel-ticketmaster/tm-event/pkg/postgresql"
	"github.com/tsel-ticketmaster/tm-event/pkg/pubsub"
	"github.com/tsel-ticketmaster/tm-event/pkg/redis"
	"github.com/tsel-ticketmaster/tm-event/pkg/server"
	"github.com/tsel-ticketmaster/tm-event/pkg/validator"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var (
	c           *config.Config
	CustomerApp string
	AdminApp    string
)

func init() {
	c = config.Get()
	AdminApp = fmt.Sprintf("%s/%s", c.Application.Name, "adminapp")
	CustomerApp = fmt.Sprintf("%s/%s", c.Application.Name, "customerapp")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := applogger.GetLogrus()

	mon := monitoring.NewOpenTelemetry(
		c.Application.Name,
		c.Application.Environment,
		c.GCP.ProjectID,
	)

	mon.Start(ctx)

	validate := validator.Get()

	jsonWebToken := jwt.NewJSONWebToken(c.JWT.PrivateKey, c.JWT.PublicKey)

	psqldb := postgresql.GetDatabase()
	if err := psqldb.Ping(); err != nil {
		logger.WithContext(ctx).WithError(err).Error()
	}

	publisher := pubsub.PublisherFromConfluentKafkaProducer(logger, kafka.NewProducer())

	rc := redis.GetClient()
	if err := rc.Ping(context.Background()).Err(); err != nil {
		logger.WithContext(ctx).WithError(err).Error()
	}

	session := session.NewRedisSessionStore(logger, rc)

	adminSessionMiddleware := internalMiddleare.NewAdminSessionMiddleware(jsonWebToken, session)
	customerSessionMiddleware := internalMiddleare.NewCustomerSessionMiddleware(jsonWebToken, session)

	router := mux.NewRouter()
	router.Use(
		otelmux.Middleware(c.Application.Name),
		middleware.HTTPResponseTraceInjection,
		middleware.NewHTTPRequestLogger(logger, c.Application.Debug).Middleware,
	)

	// admin's app
	adminappAdminRepository := admin.NewAdminRepository(logger, psqldb)
	adminappAdminUseCase := admin.NewAdminUseCase(admin.AdminUseCaseProperty{
		Logger:          logger,
		Timeout:         c.Application.Timeout,
		JSONWebToken:    jsonWebToken,
		Session:         session,
		AdminRepository: adminappAdminRepository,
	})
	admin.InitHTTPHandler(router, adminSessionMiddleware, validate, adminappAdminUseCase)

	// customer's app
	customerappCustomerRepository := customer.NewCustomerRepository(logger, psqldb)
	customerappCustomerUseCase := customer.NewCustomerUseCase(customer.CustomerUseCaseProperty{
		AppName:            CustomerApp,
		Logger:             logger,
		Timeout:            c.Application.Timeout,
		TMUserBaseURL:      c.Application.TMUser.BaseURL,
		CryptoSecret:       c.Crypto.Secret,
		JSONWebToken:       jsonWebToken,
		Session:            session,
		Cache:              rc,
		Publisher:          publisher,
		CustomerRepository: customerappCustomerRepository,
	})
	customer.InitHTTPHandler(router, customerSessionMiddleware, validate, customerappCustomerUseCase)

	handler := middleware.SetChain(
		router,
		cors.New(cors.Options{
			AllowedOrigins:   c.CORS.AllowedOrigins,
			AllowedMethods:   c.CORS.AllowedMethods,
			AllowedHeaders:   c.CORS.AllowedHeaders,
			ExposedHeaders:   c.CORS.ExposedHeaders,
			MaxAge:           c.CORS.MaxAge,
			AllowCredentials: c.CORS.AllowCredentials,
		}).Handler,
	)

	srv := &server.Server{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%d", c.Application.Port),
			Handler: handler,
		},
		Logger: logger,
	}

	go func() {
		srv.ListenAndServe()
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	srv.Shutdown(ctx)
	publisher.Close()
	psqldb.Close()
	rc.Close()
	mon.Stop(ctx)
}

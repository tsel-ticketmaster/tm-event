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
	adminapp_event "github.com/tsel-ticketmaster/tm-event/internal/module/adminapp/event"
	adminapp_order "github.com/tsel-ticketmaster/tm-event/internal/module/adminapp/order"
	adminapp_ticket "github.com/tsel-ticketmaster/tm-event/internal/module/adminapp/ticket"
	customerapp_event "github.com/tsel-ticketmaster/tm-event/internal/module/customerapp/event"
	customerapp_ticket "github.com/tsel-ticketmaster/tm-event/internal/module/customerapp/ticket"
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
	adminappEventRepository := adminapp_event.NewEventRepository(logger, psqldb)
	adminappArtistRepository := adminapp_event.NewArtistRepository(logger, psqldb)
	adminappPromotorRepository := adminapp_event.NewPromotorRepository(logger, psqldb)
	adminappShowRepository := adminapp_event.NewShowRepository(logger, psqldb)
	adminappLocationRepository := adminapp_event.NewLocationRepository(logger, psqldb)
	adminappOrderRuleRangeDateRepository := adminapp_order.NewOrderRuleRangeDateRepository(logger, psqldb)
	adminappOrderRuleDayRepository := adminapp_order.NewOrderRuleDayRepository(logger, psqldb)
	adminappTicketStockRepository := adminapp_ticket.NewTicketStockRepository(logger, psqldb)
	adminappEventUseCase := adminapp_event.NewEventUseCase(adminapp_event.EventUseCaseProperty{
		Logger:                       logger,
		Location:                     c.Application.Timezone,
		Timeout:                      c.Application.Timeout,
		EventRepository:              adminappEventRepository,
		ArtistRepository:             adminappArtistRepository,
		PromotorRepository:           adminappPromotorRepository,
		ShowRepository:               adminappShowRepository,
		LocationRepository:           adminappLocationRepository,
		OrderRuleDayRepository:       adminappOrderRuleDayRepository,
		OrderRuleRangeDateRepository: adminappOrderRuleRangeDateRepository,
		TicketStockRepository:        adminappTicketStockRepository,
	})
	adminapp_event.InitHTTPHandler(router, adminSessionMiddleware, validate, adminappEventUseCase)

	// customer's app
	customerappEventRepo := customerapp_event.NewEventRepository(logger, psqldb)
	customerappShowRepo := customerapp_event.NewShowRepository(logger, psqldb)
	customerappLocationRepo := customerapp_event.NewLocationRepository(logger, psqldb)
	customerappArtistRepo := customerapp_event.NewArtistRepository(logger, psqldb)
	customerappPromotorRepo := customerapp_event.NewPromotorRepository(logger, psqldb)
	customerappTicketStockRepo := customerapp_ticket.NewTicketStockRepository(logger, psqldb)
	customerappAcquiredTicketRepo := customerapp_ticket.NewAcquiredTicketRepository(logger, psqldb)
	customerappEventUseCase := customerapp_event.NewEventUseCase(customerapp_event.EventUseCaseProperty{
		Logger:                   logger,
		Location:                 c.Application.Timezone,
		Timeout:                  c.Application.Timeout,
		EventRepository:          customerappEventRepo,
		ArtistRepository:         customerappArtistRepo,
		PromotorRepository:       customerappPromotorRepo,
		ShowRepository:           customerappShowRepo,
		LocationRepository:       customerappLocationRepo,
		TicketStockRepository:    customerappTicketStockRepo,
		AcquiredTicketRepository: customerappAcquiredTicketRepo,
		Publisher:                publisher,
	})
	customerapp_event.InitHTTPHandler(router, customerSessionMiddleware, validate, customerappEventUseCase)
	orderPaidSubscriber := pubsub.SubscriberFromConfluentKafkaConsumer(pubsub.ConfluentKafkaConsumerProperty{
		Logger: logger,
		Topic:  "order-paid",
		EventHandler: &customerapp_event.OrderPaidEventHandler{
			EventUseCase: customerappEventUseCase,
		},
		Consumer: kafka.NewConsumer(CustomerApp, true),
	})
	orderPaidSubscriber.Subscribe()

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
	orderPaidSubscriber.Close()
	publisher.Close()
	psqldb.Close()
	rc.Close()
	mon.Stop(ctx)
}

package service

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mcp-log/planning/internal/adapters/postgres"
	"github.com/mcp-log/planning/internal/adapters/publisher"
	"github.com/mcp-log/planning/internal/app/command"
	"github.com/mcp-log/planning/internal/app/query"
	"github.com/mcp-log/planning/internal/ports"
)

// Service holds all wired dependencies for the Planning bounded context.
type Service struct {
	Handler   *ports.HTTPHandler
	Router    func() interface{} // Returns chi.Router via ports.NewRouter
	publisher *publisher.EventPublisher
}

// Config holds configuration for the service.
type Config struct {
	KafkaBrokers string
}

// New creates a fully wired Service with all dependencies.
func New(pool *pgxpool.Pool, cfg Config, logger *slog.Logger) *Service {
	// Adapters
	repo := postgres.NewPlanRepository(pool)
	pub := publisher.NewKafkaEventPublisher(cfg.KafkaBrokers, logger)

	// Command handlers
	createPlanHandler := command.NewCreatePlanHandler(repo, pub)
	addItemHandler := command.NewAddItemHandler(repo, pub)
	removeItemHandler := command.NewRemoveItemHandler(repo, pub)
	processPlanHandler := command.NewProcessPlanHandler(repo, pub)
	holdPlanHandler := command.NewHoldPlanHandler(repo, pub)
	resumePlanHandler := command.NewResumePlanHandler(repo, pub)
	releasePlanHandler := command.NewReleasePlanHandler(repo, pub)
	completePlanHandler := command.NewCompletePlanHandler(repo, pub)
	cancelPlanHandler := command.NewCancelPlanHandler(repo, pub)

	// Query handlers
	getPlanHandler := query.NewGetPlanHandler(repo)
	listPlansHandler := query.NewListPlansHandler(repo)

	// HTTP handler
	httpHandler := ports.NewHTTPHandler(
		createPlanHandler,
		addItemHandler,
		removeItemHandler,
		processPlanHandler,
		holdPlanHandler,
		resumePlanHandler,
		releasePlanHandler,
		completePlanHandler,
		cancelPlanHandler,
		getPlanHandler,
		listPlansHandler,
		logger,
	)

	return &Service{
		Handler:   httpHandler,
		publisher: pub,
	}
}

// Close gracefully shuts down service resources.
func (s *Service) Close() error {
	if s.publisher != nil {
		return s.publisher.Close()
	}
	return nil
}

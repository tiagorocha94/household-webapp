// Package server sets up the HTTP server, routing, and middleware.
package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	cashflowapi "github.com/tiagorocha94/household-webapp/internal/cashflow/api"
	creditsapi "github.com/tiagorocha94/household-webapp/internal/credits/api"
	goalsapi "github.com/tiagorocha94/household-webapp/internal/goals/api"
	investmentsapi "github.com/tiagorocha94/household-webapp/internal/investments/api"
	overviewapi "github.com/tiagorocha94/household-webapp/internal/overview/api"
	savingsapi "github.com/tiagorocha94/household-webapp/internal/savings/api"
)

// requestTimeout is how long a handler may take before the client gets a 503.
const requestTimeout = 10 * time.Second

// Options holds the optional settings of a Server.
type Options struct {
	// Frontend is the built frontend. When set, the server serves it at /
	// next to the API. Leave it nil for an API-only server.
	Frontend fs.FS
}

// Server holds the HTTP server dependencies and configuration.
type Server struct {
	logger             *slog.Logger
	version            string
	addr               string
	frontend           fs.FS
	router             *http.ServeMux
	handler            http.Handler
	authHandler        *authapi.Handler
	authService        *authservice.Service
	goalsHandler       *goalsapi.Handler
	creditsHandler     *creditsapi.Handler
	investmentsHandler *investmentsapi.Handler
	savingsHandler     *savingsapi.Handler
	cashflowHandler    *cashflowapi.Handler
	overviewHandler    *overviewapi.Handler
}

// New creates a new Server with the given dependencies.
func New(logger *slog.Logger, version, addr string, authService *authservice.Service, authHandler *authapi.Handler, goalsHandler *goalsapi.Handler, creditsHandler *creditsapi.Handler, investmentsHandler *investmentsapi.Handler, savingsHandler *savingsapi.Handler, cashflowHandler *cashflowapi.Handler, overviewHandler *overviewapi.Handler, opts Options) *Server {
	s := &Server{
		logger:             logger,
		version:            version,
		addr:               addr,
		frontend:           opts.Frontend,
		router:             http.NewServeMux(),
		authHandler:        authHandler,
		authService:        authService,
		goalsHandler:       goalsHandler,
		creditsHandler:     creditsHandler,
		investmentsHandler: investmentsHandler,
		savingsHandler:     savingsHandler,
		cashflowHandler:    cashflowHandler,
		overviewHandler:    overviewHandler,
	}

	s.routes()
	// Outermost first. recoverPanics sits inside withTimeout, which runs its
	// handler in another goroutine, so a panic is recovered where it happens
	// and the logged stack is the real one. The access log still sees the 500.
	s.handler = requestID(accessLog(logger, securityHeaders(limitBody(
		withTimeout(requestTimeout, recoverPanics(logger, s.router))))))

	return s
}

// Start serves HTTP requests until ctx is cancelled, then shuts down
// gracefully, giving in-flight requests up to ten seconds to finish.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.handler,
		ErrorLog:          slog.NewLogLogger(s.logger.Handler(), slog.LevelError),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down server: %w", err)
	}
	return nil
}

func (s *Server) routes() {
	// Health check for process managers and the frontend's dev server.
	s.router.HandleFunc("GET /api/v1/healthz", s.handleHealth)
	// Build information, also without a session.
	s.router.HandleFunc("GET /api/v1/info", s.handleInfo)

	// JSON API: authentication.
	s.router.HandleFunc("POST /api/v1/auth/login", s.authHandler.Login)
	s.router.HandleFunc("POST /api/v1/auth/logout", s.authHandler.Logout)
	s.router.Handle("GET /api/v1/auth/me", s.handle("get current user", s.authHandler.CurrentUser))

	// JSON API: devices. Enrolment needs the one-time code, not a session.
	s.router.HandleFunc("POST /api/v1/devices/enrol", s.authHandler.Enrol)

	// JSON API: read-only, authenticated with the session cookie. The goals,
	// credits, investments and savings routes (and the budget and own balance
	// below) also take a device token, so the phone can show them.
	s.router.Handle("GET /api/v1/goals/accounts", s.handleReadable("list goal accounts", s.goalsHandler.ListAccounts))
	s.router.Handle("GET /api/v1/goals/accounts/{id}", s.handleReadable("get goal account", s.goalsHandler.GetAccount))
	s.router.Handle("GET /api/v1/goals/accounts/{id}/goals", s.handleReadable("list goals", s.goalsHandler.ListGoals))
	s.router.Handle("GET /api/v1/goals/accounts/{id}/movements", s.handleReadable("list goal movements", s.goalsHandler.ListMovements))
	s.router.Handle("GET /api/v1/goals/accounts/{id}/projection", s.handleReadable("get goal projection", s.goalsHandler.GetProjection))
	s.router.Handle("GET /api/v1/goals/{id}", s.handleReadable("get goal", s.goalsHandler.GetGoal))

	s.router.Handle("GET /api/v1/overview", s.handle("get overview", s.overviewHandler.Get))
	s.router.Handle("GET /api/v1/overview/net-worth", s.handle("get net worth history", s.overviewHandler.GetNetWorth))

	s.router.Handle("GET /api/v1/credits", s.handleReadable("list credits", s.creditsHandler.ListContracts))
	s.router.Handle("GET /api/v1/credits/due", s.handleReadable("list credits due", s.creditsHandler.ListDue))
	s.router.Handle("GET /api/v1/credits/{id}", s.handleReadable("get credit", s.creditsHandler.GetContract))
	s.router.Handle("GET /api/v1/credits/{id}/payments", s.handleReadable("list credit payments", s.creditsHandler.ListPayments))

	s.router.Handle("GET /api/v1/investments/entities", s.handleReadable("list investment entities", s.investmentsHandler.ListEntities))
	s.router.Handle("GET /api/v1/investments/entities/{id}", s.handleReadable("get investment entity", s.investmentsHandler.GetEntity))
	s.router.Handle("GET /api/v1/investments/entities/{id}/instruments", s.handleReadable("list investment instruments", s.investmentsHandler.ListEntityInstruments))
	s.router.Handle("GET /api/v1/investments/instruments/{id}", s.handleReadable("get investment instrument", s.investmentsHandler.GetInstrument))
	s.router.Handle("GET /api/v1/investments/instruments/{id}/contributions", s.handleReadable("list investment contributions", s.investmentsHandler.ListContributions))
	s.router.Handle("GET /api/v1/investments/instruments/{id}/snapshots", s.handleReadable("list investment snapshots", s.investmentsHandler.ListSnapshots))

	s.router.Handle("GET /api/v1/savings/accounts", s.handleReadable("list savings accounts", s.savingsHandler.ListAccounts))
	s.router.Handle("GET /api/v1/savings/accounts/{id}", s.handleReadable("get savings account", s.savingsHandler.GetAccount))
	s.router.Handle("GET /api/v1/savings/accounts/{id}/movements", s.handleReadable("list savings movements", s.savingsHandler.ListMovements))
	s.router.Handle("GET /api/v1/savings/accounts/{id}/snapshots", s.handleReadable("list savings snapshots", s.savingsHandler.ListSnapshots))

	s.router.Handle("GET /api/v1/cashflow/categories", s.handle("list cashflow categories", s.cashflowHandler.ListCategories))
	s.router.Handle("GET /api/v1/cashflow/budget/{month}", s.handleReadable("get cashflow budget", s.cashflowHandler.GetBudget))
	s.router.Handle("GET /api/v1/cashflow/balance/{month}", s.handleReadable("get own cash balance", s.cashflowHandler.GetOwnBalance))
	s.router.Handle("GET /api/v1/cashflow/months", s.handle("list cashflow months", s.cashflowHandler.ListMonths))
	s.router.Handle("GET /api/v1/cashflow/months/{month}", s.handle("get cashflow month", s.cashflowHandler.GetMonth))
	s.router.Handle("GET /api/v1/cashflow/months/{month}/movements", s.handle("list cashflow movements", s.cashflowHandler.ListMonthMovements))

	// JSON API: the one write endpoint, for enrolled devices only (bearer
	// token, not the session cookie).
	s.router.Handle("POST /api/v1/sync/cashflow", s.authHandler.RequireDevice(s.adaptJSON("sync cashflow", s.cashflowHandler.Sync)))

	// Anything else is the frontend when there is one, and a JSON 404 or 405
	// otherwise, including for removed page routes.
	notFound := http.HandlerFunc(s.handleNotFound)
	if s.frontend == nil {
		s.router.Handle("/", notFound)
	} else {
		s.router.Handle("/", staticHandler(s.frontend, notFound))
	}
}

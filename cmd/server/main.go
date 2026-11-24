// Package main is the entry point for the household-app server.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/cashflow"
	cashflowapi "github.com/tiagorocha94/household-webapp/internal/cashflow/api"
	cashflowservice "github.com/tiagorocha94/household-webapp/internal/cashflow/service"
	cashflowstore "github.com/tiagorocha94/household-webapp/internal/cashflow/store"
	"github.com/tiagorocha94/household-webapp/internal/config"
	"github.com/tiagorocha94/household-webapp/internal/credits"
	creditsapi "github.com/tiagorocha94/household-webapp/internal/credits/api"
	creditsservice "github.com/tiagorocha94/household-webapp/internal/credits/service"
	creditsstore "github.com/tiagorocha94/household-webapp/internal/credits/store"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/database/migrations"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	goalsapi "github.com/tiagorocha94/household-webapp/internal/goals/api"
	goalsservice "github.com/tiagorocha94/household-webapp/internal/goals/service"
	goalsstore "github.com/tiagorocha94/household-webapp/internal/goals/store"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	investmentsapi "github.com/tiagorocha94/household-webapp/internal/investments/api"
	investmentsservice "github.com/tiagorocha94/household-webapp/internal/investments/service"
	investmentsstore "github.com/tiagorocha94/household-webapp/internal/investments/store"
	"github.com/tiagorocha94/household-webapp/internal/overview"
	overviewapi "github.com/tiagorocha94/household-webapp/internal/overview/api"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	savingsapi "github.com/tiagorocha94/household-webapp/internal/savings/api"
	savingsservice "github.com/tiagorocha94/household-webapp/internal/savings/service"
	savingsstore "github.com/tiagorocha94/household-webapp/internal/savings/store"
	"github.com/tiagorocha94/household-webapp/internal/seed"
	"github.com/tiagorocha94/household-webapp/internal/server"
	"github.com/tiagorocha94/household-webapp/web"
)

func main() {
	// "household-app device ..." manages the phones that write data.
	if len(os.Args) > 1 && os.Args[1] == "device" {
		if err := runDevice(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "household-app device:", err)
			os.Exit(1)
		}
		return
	}

	configPath := flag.String("config", "config.yaml", "path to the configuration file")
	showVersion := flag.Bool("version", false, "print the version and exit")
	dryRun := flag.Bool("dry-run", false, "report what starting would change in the database and CSV files, change nothing, and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildVersion())
		return
	}

	if err := run(*configPath, *dryRun); err != nil {
		slog.Error("household-app failed", "error", err)
		os.Exit(1)
	}
}

func run(configPath string, dryRun bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger, err := newLogger(os.Stderr, cfg.Log.Level)
	if err != nil {
		return err
	}
	logger.Info("starting household-app", "version", buildVersion())

	if dryRun {
		return previewStartup(os.Stdout, cfg.Database.Path, cfg.Seed.Dir, logger)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	if err := prepareData(db, cfg.Seed.Dir, logger); err != nil {
		return err
	}

	authStore := authstore.New(db)
	authService := authservice.New(authStore)
	authHandler := authapi.New(authService, logger, authapi.Options{SecureCookies: cfg.Server.SecureCookies})

	goalsStore := goalsstore.New(db)
	goalsService := goalsservice.New(goalsStore)
	goalsHandler := goalsapi.New(goalsService)

	creditsStore := creditsstore.New(db)
	creditsService := creditsservice.New(creditsStore)
	creditsHandler := creditsapi.New(creditsService)

	investmentsStore := investmentsstore.New(db)
	investmentsService := investmentsservice.New(investmentsStore)
	investmentsHandler := investmentsapi.New(investmentsService)
	savingsService := savingsservice.New(savingsstore.New(db))
	savingsHandler := savingsapi.New(savingsService)

	cashflowStore := cashflowstore.New(db)
	cashflowService := cashflowservice.New(cashflowStore)
	// Devices change movements while the server runs; keep the CSV file in step.
	cashflowService.EnableMirror(cashflow.NewCSVMirror(db, cfg.Seed.Dir), logger)
	// The detail of a movement is saved in the savings, investments and credits
	// tables; rewrite their files too.
	cashflowService.EnableDetailMirrors(csvstate.NewMirror(db, cfg.Seed.Dir, savings.SeedModule{}),
		csvstate.NewMirror(db, cfg.Seed.Dir, investments.SeedModule{}),
		csvstate.NewMirror(db, cfg.Seed.Dir, credits.SeedModule{}))
	cashflowHandler := cashflowapi.New(cashflowService)
	overviewHandler := overviewapi.New(
		overview.New(goalsService, creditsService, investmentsService, savingsService, cashflowService),
		overview.NewHistory(goalsService, creditsService, investmentsService, savingsService, cashflowService),
	)

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	srv := server.New(logger, buildVersion(), addr, authService, authHandler, goalsHandler, creditsHandler, investmentsHandler, savingsHandler, cashflowHandler, overviewHandler,
		server.Options{Frontend: frontend(logger)})

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("serve http: %w", err)
	}
	// The deferred close of the database runs after this line.
	logger.Info("server stopped")
	return nil
}

// prepareData brings the database up to date before the server starts:
// schema migrations, an idempotent import of the CSV files, sync (missing
// allocations and goal completions), and an export so that sync
// changes are persisted back to the CSV files. Any failure aborts startup,
// and export only runs after a successful import.
func prepareData(db *sql.DB, seedDir string, logger *slog.Logger) error {
	if err := migrations.Run(db, logger); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	runner := newSeedRunner(logger)
	if err := runner.Import(db, seedDir); err != nil {
		return fmt.Errorf("import seed data: %w", err)
	}
	if err := runner.Sync(db); err != nil {
		return fmt.Errorf("sync data: %w", err)
	}
	if err := runner.Export(db, seedDir); err != nil {
		return fmt.Errorf("export seed data: %w", err)
	}
	return nil
}

// newSeedRunner lists the modules that import, sync and export CSV data. The
// modules do not depend on each other's rows, except that investments,
// savings and cashflow name users from users.csv, so auth stays first.
func newSeedRunner(logger *slog.Logger) *seed.Runner {
	return seed.NewRunner(logger, auth.SeedModule{}, goals.SeedModule{}, credits.SeedModule{}, investments.SeedModule{}, savings.SeedModule{}, cashflow.SeedModule{})
}

// frontend returns the frontend built into the binary, or nil (the server then
// answers only the API) when it was never built.
func frontend(logger *slog.Logger) fs.FS {
	files := web.Dist()
	if _, err := fs.Stat(files, "index.html"); err != nil {
		logger.Warn("the frontend was not built into this binary, serving only the API; use make run")
		return nil
	}
	return files
}

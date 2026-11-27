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
	creditsdetail "github.com/tiagorocha94/household-webapp/internal/credits/detail"
	creditsservice "github.com/tiagorocha94/household-webapp/internal/credits/service"
	creditsstore "github.com/tiagorocha94/household-webapp/internal/credits/store"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/database/migrations"
	"github.com/tiagorocha94/household-webapp/internal/details"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	goalsapi "github.com/tiagorocha94/household-webapp/internal/goals/api"
	goalsservice "github.com/tiagorocha94/household-webapp/internal/goals/service"
	goalsstore "github.com/tiagorocha94/household-webapp/internal/goals/store"
	"github.com/tiagorocha94/household-webapp/internal/investments"
	investmentsapi "github.com/tiagorocha94/household-webapp/internal/investments/api"
	investmentsdetail "github.com/tiagorocha94/household-webapp/internal/investments/detail"
	investmentsservice "github.com/tiagorocha94/household-webapp/internal/investments/service"
	investmentsstore "github.com/tiagorocha94/household-webapp/internal/investments/store"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/overview"
	overviewapi "github.com/tiagorocha94/household-webapp/internal/overview/api"
	"github.com/tiagorocha94/household-webapp/internal/savings"
	savingsapi "github.com/tiagorocha94/household-webapp/internal/savings/api"
	savingsdetail "github.com/tiagorocha94/household-webapp/internal/savings/detail"
	savingsservice "github.com/tiagorocha94/household-webapp/internal/savings/service"
	savingsstore "github.com/tiagorocha94/household-webapp/internal/savings/store"
	"github.com/tiagorocha94/household-webapp/internal/scheduler"
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
	flag.Usage = func() { printUsage(flag.CommandLine.Output(), flag.CommandLine) }
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
	moduleSet := cfg.ModuleSet()
	logger.Info("starting household-app", "version", buildVersion(), "modules", moduleSet.Names())

	if dryRun {
		return previewStartup(os.Stdout, cfg.Database.Path, cfg.Seed.Dir, moduleSet, logger)
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

	if err := prepareData(db, cfg.Seed.Dir, moduleSet, logger); err != nil {
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
	// The detail of a movement is saved by the modules that run, in their own
	// tables; they rewrite their files too.
	detailRegistry, detailMirrors := detailModules(db, cfg.Seed.Dir, moduleSet)
	cashflowService.EnableDetails(detailRegistry, detailMirrors...)
	cashflowHandler := cashflowapi.New(cashflowService)
	overviewHandler := overviewapi.New(overviewSources(moduleSet, goalsService, creditsService, investmentsService, savingsService, cashflowService))

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	srv := server.New(logger, buildVersion(), addr, authService, authHandler, goalsHandler, creditsHandler, investmentsHandler, savingsHandler, cashflowHandler, overviewHandler,
		server.Options{Frontend: frontend(logger), Modules: moduleSet})

	// The background jobs stop with the server, and before the database closes.
	jobsCtx, stopJobs := context.WithCancel(ctx)
	jobsDone := make(chan struct{})
	go func() {
		defer close(jobsDone)
		scheduler.New(logger, backgroundJobs(db, cfg.Seed.Dir, moduleSet, authHandler, cfg.Backup, logger)...).Run(jobsCtx)
	}()
	err = srv.Start(ctx)
	stopJobs()
	<-jobsDone
	if err != nil {
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
func prepareData(db *sql.DB, seedDir string, enabled modules.Set, logger *slog.Logger) error {
	if err := migrations.Run(db, logger); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	runner := newSeedRunner(logger, enabled)
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

// newSeedRunner lists the modules that import, sync and export CSV data: the
// enabled feature modules, after authentication. The modules do not depend on
// each other's rows, except that investments, savings and cashflow name users
// from users.csv, so auth stays first. The files of a module that is not
// enabled are neither read nor written.
func newSeedRunner(logger *slog.Logger, enabled modules.Set) *seed.Runner {
	list := []seed.Module{auth.SeedModule{}}
	if enabled.Has(modules.Goals) {
		list = append(list, goals.SeedModule{})
	}
	if enabled.Has(modules.Credits) {
		list = append(list, credits.SeedModule{})
	}
	if enabled.Has(modules.Investments) {
		list = append(list, investments.SeedModule{})
	}
	if enabled.Has(modules.Savings) {
		list = append(list, savings.SeedModule{})
	}
	if enabled.Has(modules.Cashflow) {
		list = append(list, cashflow.SeedModule{})
	}
	return seed.NewRunner(logger, list...)
}

// detailModules returns the appliers of the modules that keep the detail of a
// cashflow movement, and a mirror for each one's files. A module that does not
// run is left out, so a device cannot send a detail for it.
func detailModules(db *sql.DB, seedDir string, enabled modules.Set) (*details.Registry, []cashflowservice.Mirror) {
	var appliers []details.Applier
	var mirrors []cashflowservice.Mirror
	if enabled.Has(modules.Savings) {
		appliers = append(appliers, savingsdetail.Applier{})
		mirrors = append(mirrors, csvstate.NewMirror(db, seedDir, savings.SeedModule{}))
	}
	if enabled.Has(modules.Investments) {
		appliers = append(appliers, investmentsdetail.Applier{})
		mirrors = append(mirrors, csvstate.NewMirror(db, seedDir, investments.SeedModule{}))
	}
	if enabled.Has(modules.Credits) {
		appliers = append(appliers, creditsdetail.Applier{})
		mirrors = append(mirrors, csvstate.NewMirror(db, seedDir, credits.SeedModule{}))
	}
	return details.NewRegistry(appliers...), mirrors
}

// overviewSources gives the overview and the net worth history the modules that
// run; the others are passed as untyped nils, which they read as absent.
func overviewSources(enabled modules.Set, g *goalsservice.Service, c *creditsservice.Service, i *investmentsservice.Service,
	sv *savingsservice.Service, cf *cashflowservice.Service) (*overview.Service, *overview.History) {
	var (
		og  overview.Goals
		hg  overview.HistoryGoals
		oc  overview.Credits
		hc  overview.HistoryCredits
		oi  overview.Investments
		hi  overview.HistoryInvestments
		osv overview.Savings
		hs  overview.HistorySavings
		ocf overview.Cashflow
		hf  overview.HistoryCashflow
	)
	if enabled.Has(modules.Goals) {
		og, hg = g, g
	}
	if enabled.Has(modules.Credits) {
		oc, hc = c, c
	}
	if enabled.Has(modules.Investments) {
		oi, hi = i, i
	}
	if enabled.Has(modules.Savings) {
		osv, hs = sv, sv
	}
	if enabled.Has(modules.Cashflow) {
		ocf, hf = cf, cf
	}
	return overview.New(og, oc, oi, osv, ocf), overview.NewHistory(hg, hc, hi, hs, hf)
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

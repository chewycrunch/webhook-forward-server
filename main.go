package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chewycrunch/webhook-forward-server/internal/config"
	"github.com/chewycrunch/webhook-forward-server/internal/discord"
	"github.com/chewycrunch/webhook-forward-server/internal/dispatch"
	"github.com/chewycrunch/webhook-forward-server/internal/forward"
	"github.com/chewycrunch/webhook-forward-server/internal/httpapi"
	"github.com/chewycrunch/webhook-forward-server/internal/logger"
	"github.com/chewycrunch/webhook-forward-server/internal/queue"
	"github.com/chewycrunch/webhook-forward-server/internal/store"
)

// build is overridden at link time with -ldflags "-X main.build=$(git rev-parse --short HEAD)".
var build = "develop"

func main() {
	if err := run(); err != nil {
		// The logger may not exist yet, so fail loudly on the default one.
		slog.Error("startup failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(build)
	if err != nil {
		// --help and --version already printed their output.
		if errors.Is(err, config.ErrHelpWanted) || errors.Is(err, config.ErrVersionWanted) {
			return nil
		}
		return err
	}

	log, err := logger.New(os.Stdout, cfg.Log.Format, cfg.Log.Level)
	if err != nil {
		return err
	}

	// Anything reaching for the package-level slog (a library, a stray call)
	// lands in the same stream and format as everything else.
	slog.SetDefault(log)

	log.Info("starting", "build", build, "addr", cfg.Web.Addr)
	log.Info("config resolved", "config", config.String(cfg))

	// --- infrastructure ---

	db, err := store.Open(cfg.DB.Path)
	if err != nil {
		return err
	}
	defer db.Close()

	q, err := queue.Open(cfg.DB.Path)
	if err != nil {
		return err
	}
	defer q.Close()

	// --- services ---

	fwd := forward.New(db, q, cfg.Dispatch.MaxQueueDepth, log)
	dsp := dispatch.New(q, discord.New(cfg.Dispatch.OutboundTimeout), cfg.Dispatch.GlobalRateLimit, log)

	// --- lifecycle ---

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dispatchErr := make(chan error, 1)
	go func() { dispatchErr <- dsp.Run(ctx) }()

	srv := http.Server{
		Addr:              cfg.Web.Addr,
		Handler:           httpapi.NewRouter(log, fwd),
		ReadHeaderTimeout: cfg.Web.ReadHeaderTimeout,

		// Bridge net/http's internal error logging into slog so connection
		// level failures are structured like everything else.
		ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Web.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err := <-dispatchErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		log.Info("shutdown signal received", "timeout", cfg.Web.ShutdownTimeout)
	}

	// Stop accepting new work first, then let the dispatcher finish what is
	// already in flight. Queued jobs survive in the database either way.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		return srv.Close()
	}

	<-dispatchErr
	log.Info("shutdown complete")
	return nil
}

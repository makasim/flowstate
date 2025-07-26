package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/badgerdriver"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/netdriver"
	"github.com/makasim/flowstate/netflow"
	"github.com/makasim/flowstate/pgdriver"
	"github.com/makasim/flowstate/ui"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	go handleSignals(cancel)

	cfg := config{
		Driver: "memdriver",
		BadgerDriver: badgerDriverConfig{
			Path: "badgerdb",
		},
	}
	if os.Getenv("FLOWSTATE_DRIVER") != "" {
		cfg.Driver = os.Getenv("FLOWSTATE_DRIVER")
	}
	if os.Getenv("FLOWSTATE_BADGERDRIVER_PATH") != "" {
		cfg.BadgerDriver.Path = os.Getenv("FLOWSTATE_BADGERDRIVER_PATH")
	}
	if os.Getenv("FLOWSTATE_BADGERDRIVER_IN_MEMORY") != "" {
		cfg.BadgerDriver.InMemory = os.Getenv("FLOWSTATE_BADGERDRIVER_IN_MEMORY") == `true`
	}
	if os.Getenv("FLOWSTATE_PGDRIVER_CONN_STRING") != "" {
		cfg.PostgresDriver.ConnString = os.Getenv("FLOWSTATE_PGDRIVER_CONN_STRING")
	}

	if err := newApp(cfg).Run(ctx); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}
}

func handleSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	<-signals
	log.Printf("INFO: got signal; canceling context")
	cancel()

	<-signals
	log.Printf("WARN: got second signal; force exiting")
	os.Exit(1)
}

type badgerDriverConfig struct {
	InMemory bool
	Path     string
}

type postgresDriverConfig struct {
	ConnString string
}

type config struct {
	Driver         string
	BadgerDriver   badgerDriverConfig
	PostgresDriver postgresDriverConfig
}

type app struct {
	cfg config
	l   *slog.Logger
}

func newApp(cfg config) *app {
	return &app{
		cfg: cfg,
		l:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (a *app) Run(ctx context.Context) error {
	var d flowstate.Driver
	switch a.cfg.Driver {
	case "memdriver":
		a.l.Info("init memdriver")
		d = memdriver.New(a.l)
	case "badgerdriver":
		a.l.Info("init badgerdriver")

		badgerCfg := badger.DefaultOptions(a.cfg.BadgerDriver.Path).
			WithInMemory(a.cfg.BadgerDriver.InMemory).
			WithLoggingLevel(2)
		db, err := badger.Open(badgerCfg)
		if err != nil {
			return fmt.Errorf("badger: open: %w", err)
		}
		defer db.Close()

		d0, err := badgerdriver.New(db)
		if err != nil {
			return fmt.Errorf("badgerdriver: new: %w", err)
		}
		defer d0.Shutdown(context.Background())

		d = d0
	case "pgdriver":
		a.l.Info("init pgdriver")
		conn, err := pgxpool.New(context.Background(), a.cfg.PostgresDriver.ConnString)
		if err != nil {
			return fmt.Errorf("pgxpool: new: %w", err)
		}
		defer conn.Close()

		d = pgdriver.New(conn, a.l)
	default:
		return fmt.Errorf("unknown driver: %s; support: memdriver, badgerdriver", a.cfg.Driver)
	}

	httpHost := `http://localhost:8080`
	if os.Getenv(`FLOWSTATE_HTTP_HOST`) != `` {
		httpHost = os.Getenv(`FLOWSTATE_HTTP_HOST`)
	}
	fr := netflow.NewRegistry(httpHost, d, a.l)
	defer fr.Close()

	e, err := flowstate.NewEngine(d, fr, a.l)
	if err != nil {
		return fmt.Errorf("new engine: %w", err)
	}

	r, err := flowstate.NewRecoverer(e, a.l)
	if err != nil {
		return fmt.Errorf("recoverer: new: %w", err)
	}

	dlr, err := flowstate.NewDelayer(e, a.l)
	if err != nil {
		return fmt.Errorf("delayer: new: %w", err)
	}

	addr := `0:8080`
	if os.Getenv(`FLOWSTATE_ADDR`) != `` {
		addr = os.Getenv(`FLOWSTATE_ADDR`)
	}

	uiH := http.FileServerFS(ui.PublicFS())

	a.l.Info("http server starting", "addr", addr)
	srv := &http.Server{
		Addr: addr,
		Handler: h2c.NewHandler(handleCORS(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if netdriver.HandleAll(rw, r, d, a.l) {
				return
			}
			if netflow.HandleExecute(rw, r, e) {
				return
			}

			uiH.ServeHTTP(rw, r)
		})), &http2.Server{}),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("WARN: http server: listen and serve: %s", err)
		}
	}()

	<-ctx.Done()

	var shutdownRes error
	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), time.Second*30)
	defer shutdownCtxCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		shutdownRes = errors.Join(shutdownRes, fmt.Errorf("http server: shutdown: %w", err))
	}

	if err := r.Shutdown(shutdownCtx); err != nil {
		shutdownRes = errors.Join(shutdownRes, fmt.Errorf("recovery: shutdown: %w", err))
	}

	if err := dlr.Shutdown(shutdownCtx); err != nil {
		shutdownRes = errors.Join(shutdownRes, fmt.Errorf("delayer: shutdown: %w", err))
	}

	if err := e.Shutdown(shutdownCtx); err != nil {
		shutdownRes = errors.Join(shutdownRes, fmt.Errorf("engine: shutdown: %w", err))
	}

	return shutdownRes
}

func handleCORS(h http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{`*`},
		AllowedMethods:   []string{`POST`, `GET`},
		AllowedHeaders:   []string{`*`},
		AllowCredentials: true,
		MaxAge:           600,
	}).Handler(h)
}

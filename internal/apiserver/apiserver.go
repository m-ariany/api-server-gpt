package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

type Server interface {
	// Start the server on the given port and keep it running as long as the stopCh is not signaled.
	//
	// Once the stopCh is signaled, it will try to gracefully shutdown the server without
	// interrupting any active connections within a ShutdownTimeout.
	//
	// If the ShutdownTimeout exceeds, it tries to immediately close all active net.Listeners and connections.
	Start(stopCh <-chan os.Signal)

	// Handle registers the handler for the given pattern.
	// If a handler already exists for pattern, throws an error.
	Handle(pattern string, handler http.Handler) error

	// HandleFunc registers the handler function for the given pattern.
	// If a HandleFunc already exists for pattern, throws an error.
	HandleFunc(pattern string, handler http.HandlerFunc) error

	// Get Server options
	ServerOptions() *ServerOptions
}

type ServerOptions struct {
	HttpAPITimeout  time.Duration
	ShutdownTimeout time.Duration
	Port            int
	HealthFunc      http.HandlerFunc
	ReadyFunc       http.HandlerFunc
	MetricsFunc     http.HandlerFunc
}

// httpserver describes HTTP API httpserver
type httpserver struct {
	http     http.Server
	mux      *http.ServeMux
	opts     ServerOptions
	started  int32
	handlers map[string]http.Handler
}

// Creates a new pointer of the Server.
// The server options that are not parametrized will be substituted with their default values.
func NewHttpServer(opts ServerOptions) Server {

	parseOptions(&opts)

	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	return &httpserver{
		mux: mux,
		http: http.Server{
			Addr: fmt.Sprintf(":%d", opts.Port),
		},
		opts:     opts,
		handlers: make(map[string]http.Handler),
	}
}

// Subsitute the opetions that are not parameterized with their default values.
func parseOptions(opts *ServerOptions) {
	if opts.Port == 0 {
		opts.Port = PORT
	}

	if opts.HttpAPITimeout == 0 {
		opts.HttpAPITimeout = HTTP_API_TIMEOUT
	}

	if opts.ShutdownTimeout == 0 {
		opts.ShutdownTimeout = SHUTDOWN_TIMEOUT
	}

	if opts.HealthFunc == nil {
		opts.HealthFunc = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(200)
		}
	}

	if opts.ReadyFunc == nil {
		opts.ReadyFunc = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(200)
		}
	}
}

func (s *httpserver) ServerOptions() *ServerOptions {
	return &s.opts
}

func (s *httpserver) Start(stopCh <-chan os.Signal) {

	if s.isStarted() {
		return
	}

	// So no other process can pass the above check
	atomic.AddInt32(&s.started, 1)

	s.mux.Handle("/health", s.opts.HealthFunc)
	s.mux.Handle("/ready", s.opts.ReadyFunc)
	s.mux.Handle("/metrics", s.opts.MetricsFunc)

	// TODO
	//s.http.MaxHeaderBytes
	//s.http.ReadTimeout
	//s.http.WriteTimeout
	//FIXME: this breaks stream response
	//s.http.Handler = http.TimeoutHandler(s.mux, s.opts.HttpAPITimeout, "")

	s.http.Handler = s.mux

	go func() {
		if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Could not start the http server")
			atomic.AddInt32(&s.started, -1)
			panic(err)
		}
	}()

	log.Info().Msgf("Listening on %s", s.http.Addr)

	<-stopCh

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), s.opts.ShutdownTimeout)
	defer cancel()

	// gracefully shutdown server
	if err = s.http.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("could not shutdown http server.")
		if err == context.DeadlineExceeded {
			log.Warn().Msg("Shutdown timeout exceeded. closing http server")
			if closeErr := s.http.Close(); closeErr != nil {
				log.Error().Err(err).Msg("could not close http connection.")
			}
		}
		return
	}

	log.Info().Msg("Http server shut down")
	return
}

func (s *httpserver) isStarted() bool {
	return atomic.LoadInt32(&s.started) == 1
}

func (s *httpserver) isHandled(pattern string) bool {
	_, ok := s.handlers[pattern]
	return ok
}

func (s *httpserver) Handle(pattern string, handler http.Handler) error {
	if s.isHandled(pattern) {
		return fmt.Errorf("%s is already registered", pattern)
	}
	s.handlers[pattern] = handler
	s.mux.Handle(pattern, handler)
	return nil
}

func (s *httpserver) HandleFunc(pattern string, handler http.HandlerFunc) error {
	if s.isHandled(pattern) {
		return fmt.Errorf("%s is already registered", pattern)
	}
	s.handlers[pattern] = handler
	s.mux.HandleFunc(pattern, handler)
	return nil
}

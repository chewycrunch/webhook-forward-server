// Package config loads runtime settings from defaults, environment
// variables, and command line flags, in that order of precedence. It is a
// leaf package: it imports nothing else from this module, so anything may
// depend on it without risking an import cycle.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ardanlabs/conf/v3"
)

// prefix namespaces every environment variable. Field paths are appended
// camel-split and uppercased, so Dispatch.MaxQueueDepth is read from
// WFS_DISPATCH_MAX_QUEUE_DEPTH, or from --dispatch-max-queue-depth.
const prefix = "WFS"

// These signal that the user asked for usage or version text, which Load has
// already printed. main should exit zero without starting the server.
var (
	ErrHelpWanted    = conf.ErrHelpWanted
	ErrVersionWanted = conf.ErrVersionWanted
)

// Config holds every runtime knob. Each field carries a working default so
// the server starts with nothing configured at all.
type Config struct {
	conf.Version

	Web struct {
		// Addr is the listen address for the HTTP server.
		Addr string `conf:"default::8080"`

		// ReadHeaderTimeout bounds how long a client may take to send its
		// headers, which is the cheap defense against a slowloris.
		ReadHeaderTimeout time.Duration `conf:"default:5s"`

		// ShutdownTimeout is how long in-flight requests get to finish
		// before the process exits.
		ShutdownTimeout time.Duration `conf:"default:20s"`
	}

	Log struct {
		// Level is the minimum severity to emit. Typed as slog.Level so
		// conf validates it at startup rather than at first log call.
		Level slog.Level `conf:"default:info,help:one of debug info warn or error"`

		// Format selects the handler: text for a terminal during local
		// development, json for a log aggregator in production.
		Format string `conf:"default:text,help:one of text or json"`
	}

	DB struct {
		// Path is where the durable job queue lives on disk. A restart
		// must not drop a deep backlog.
		Path string `conf:"default:webhooks.db"`
	}

	Dispatch struct {
		// GlobalRateLimit caps outbound requests per second across every
		// destination. Discord limits per IP as well as per webhook, so
		// per-bucket pacing alone is not enough.
		GlobalRateLimit float64 `conf:"default:45,help:outbound requests per second across all destinations"`

		// OutboundTimeout bounds a single delivery attempt to Discord.
		OutboundTimeout time.Duration `conf:"default:10s"`

		// MaxQueueDepth is how many jobs may be queued for one endpoint
		// before new sends are rejected, so one noisy caller cannot
		// exhaust memory or starve everyone else.
		MaxQueueDepth int `conf:"default:10000,help:queued jobs per endpoint before shedding load"`
	}
}

// Load parses configuration. On --help or --version it prints the requested
// text and returns the matching sentinel; on a malformed value it returns an
// error rather than silently falling back to the default.
func Load(build string) (Config, error) {
	cfg := Config{
		Version: conf.Version{
			Build: build,
			Desc:  "queues discord webhooks to prevent rate limiting",
		},
	}

	// On --help and --version, Parse returns the text to display as its
	// first value alongside a sentinel error.
	out, err := conf.Parse(prefix, &cfg)
	if err != nil {
		switch {
		case errors.Is(err, conf.ErrHelpWanted), errors.Is(err, conf.ErrVersionWanted):
			fmt.Println(out)
			return Config{}, err
		default:
			// conf already prefixes its errors with "parsing config".
			return Config{}, err
		}
	}

	return cfg, nil
}

// String renders the resolved config for logging at startup. Fields tagged
// mask or noprint are redacted, so this stays safe to log once secrets land
// in here.
func String(cfg Config) string {
	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Sprintf("rendering config: %v", err)
	}
	return out
}

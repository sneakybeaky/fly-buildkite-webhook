// This program demonstrates how to use tsnet as a library.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/buildkite/go-buildkite/v4"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port  string `env:"PORT, default=8080"`
	Token []byte `env:"BUILDKITE_TOKEN, required"`
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			method = r.Method
			url    = r.URL.String()
			proto  = r.Proto
		)

		userAttrs := slog.Group("user", "ip", ip)
		requestAttrs := slog.Group("request", "method", method, "url", url, "proto", proto)

		slog.Info("request received", userAttrs, requestAttrs)
		next.ServeHTTP(w, r)
	})
}

func timeRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			start = time.Now()
		)

		next.ServeHTTP(w, r)
		slog.Info("request time", slog.Duration("duration", time.Since(start)))
	})
}

func messageHandler(message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(message))
	})
}

func headersHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		for name, headers := range r.Header {
			for _, h := range headers {
				fmt.Fprintf(w, "%v: %v\n", name, h)
			}
		}
	})
}

func webhookHandler(secretKey []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		payload, err := buildkite.ValidatePayload(r, secretKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			slog.Error("Invalid payload", slog.String("error", err.Error()))
			return
		}

		event, err := buildkite.ParseWebHook(buildkite.WebHookType(r), payload)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			slog.Error("Unable to parse webhook", slog.String("error", err.Error()))
			return
		}

		slog.Info("Got event from webhook", "event", event)
	})
}

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx := context.Background()

	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /hello", logRequest(timeRequest(messageHandler("Hello world!"))))
	mux.Handle("GET /headers", logRequest(timeRequest(headersHandler())))

	// probe endpoints - don't log these
	mux.Handle("POST /", webhookHandler(cfg.Token))
	mux.Handle("GET /health", messageHandler("OK"))

	err := http.ListenAndServe(":"+cfg.Port, mux)

	if err != nil {
		slog.Error("Unable to serve", "error", err)
		os.Exit(1)
	}

}

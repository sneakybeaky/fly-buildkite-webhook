// This program demonstrates how to use tsnet as a library.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port string `env:"PORT, default=8080"`
	//Dataset   string `env:"BQ_DATASET_NAME, required"`
	//Table     string `env:"BQ_TABLE_NAME, required"`
	//ProjectId string `env:"PROJECT_ID, required"`
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

func webhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		for name, headers := range r.Header {
			for _, h := range headers {
				slog.Info("Header", "name", name, "value", h)
			}
		}

		data, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(w, err.Error(), 500)
			slog.Error("webhookHandler: failed to read webhook body", slog.String("error", err.Error()))
			return
		}

		slog.Info("Body", "payload", string(data))
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
	mux.Handle("POST /", webhookHandler())
	mux.Handle("GET /health", messageHandler("OK"))

	err := http.ListenAndServe(":"+cfg.Port, mux)

	if err != nil {
		slog.Error("Unable to serve", "error", err)
		os.Exit(1)
	}

}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yourname/click-counter/internal/httpapi"
	"github.com/yourname/click-counter/internal/storage"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := getenv("HTTP_ADDR", ":3000")
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getenv("MONGO_DB", "clicksdb")
	collName := getenv("MONGO_COLL", "clicks")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := storage.NewMongoRepo(ctx, mongoURI, dbName, collName)
	if err != nil {
		log.Fatalf("mongo init: %v", err)
	}
	defer repo.Close(context.Background())

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	h := httpapi.NewHandlers(repo)
	r.Get("/counter/{bannerID}", h.Counter)
	r.Post("/stats/{bannerID}", h.Stats)
	r.Post("/stats", h.StatsAll)

	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
}

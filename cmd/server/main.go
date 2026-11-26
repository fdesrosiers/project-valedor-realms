package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"realm/internal/game"
	"realm/internal/transport"
	"realm/internal/world"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	tickRate := flag.Int("tick", 20, "simulation ticks per second")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := world.New(world.DefaultZone(), world.DefaultSpawn)
	g := game.New(w, *tickRate)
	go g.Run(ctx)

	mux := http.NewServeMux()
	mux.Handle("/ws", transport.Handler(g))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	srv := &http.Server{Addr: *addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("realm server listening on %s (tick %d Hz)", *addr, *tickRate)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	log.Print("server stopped")
}

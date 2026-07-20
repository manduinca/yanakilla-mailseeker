package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flag"

	"github.com/manduinca/yanakilla-mailseeker/internal/api"
	"github.com/manduinca/yanakilla-mailseeker/internal/zinc"
	"github.com/manduinca/yanakilla-mailseeker/web"
)

func main() {
	var (
		port     = flag.Int("port", 3000, "puerto HTTP")
		zincURL  = flag.String("zinc", envOr("ZINC_URL", "http://localhost:4080"), "URL de ZincSearch")
		zincUser = flag.String("user", envOr("ZINC_USER", "admin"), "usuario de ZincSearch")
		zincPass = flag.String("pass", envOr("ZINC_PASSWORD", "Complexpass#123"), "password de ZincSearch")
		index    = flag.String("index", "emails", "nombre del índice")
	)
	flag.Parse()

	client := zinc.New(*zincURL, *zincUser, *zincPass)
	server := api.NewServer(client, *index, web.Handler())

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("no se pudo abrir el puerto %d: %v", *port, err)
	}

	fmt.Printf("Yanakilla is running in http://localhost:%d\n", *port)

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("cierre forzado: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

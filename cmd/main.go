package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	h "github.com/htetmyatthar/server-manager/api/handler"
	. "github.com/htetmyatthar/server-manager/internal/config"
	"github.com/htetmyatthar/server-manager/internal/utils"
)

var (
	mux    *http.ServeMux
	server *http.Server
)

func init() {
	mux, server = InitServer()

	// register the handlers via mux
	server.Handler = mux
}

func main() {

	fs := http.FileServer(http.Dir("web/static"))
	staticHandler := http.StripPrefix("/static/", fs)
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if filepath.Ext(r.URL.Path) == ".css" {
			w.Header().Set("Content-Type", "text/css")
		} else if filepath.Ext(r.URL.Path) == ".js" {
			w.Header().Set("Content-Type", "text/javascript")
		}
		staticHandler.ServeHTTP(w, r)
	}))

	// Normal end points
	mux.HandleFunc("/", h.ApologyHandler)
	mux.HandleFunc("GET /admin", h.AdminHandlerGET)
	mux.HandleFunc("GET /admin/dashboard", h.AdminDashboardGet)

	// point for admin authentication.
	mux.HandleFunc("POST /admin", h.AdminHandlerPOST)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("Server started on http://%v%v\nMemory Usage: %d bytes", WebHost, WebPort, utils.GetMemoryUsage())
		err := server.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatalln("Internal server error", err)
			} else {
				log.Println("Http server shutting down...")
			}
		}
	}()

	signal := <-sigChan
	log.Println("Received shutdown request signal:", signal)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalln("HTTP shutdown err: ", err)
	}
}

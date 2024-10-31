// v2ray server-manager that can manage the user and provide the QR codes that can be both
// opened or the device-locked one which are compatible with v2box application.
// You can create, and delete users, restart the v2ray server, and generate QR codes for each users.
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

	// "path/filepath"
	"syscall"
	"time"

	h "github.com/htetmyatthar/server-manager/api/handler"
	m "github.com/htetmyatthar/server-manager/api/middleware"
	. "github.com/htetmyatthar/server-manager/internal/config"
	d "github.com/htetmyatthar/server-manager/internal/database"
	"github.com/htetmyatthar/server-manager/internal/utils"
)

var (
	muxHTTPS    *http.ServeMux
	serverHTTPS *http.Server

	muxHTTP      *http.ServeMux
	serverHTTP   *http.Server
	sessionStore d.SessionStore

	// embedded static file handler
	staticHandler http.Handler

	// userLocker locks out the user from logging in if the user exceeds certain number of trials.
	userLocker = utils.NewLockedOutRateLimiter()

	// Defined rate limit: 1 request per every 5 seconds with a burst of 3 for each ip address.
	// call Limit() method to apply the defined rate limit on the end points.
	genericRateLimiter = m.NewRateLimiterMiddleware(1.0/5.0, 3, 5*time.Minute)
)

func init() {
	// gets the new mem session store.
	sessionStore = d.NewMemSessionStore()

	// HTTPS server config
	muxHTTPS, serverHTTPS = InitHTTPSServer()
	serverHTTPS.Handler = m.Logging(muxHTTPS)

	// HTTP server config
	muxHTTP, serverHTTP = InitHTTPServer()
	serverHTTP.Handler = m.Logging(muxHTTP)

	// static files handler via file paths.
	staticHandler = InitStaticServer()
}

func main() {
	// TODO: store the keys in the backend and produce the config URI in backend.
	// TODO: check the index out of bound cases and if exists in slices when deleting and creating a qr.
	// TODO: save the backup file and then check the config of the prepared file and if correct run and override,
	// if not don't run and roll back to the back up file.

	// static file server
	muxHTTPS.Handle("GET /static/"+Version+"/", http.StripPrefix("/static/"+Version+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set appropriate content type headers
		ext := filepath.Ext(r.URL.Path)
		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "text/javascript")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		}

		// Set cache headers
		w.Header().Set("Cache-Control", "public, max-age=31536000")

		staticHandler.ServeHTTP(w, r)
	})))

	// routes HTTPS
	muxHTTPS.HandleFunc("/", m.LoginRequired(h.DefaultHandler, sessionStore))
	muxHTTPS.HandleFunc("/hello", h.Hello)
	muxHTTPS.HandleFunc("GET /admin/login", h.AdminLoginGET(sessionStore))
	muxHTTPS.HandleFunc("GET /admin/dashboard", m.LoginRequired(h.AdminDashboardGET, sessionStore))
	muxHTTPS.HandleFunc("GET /server/ip", h.ServerIPHandlerGET)

	muxHTTPS.HandleFunc("POST /admin/login", m.CSRFRequired(h.AdminLoginPOST(sessionStore, userLocker)))
	muxHTTPS.HandleFunc("POST /admin/accounts/edit", m.CSRFRequired(m.LoginRequired(h.AccountEditPOST, sessionStore)))
	muxHTTPS.HandleFunc("POST /admin/accounts", m.CSRFRequired(m.LoginRequired(h.AccountCreatePOST, sessionStore)))
	muxHTTPS.HandleFunc("POST /admin/accounts/delete", m.CSRFRequired(m.LoginRequired(h.AccountDeletePOST, sessionStore)))
	muxHTTPS.HandleFunc("POST /server", genericRateLimiter.Limit(m.CSRFRequired(m.LoginRequired(h.ServerRestartPOST, sessionStore))))

	// routes HTTP
	muxHTTP.HandleFunc("/", h.RedirectToHTTPSHandler)

	// server settings
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("HTTPS Server started on https://%s%s\nMemory Usage: %d bytes\n", *WebHost, *WebPort, utils.GetMemoryUsage())
		err := serverHTTPS.ListenAndServeTLS(*WebCert, *WebKey)
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatalln("Starting up HTTPS server error, might be config related: ", err)
			} else {
				log.Println("HTTPS server shutting down...")
			}
		}
	}()

	go func() {
		fmt.Printf("HTTP Server started on http://%s%s\nMemory Usage: %d bytes\n", *WebHost, ":80", utils.GetMemoryUsage())
		err := serverHTTP.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatalln("Starting up HTTP server error, might be config related: ", err)
			} else {
				log.Println("HTTP server shutting down...")
			}
		}
	}()

	signal := <-sigChan
	log.Println("Received shutdown request signal:", signal)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := serverHTTPS.Shutdown(ctx); err != nil {
		log.Fatalln("HTTPS server shutdown err: ", err)
	}
	if err := serverHTTP.Shutdown(ctx); err != nil {
		log.Fatalln("HTTP server shudown err: ", err)
	}
}

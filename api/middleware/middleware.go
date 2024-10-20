package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	. "github.com/htetmyatthar/server-manager/internal/config"
	data "github.com/htetmyatthar/server-manager/internal/database"
	"github.com/htetmyatthar/server-manager/internal/utils"
)

// wrappedWriter is extended form of http.ResponseWriter for getting the servedStatusCode to log.
type wrappedWriter struct {
	http.ResponseWriter     // responsewriter
	servedStatusCode    int // the statusCode that is served by the response http.ResponseWriter
}

// WriteHeader is to override the http.ResponseWriter.WriteHeader method
// wrappedWriter.WriteHeader also write the statusCode to the servedStatusCode
func (w *wrappedWriter) WriteHeader(statusCode int) {
	// call the original one
	w.ResponseWriter.WriteHeader(statusCode)
	w.servedStatusCode = statusCode
}

// Logging for every handler in server. Following things are being logged to stdout.
//
// 1. servedStatusCode for the request
// 2. request method
// 3. request url
// 4. time it took to response
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter:   w,
			servedStatusCode: http.StatusOK,
		}

		// NOTE: METHOD INHERITANCE THROUGH EMBEDDING
		// if next didn't(explicitly) called the wrapped.WriteHeader method, it will be the same as
		// default behaviour which is calling the http.ResponseWriter.WriteHeader
		next.ServeHTTP(wrapped, r)
		log.Println(wrapped.servedStatusCode, r.Method, r.URL.Path, time.Since(start))
	})
}

// LoginRequired checks the user has already logged in or not by
// checking the session cookie. Otherwise, the user is redirect to
// Login page and forced to login.
func LoginRequired(next http.HandlerFunc, sessionStore data.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, deleteCookiesFunc := utils.SessionValidate(r, sessionStore)
		if deleteCookiesFunc != nil {
			log.Println("Invalid session deleting cookies")
			// utils.ErrInvalidSession will be returned
			deleteCookiesFunc(w)
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		if value == SessionPublic { // user has pre-session id
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		log.Println("Login middleware success id:", value)
		next.ServeHTTP(w, r)
	}
}

// CSRFRequired checks that the given request has valid CSRF token or not.
// Rejecting to serve the next if the given CSRF is invalid. this checks the form values
// and then header cookies for csrf token. This can also be used in JSON APIs.
// For JSON APIs, the CSRF token name should be in the headers and stored as the name
// that the requester got from.
func CSRFRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// finding out the request is in JSON format or Normal browser
		requestContentType := r.Header.Get("Content-Type")
		isRequestJSON := strings.Contains(requestContentType, "application/json")

		// extract CSRF token from the form value or headers.
		token := r.FormValue(CSRFFormFieldName)
		if token == "" {
			token = r.Header.Get(CSRFHeaderFieldName)
		}

		// handle missing token
		if token == "" {
			responseMessage := "Missing valid CSRF token"
			if isRequestJSON {
				utils.JSONRespondError(w, http.StatusBadRequest, responseMessage)
			} else {
				utils.RenderError(w, "Bad request.", http.StatusBadRequest)
			}
			log.Println("Error: ", responseMessage)
			return
		}

		// validate CSRF token
		_, err := utils.VerifyCSRF(token, r)
		if err != nil {
			responseMessage := "Bad request, invalid CSRF token"
			if isRequestJSON {
				utils.JSONRespondError(w, http.StatusBadRequest, "Bad request, invalid")
			} else {
				utils.RenderError(w, "Bad request", http.StatusForbidden)
			}
			log.Println("Error: ", err, responseMessage)
			return
		}

		next.ServeHTTP(w, r)
		return
	}
}

// HttpClient represents a client with a rate limiter and the last seen time.
type HttpClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter holds the configuration for rate limiting.
type RateLimiter struct {
	// maps the clients's unique identifiers to the HttpClients
	clients         map[string]*HttpClient
	mu              sync.Mutex
	r               rate.Limit
	b               int
	cleanupInterval time.Duration
}

// NewRateLimiterMiddleware initializes a new RateLimiterMiddleware. Also start a go routine to clean up the old clients.
func NewRateLimiterMiddleware(r rate.Limit, b int, cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:         make(map[string]*HttpClient),
		r:               r,
		b:               b,
		cleanupInterval: cleanupInterval,
	}

	// Start a background goroutine to clean up old clients.
	go rl.cleanupClients()

	return rl
}

// getClient retrieves the client's rate limiter, creating one if necessary.
func (rl *RateLimiter) getClient(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if client, exists := rl.clients[ip]; exists {
		client.lastSeen = time.Now()
		return client.limiter
	}

	limiter := rate.NewLimiter(rl.r, rl.b)
	rl.clients[ip] = &HttpClient{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// cleanupClients periodically removes clients that haven't been seen for a while.
func (rl *RateLimiter) cleanupClients() {
	for {
		time.Sleep(rl.cleanupInterval)
		rl.mu.Lock()
		for ip, client := range rl.clients {
			if time.Since(client.lastSeen) > rl.cleanupInterval {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Limit limits the api calls with the rl's defined rules.
func (rl *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		limiter := rl.getClient(ip)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
}

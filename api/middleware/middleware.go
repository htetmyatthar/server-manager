package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	. "github.com/htetmyatthar/server-manager/internal/config"
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
func LoginRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, deleteCookiesFunc := utils.SessionValidate(r)
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
		log.Println("why is it missing the token.")

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

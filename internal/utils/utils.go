package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	. "github.com/htetmyatthar/server-manager/internal/config"
	. "github.com/htetmyatthar/server-manager/internal/database"
)

// V2rayClient is to add or remove the users from the v2ray config.
type V2rayClient struct {
	Id      string `json:"id"`
	AlterId int    `json:"alterId"`
}

// Client is to store the user basic info in the seperate json data file.
type Client struct {
	Id         string `json:"id"`
	AlterId    int    `json:"alterId"`
	Username   string `json:"username"`
	DeviceId   string `json:"deviceId"`
	StartDate  string `json:"startDate"`
	ExpireDate string `json:"expireDate"`
}

type InboundSettings struct {
	Clients []V2rayClient `json:"clients"`
}

type Inbound struct {
	Port           int             `json:"port"`
	Listen         string          `json:"listen"`
	Protocol       string          `json:"protocol"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings"` // Handles streamSettings dynamically
}

var (
	// parse the templates from config
	templates = InitTemplates()
)

// CAUTION: Before calling this function always ensure to provided the status code with w.WriteHeader().
// RenderTemplate renders the preparsed templates with the given templateName and return the parsed
// template as http response. templateName should be the name of the preparsed template files that exists
// the '{project root}/web/templates/' path. You can find the available templateNames in the Templates.Temps map.
func RenderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	tmpl, ok := templates[templateName]
	if !ok {
		http.Error(w, "Template not found", http.StatusNotFound)
		log.Println("Template error, not found.")
		return
	}
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Println("Templating error,", err)
		http.Error(w, "Internal server, error", http.StatusInternalServerError)
		return
	}
}

// RenderTemplate renders the given templates file names in the `{project root}/web/templates/` path.
// Return the parsed template as http response.
func RenderParseTemplate(w http.ResponseWriter, data any, filenames ...string) {
	// prefix with working directory
	for i, filename := range filenames {
		filenames[i] = filepath.Join(TemplateBasePath, filename)
	}

	tmpl := template.Must(template.ParseFiles(filenames...))
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Println("Templating error,", err)
		http.Error(w, "Internal server error.", http.StatusInternalServerError)
		return
	}
}

// apologyTemplate is parsed apology.html one to used with RenderError.+
var apologyTemplate = template.Must(template.ParseFiles("web/templates/includes/apology.html"))

// RenderError reders the apology template with the given status and return the parsed
// template as http response. `apology.html` file should be in the path
// `{project root}/web/templates/` to be able to work with this function.
func RenderError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	log.Println(apologyTemplate)
	data := struct {
		StatusCode int
		Message    string
	}{
		StatusCode: status,
		Message:    message,
	}
	err := apologyTemplate.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getMemoryUsage returns the memory usage in the current
// state of the function being called.
func GetMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.TotalAlloc
}

// JSONRespond responds the request with the given http status code and data.
// The given data will be marshal into JSON format.
func JSONRespond(w http.ResponseWriter, code int, data any) {
	response, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(response); err != nil {
		http.Error(w, "Unable to create a json response", http.StatusInternalServerError)
		return
	}
}

// JSONRespondError responds with errors in JSON format using the given
// http.ResponseWriter, http status code and error message.
// This function doesn't allow to use the http.StatusOK for the code, and it'll
// panic if one try to use.
func JSONRespondError(w http.ResponseWriter, code int, msg string) {
	if code == http.StatusOK {
		panic("You can't use http status ok(200) for error responses.")
	}
	JSONRespond(w, code, map[string]string{"error": msg})
}

// Function to restart V2Ray service
func RestartService() error {
	cmd := exec.Command("sudo", "systemctl", "restart", "v2ray2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to restart service: %s, %v", string(output), err)
	}
	return nil
}

// Function to validate V2Ray configuration
func ValidateConfig() error {
	cmd := exec.Command("v2ray", "-test", "-config", *ConfigFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Config test failed: %s, %v", string(output), err)
	}
	return nil
}

// GererateSessionId generate CSPRN(cryptographically secure pseudo-random numbers) base on
// the given number of bytes and encode it into base64 to return a random, unique and url safe string.
// Returns error only if the internal CSPRNG is broken.
// example if n = 32 --> 42 byte, random string will return.
func GenerateSessionId(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SessionValidate validates the session from the given request if there's any.
// Session id of the given request is valid(present in the cache) if return error is nil.
// CAUTION: type of the session is not validated as there are different type of sessions
// Returns the utils.DeleteAllCookies function if the session that the user has is invalid.
//
// for e.g.(pre-session<public>, private<authenticated>)
//
// Return the result(value) string(probably user id) that is stored inside dbRedis cache.
// If result string is "NaN", it is utils.SessionPublic
func SessionValidate(r *http.Request) (string, func(http.ResponseWriter) error) {
	session, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", nil
	}
	value, ok := SessionStore[session.Value]
	if !ok {
		return "", func(w http.ResponseWriter) error {
			DeleteAllCookies(w, r)
			return ErrInvalidSession
		}
	}
	return value, nil
}

// DeleteAllCookies deletes the cookies in the following paths to be deleted.
// ["/", "/admin/login"]
func DeleteAllCookies(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	paths := []string{"/", "/admin/login"}

	for _, cookie := range cookies {
		for _, path := range paths {
			http.SetCookie(w, &http.Cookie{
				Name:   cookie.Name,
				Value:  "",
				Path:   path,
				MaxAge: -1,
			})
		}
	}
}

// TODO: fix the documentation:
// SessionSetPath sets a new session to the given response to be able to access
// the given path while also adding to the dbRedis cache. Returns the session cookie
// that is being set and error if there's problem with creating random strings or redis cache problem.
// CAUTION: for pattern matching for path parameter. if the path is suffixed with "/"(back-slash),
// the cookie will be sent to all the sub-paths under the prefix.
// e.g. "/login/" will match both "/login/60", andf"/login/page/" but not "/loginpage/"
// e.g. "/login" will exactly match with http path "/login"
func SessionSetPath(w http.ResponseWriter, path string) (*http.Cookie, string, error) {
	// pre-session tokens for login form
	sessionString, err := GenerateSessionId(32)
	if err != nil {
		return nil, "", err
	}
	expireTime := 600 // NOTE: 10mins

	SessionStore[sessionString] = SessionPublic

	// create new session cookie
	session := &http.Cookie{
		Name:   SessionCookieName,
		Value:  sessionString,
		Path:   path,
		Domain: *WebHost,
		MaxAge: expireTime,
		// WARN: use true only with https
		Secure: true,
		// for the http request only.
		HttpOnly: true,
		// don't allow even the subdomain this can help stop csrf attacks.
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, session)
	return session, sessionString, nil
}

// generateHMACHash returns the hash value of HMAC with the given secret and message
func generateHMACHash(message []byte, secret []byte) ([]byte, error) {
	hasher := hmac.New(sha256.New, secret)
	_, err := hasher.Write(message)
	if err != nil {
		return nil, nil
	}
	return hasher.Sum(nil), nil
}

// GenerateCSRF generate CSRF tokens for state changing that needed to be guarded,
// using the given sessionId. Return error if the random number generation is failing or
// creating MAC hash is failing. Given sessionId should be in the form of base64.URLEncoding
//
// full process: token => b64(HMAC(sessionId + "!" + b64(random_bytes), CsrfSecretBytes))+"."+(sessionId+"!"+b64(random_bytes))
// simplify: token=b64(HMAC)+"."+(sessionId + "!" + b64(random_bytes))
func GenerateCSRF(sessionId string) (token string, err error) {
	var buf bytes.Buffer

	// panic can happen in write operations to buf but is mostly impossible.
	defer func() {
		if r := recover(); r != nil {
			log.Println("DANGER: Panic occurred when creating CSRF token", r)
			err = errors.New("Error while writing to buffer.")
		}
	}()

	// create random values for avoiding collision
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	// concat to create message buf.Bytes() is the message.
	// considering sessionId be in the base64 encoded.
	buf.WriteString(sessionId)
	buf.WriteString("!")
	buf.WriteString(base64.URLEncoding.EncodeToString(b))
	messageString := buf.String()
	messageBytes := buf.Bytes()
	buf.Reset() // reset to clear out the values

	// generate HMAC hash value
	hash, err := generateHMACHash(messageBytes, CsrfSecretBytes)
	if err != nil {
		return
	}

	// concat to create a CSRF token
	buf.WriteString(base64.URLEncoding.EncodeToString(hash)) // base64 encode for large entropy
	buf.WriteString(".")
	buf.WriteString(messageString) // base64 encoded for large entropy
	token = buf.String()
	return
}

// VerifyCSRF verifies the CSRF token is valid or not using the request.
// The given csrf token is valid only if there's no error.
func VerifyCSRF(token string, r *http.Request) (bool, error) {
	tokenValues := strings.Split(token, ".")
	messageValues := strings.Split(tokenValues[1], "!")

	// be aware of the same site strict policy in cookies.
	rSession, err := r.Cookie(SessionCookieName)
	if err != nil {
		return false, err
	}
	log.Println("they didn't make pass this.")

	// comparing token's session and requested session
	if messageValues[0] != rSession.Value {
		log.Println("csrf: ", messageValues[0], "session: ", rSession.Value)
		return false, errors.New("Given token and requested session Id is not the same")
	}

	// decoding the given token hash
	tokenHash, err := base64.URLEncoding.DecodeString(tokenValues[0])
	if err != nil {
		return false, err
	}

	// regerating hash with the given info
	hash, err := generateHMACHash([]byte(tokenValues[1]), CsrfSecretBytes)
	if err != nil {
		return false, err
	}

	// only true if the HMAC hashes are the same
	if hmac.Equal(hash, tokenHash) {
		return true, nil
	}
	return false, errors.New("DANGER: Internal Server error.")
}

// SessionSetPrivate sets the session to the given w using u while also adding
// to the dbRedis cache. User u's id field should be initialized with real value.
// Return error if there's problem with creating random session strings or redis cache problem,
// and also when the user's id
func SessionSetPrivate(w http.ResponseWriter) error {
	// create session string
	sessionString, err := GenerateSessionId(32)
	if err != nil {
		return err
	}

	// NOTE: valid for 10mins.
	expireTime := 600

	SessionStore[sessionString] = strconv.Itoa(SessionCount)
	SessionCount++

	// create new session cookie
	session := &http.Cookie{
		Name:   SessionCookieName,
		Value:  sessionString,
		Path:   "/",
		Domain: *WebHost,
		MaxAge: expireTime,
		// WARN: use true only with https
		Secure: true,
		// for the http request only.
		HttpOnly: true,
		// don't allow even the subdomain this can help stop csrf attacks.
		SameSite: http.SameSiteStrictMode,
	}

	// only set at the end for ensuring the safety
	http.SetCookie(w, session)
	return nil
}

// VerifyPassword verify the password Given with the correct Password.
// This method can be used to check the input password is the correct u's Password or not,
// while returning an error if there's any.
//
// The password is a correct password, only if the boolean is "true", and error is "nil".
func VerifyPassword(password string, correct string) (bool, error) {
	_, hashBytes, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	userPassword, err := hex.DecodeString(correct)
	if err != nil {
		return false, err
	}
	if subtle.ConstantTimeCompare(hashBytes, userPassword) == 1 {
		return true, nil
	}
	log.Println("Verify password gone wrong.")
	return false, errors.New("Wrong password")
}

// HashPassword hashes the given password string to sha-256 hash returning the hashed values
// as a hex-dec value string and also in the form of byte slice.
func HashPassword(password string) (string, []byte, error) {
	hasher := sha256.New()
	_, err := hasher.Write([]byte(password))
	if err != nil {
		return "", nil, err
	}
	hashedBytes := hasher.Sum(nil)
	// hex is easier to maintain
	return hex.EncodeToString(hashedBytes), hashedBytes, err
}

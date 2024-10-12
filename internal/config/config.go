// This is the whole configurations needed for the server-manager to run.
// It includes the servers, and variables.
package config

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var (
	WebHost          *string
	WebHostIP        *string
	Admin            *string
	AdminPw          *string
	WebPort          *string
	WebCert          *string
	WebKey           *string
	V2rayPort        *string
	UserFile         *string
	ConfigFile       *string
	TemplateBasePath string = "web/templates/"

	CsrfSecret      string
	CsrfSecretBytes []byte
)

const (
	// SessionCookieName is the name of the cookie, the session id will be stored in.
	// Using this to make sure the cookies are not named explicitly to avoid from adavasaries
	// attempt of stealing cookies.
	SessionCookieName string = "lothoneId"

	// SessionPublic is the value of the values that is stored as the public session used for
	// logining in for and such. Value of NaN is consider public session for session id keys.
	SessionPublic string = "NaN"

	// Name of the form field the csrf token will be.
	CSRFFormFieldName string = "token"

	// Name of the header the csrf token will be stored in.
	// Using this to make sure the cookies are not named explicityly to avoid from advasaries
	// attempt of stealing cookies.
	CSRFHeaderFieldName string = "token"
)

func init() {
	WebHost = flag.String("hostname", "127.0.0.1", "fully qualify domain name of the server")
	WebHostIP = flag.String("hostip", "127.0.0.1", "ipv4 or ipv6 address of the server")
	WebPort = flag.String("webport", ":8888", "port number of the control panel web server")
	V2rayPort = flag.String("v2rayport", "443", "port number of the v2ray proxy server")
	WebCert = flag.String("webcert", "localhost.crt", "ssl/tls certificate for the web server")
	WebKey = flag.String("webkey", "localhost.key", "ssl/tls certificate key for the web server")
	Admin = flag.String("admin", "lothoneadmin", "admin username for the web server")
	AdminPw = flag.String("adminpw", "lothoneadmin0", "admin password for the web server")
	UserFile = flag.String("userfile", "test/user_data.json", "track the users of the server")
	ConfigFile = flag.String("configfile", "test/server.json", "config file of the v2ray proxy server")

	// parse the flags
	flag.Parse()

	CsrfSecret = "a1b2c3d4e5f60708192a3b4c5d6e7f80"
	CsrfSecretBytes = []byte(CsrfSecret)
}

// InitServer initizlie the HTTPS server returning a multiplexor and the server
func InitHTTPSServer() (*http.ServeMux, *http.Server) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:           *WebPort,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return mux, server
}

// InitServer initizlie the HTTP server returning a multiplexor and the server that will run on port 80
func InitHTTPServer() (*http.ServeMux, *http.Server) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:           ":80",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return mux, server
}

// initTemplates function is for initializing the configured templates
// and storing it inside a map and return it
func InitTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	// getting the base files
	layouts, err := filepath.Glob(TemplateBasePath + "layouts/*.html")
	if err != nil {
		log.Fatal(err)
	}

	// getting the optional files
	includes, err := filepath.Glob(TemplateBasePath + "includes/*.html")
	if err != nil {
		log.Fatal(err)
	}

	// Generate our templates map from our layouts/ and templates/directories
	// REFERENCE : https://blog.questionable.services/article/approximating-html-template-inheritance/
	for _, include := range includes {
		files := append(layouts, include)
		values := strings.Split(filepath.Base(include), ".")
		log.Println("Parsing templates: ", filepath.Base(include))
		templates[values[0]] = template.Must(template.ParseFiles(files...))
	}

	for _, temp := range templates {
		log.Println("these are define:", temp.DefinedTemplates())
		log.Println("names: ", temp.Name())
	}

	return templates
}

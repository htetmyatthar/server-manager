// This is the whole configurations needed for the server-manager to run.
// It includes the servers, and variables.
package config

import (
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/htetmyatthar/server-manager/web"
)

var (
	gotifyAPIKeys    *string
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
	SessionDuration  *int
	LockOutDuration  *int
	GotifyServer     *string
	GotifyAPIKeys    []string
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

	// Maximum allowed failed attempts for user authentications.
	MaxFailedAttempts = 5
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
	GotifyServer = flag.String("gotifyserver", "meet.htetmyatthar.me:8080", "push nofication server domain name")
	gotifyAPIKeys = flag.String("gotifyapikeys", "somekey,somekey", "keys for using with push notification system seperated by comma(,)")
	SessionDuration = flag.Int("sessionduration", 10, "loggedin session remembered duration in minutes")
	LockOutDuration = flag.Int("lockoutduration", 30, "locking out time for wrong password in minutes")

	// parse the flags
	flag.Parse()

	GotifyAPIKeys = strings.Split(*gotifyAPIKeys, ",")
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

// Initialize templates from embedded filesystem
func InitEmbedTemplates() (*template.Template, map[string]*template.Template){
	// Initialize template maps
	templates := make(map[string]*template.Template)

	// Initialize error template first
	errorTmpl, err := template.ParseFS(web.WebFS, "templates/includes/apology.html")
	if err != nil {
		log.Fatal(err)
	}

	// Read layout files
	layouts, err := fs.Glob(web.WebFS, "templates/layouts/*.html")
	if err != nil {
		log.Fatal(err)
	}

	// Read include files
	includes, err := fs.Glob(web.WebFS, "templates/includes/*.html")
	if err != nil {
		log.Fatal(err)
	}

	// Parse each template with its layout
	for _, include := range includes {
		// Skip apology template as it's handled separately
		if strings.Contains(include, "apology.html") {
			continue
		}

		var files []string
		files = append(files, layouts...)
		files = append(files, include)

		name := strings.TrimSuffix(filepath.Base(include), ".html")
		t := template.New(filepath.Base(include))

		// Parse all files from embedded filesystem
		for _, file := range files {
			data, err := web.WebFS.ReadFile(file)
			if err != nil {
				log.Fatal(err)
			}

			_, err = t.Parse(string(data))
			if err != nil {
				log.Fatal(err)
			}
		}

		templates[name] = t
	}

	return errorTmpl, templates
}

// Initialize static file server from embedded filesystem
func InitStaticServer() http.Handler {
	// Get the static subdirectory
	staticFS, err := fs.Sub(web.WebFS, "static")
	if err != nil {
		log.Fatal("failed to create sub file system: ", err)
	}

	// Create file server from embedded files
	fsHandler := http.FileServer(http.FS(staticFS))
	return fsHandler
}

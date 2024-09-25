package config

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var (
	WebHost = "127.0.0.1"
	WebPort = ":8888"
	TemplateBasePath string = "web/templates/"
)

func InitServer() (*http.ServeMux, *http.Server) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:           WebPort,
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

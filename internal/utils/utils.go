package utils

import (
	"html/template"
	"log"
	"path/filepath"
	"net/http"
	"runtime"
	. "github.com/htetmyatthar/server-manager/internal/config"
)

var (
	templates = InitTemplates() // parse the templates from config
)

// RenderTemplate renders the preparsed templates with the given templateName and return the parsed
// template as http response. templateName should be the name of the preparsed template files that exists
// the '{project root}/web/templates/' path. You can find the available templateNames in the Templates.Temps map.
// CAUTION: Before calling this function always ensure to provided the status code with w.WriteHeader().
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

// apologyTemplate is parsed apology.html one to used in RenderError.+
var apologyTemplate = template.Must(template.ParseFiles("web/templates/includes/apology.html"))

// RenderError reders the apology template with the given status and return the parsed
// template as http response. `apology.html` file should be in the path
// `{project root}/web/templates/` to be able to work with this function.
func RenderError(w http.ResponseWriter, data any, status int) {
	w.WriteHeader(status)
	log.Println(apologyTemplate)
	err := apologyTemplate.Execute(w, struct{ StatusCode int }{StatusCode: status})
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

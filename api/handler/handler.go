package handler

import (
	"net/http"

	"github.com/htetmyatthar/server-manager/internal/utils"
)

func ApologyHandler(w http.ResponseWriter, r *http.Request) {
	utils.RenderError(w, nil, http.StatusNotFound)
}

func AdminHandlerGET(w http.ResponseWriter, r *http.Request) {
	utils.RenderTemplate(w, "admin", "something" /*data*/)
}

func AdminHandlerPOST(w http.ResponseWriter, r *http.Request) {
	// TODO: add admin authentication with captcha

	// since this has to be relative path there shouldn't be any "/" infront of the current path.
	http.Redirect(w, r, "admin/dashboard", http.StatusFound)
}

func AdminDashboardGet(w http.ResponseWriter, r *http.Request) {
	utils.RenderTemplate(w, "dashboard", "something" /*data*/)
}

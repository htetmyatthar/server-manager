package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/htetmyatthar/server-manager/internal/config"
	"github.com/htetmyatthar/server-manager/internal/database"
	"github.com/htetmyatthar/server-manager/internal/utils"
)

// added easter egg
func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello LoTone users.")
}

func RedirectToHTTPSHandler(w http.ResponseWriter, r *http.Request) {
	url := "https://" + *config.WebHost + *config.WebPort + r.RequestURI
	http.Redirect(w, r, url, http.StatusMovedPermanently)
	log.Println("Redirected to HTTPS server.")
	return
}

// DefaultHandler is to handle all routes and redirect to the main page.
func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie(config.SessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	return
}

// ApologyHandler is to handle for the undefind paths with an apology.
func ApologyHandler(w http.ResponseWriter, r *http.Request) {
	utils.RenderError(w, "Not Found.", http.StatusNotFound)
}

// AdminLoginGET is to show the admin login page.
func AdminLoginGET(sessionStore data.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(config.SessionCookieName)
		var sessionId string
		// if no cookies login again.
		if err != nil && err == http.ErrNoCookie {
			_, sessionId, err = utils.SessionSetPath(w, "/admin/login", sessionStore)
			if err != nil {
				utils.RenderError(w, "Public session generation gone wrong. Please try again.", http.StatusInternalServerError)
				return
			}
			log.Println("empty sessionvalue to new sessionvalue: ", sessionId)
		} else {
			sessionId = sessionCookie.Value
		}
		// generate token
		token, err := utils.GenerateCSRF(sessionId)
		if err != nil {
			log.Println("csrf generation gone wrong.", err)
			utils.RenderError(w, "CSRF token generation gone wrong", http.StatusInternalServerError)
			return
		}

		data := struct {
			CSRFToken     string
			CSRFTokenName string
			Version       string
		}{
			CSRFToken:     token,
			CSRFTokenName: config.CSRFFormFieldName,
			Version:       config.Version,
		}

		w.WriteHeader(http.StatusOK)
		utils.RenderTemplate(w, "admin", data)
		return
	}
}

// AdminLoginPOST is a handler for logging into the admin dashboard.
func AdminLoginPOST(sessionStore data.SessionStore, userLocker *utils.LockedOutRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie(config.SessionCookieName)
		// if no cookies login again.
		if err == http.ErrNoCookie {
			log.Println("Attempt to access dashboard without the session cookie.")
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if userLocker.IsLockedOut(username) {

			log.Println("Too many failed attempts, ", ip)
			utils.RenderError(w, "Too many failed attempts. Try again later. Contact administrator if needed.", http.StatusTooManyRequests)
			return
		}

		if username == "" || password == "" {
			log.Println("Attempt with empty password or username.")
			utils.RenderError(w, "Invalid format.", http.StatusBadRequest)
			return
		}

		if _, ok := utils.PanelUsers[username]; !ok {
			log.Println("Attempt with wrong username.")
			utils.RenderError(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		correct, err := utils.VerifyPassword(password, utils.PanelUsers[username])
		// handle hashing errors.
		if err != nil && err != utils.ErrWrongPassword {
			log.Println("verifying user password gone wrong.", err)
			utils.RenderError(w, "Internal Server Error.", http.StatusInternalServerError)
			return
		}

		// incorrect password.
		if correct != true {

			err = userLocker.RecordFailedAttempt(username)
			// prepare and send the noti if the user is being locked out.
			if err != nil && err == utils.ErrUserLockedOut {
				// prepare and send a notification
				title := *config.WebHost + " - User locked out"
				message := "User [[" + username + "]] is locked out for " + strconv.Itoa(*config.LockOutDuration) + " minutes"
				for _, key := range config.GotifyAPIKeys {
					utils.SendNoti(*config.GotifyServer, key, title, message, 9)
				}
			}

			log.Println("Attempt with wrong password.")
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// reset password attempts.
		userLocker.ResetAttempts(username)

		// set the session.
		err = utils.SessionSetPrivate(w, "/", sessionStore)
		if err != nil {
			log.Println("session setting gone wrong.", err)
			utils.RenderError(w, "session setting gone wrong", http.StatusInternalServerError)
			return
		}

		// send a notification to the gotify server.
		title := *config.WebHost + " - " + username + " logged in"
		message := username + " logged into " + *config.WebHostIP + " using " + ip
		for _, key := range config.GotifyAPIKeys {
			utils.SendNoti(*config.GotifyServer, key, title, message, 9)
		}

		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		return
	}
}

// AdminDashboardGET is to show the admin dashboard.
func AdminDashboardGET(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie(config.SessionCookieName)
	if err == http.ErrNoCookie { // if no cookies login again.
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	token, err := utils.GenerateCSRF(session.Value)
	if err != nil {
		log.Println("csrf generation gone wrong.", err)
		utils.RenderError(w, "CSRF token generation gone wrong", http.StatusInternalServerError)
		return
	}

	userData, err := os.ReadFile(*config.UserFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		utils.RenderError(w, "Unable to read the config file.", http.StatusInternalServerError)
		return
	}

	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println(err)
		return
	}

	var users []utils.Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("marshalling into the utils.client gone wrong: ,", err)
		return
	}

	data := struct {
		Clients         []utils.Client
		ServerRegion    string
		ServerIP        string
		V2rayServerPort string
		CSRFToken       string
		CSRFTokenName   string
	}{
		Clients:         users,
		ServerRegion:    *config.WebHostRegion,
		ServerIP:        *config.WebHostIP,
		V2rayServerPort: *config.V2rayPort,
		CSRFToken:       token,
		CSRFTokenName:   config.CSRFFormFieldName,
	}

	w.WriteHeader(http.StatusOK)
	utils.RenderTemplate(w, "dashboard", data)
}

// AccountCreatePOST is to create an new user account and add it to the v2ray server configuration file.
func AccountCreatePOST(w http.ResponseWriter, r *http.Request) {

	// load the config file.
	configData, err := os.ReadFile(*config.ConfigFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		utils.RenderError(w, "Unable to read the config file.", http.StatusInternalServerError)
		return
	}

	// load the users file.
	userData, err := os.ReadFile(*config.UserFile)
	if err != nil {
		fmt.Println("Error reading user data file: ", err)
		utils.RenderError(w, "Unable to read the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON config file into a map
	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON users file into a map
	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "inbounds" key into a slice of Inbound structs
	var inbounds []utils.Inbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "users" key into a slice of clients.
	var users []utils.Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	// modify the inbounds by adding a new client with default AlterId of value 1.
	newV2rayClient := utils.V2rayClient{
		Id:      r.FormValue("serverUUID"),
		AlterId: 1,
	}

	// modify the users by adding a new user entity to the users file.
	newClient := utils.Client{
		Id:         r.FormValue("serverUUID"),
		AlterId:    1,
		Username:   r.FormValue("username"),
		DeviceId:   r.FormValue("deviceUUID"),
		StartDate:  r.FormValue("startDate"),
		ExpireDate: r.FormValue("expireDate"),
	}

	inbounds[0].Settings.Clients = append(inbounds[0].Settings.Clients, newV2rayClient)
	users = append(users, newClient)

	// marshal the modified inbounds back to JSON
	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}
	// update the original result map with the modified inbounds
	configResult["inbounds"] = inboundBytes

	// marshal the modified users back to JSON
	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}
	userResult["clients"] = usersBytes

	// marshal the entire result map back to JSON
	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// marshal the entire users result map back to JSON
	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.ConfigFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.UserFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// prepare and send a push notification
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println("Error getting the ip address of the requester.")
		utils.RenderError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	title := *config.WebHost + " - New user is created"
	message := newClient.Username + "@" + *config.WebHostIP + " with [[" + newClient.Id + "]] is created by " + ip
	for _, key := range config.GotifyAPIKeys {
		utils.SendNoti(*config.GotifyServer, key, title, message, 5)
	}

	// since this has to be relative path there shouldn't be any "/" infront of(admin) the current path.
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

func AccountDeletePOST(w http.ResponseWriter, r *http.Request) {

	// load the config file.
	configData, err := os.ReadFile(*config.ConfigFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		utils.RenderError(w, "Unable to read the config file.", http.StatusInternalServerError)
		return
	}

	// load the users file.
	userData, err := os.ReadFile(*config.UserFile)
	if err != nil {
		fmt.Println("Error reading user data file: ", err)
		utils.RenderError(w, "Unable to read the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON config file into a map
	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON users file into a map
	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "inbounds" key into a slice of Inbound structs
	var inbounds []utils.Inbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "users" key into a slice of clients.
	var users []utils.Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	username := r.FormValue("username")
	serverUUID := r.FormValue("serverUUID")
	userNumber, err := strconv.Atoi(r.FormValue("userNumber"))
	if err != nil {
		log.Println("Error getting the user number.")
		utils.RenderError(w, "Error getting the user's order number in configuration.", http.StatusBadRequest)
		return
	}

	if inbounds[0].Settings.Clients[userNumber].Id != serverUUID && users[userNumber].Username != username && users[userNumber].Id != serverUUID {
		log.Println("Error invoking user deletion with incorrect information")
		utils.RenderError(w, "Incorrect user information!", http.StatusBadRequest)
		return
	}

	// store for future references.
	deletedUser := users[userNumber]

	inbounds[0].Settings.Clients = append(inbounds[0].Settings.Clients[:userNumber], inbounds[0].Settings.Clients[userNumber+1:]...)
	users = append(users[:userNumber], users[userNumber+1:]...)

	// marshal the modified inbounds back to JSON
	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}
	// update the original result map with the modified inbounds
	configResult["inbounds"] = inboundBytes

	// marshal the modified users back to JSON
	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}
	userResult["clients"] = usersBytes

	// marshal the entire result map back to JSON
	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// marshal the entire users result map back to JSON
	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.ConfigFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.UserFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// prepare and send a push notification
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println("Error getting the ip address of the requester.")
		utils.RenderError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	title := *config.WebHost + " - Existing user is deleted."
	message := deletedUser.Username + "@" + *config.WebHostIP + " with [[" + deletedUser.Id + "]] is deleted by " + ip
	for _, key := range config.GotifyAPIKeys {
		utils.SendNoti(*config.GotifyServer, key, title, message, 5)
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

func ServerIPHandlerGET(w http.ResponseWriter, r *http.Request) {
	utils.JSONRespond(w, http.StatusOK, map[string]string{"ip": *config.WebHostIP})
}

// Request body structure
type RestartRequest struct {
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`
}

// ServerRestartPOST handles to restart the v2ray server when invoked with correct password.
func ServerRestartPOST(w http.ResponseWriter, r *http.Request) {
	var restartRequest RestartRequest

	err := json.NewDecoder(r.Body).Decode(&restartRequest)
	if err != nil {
		utils.JSONRespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, ok := utils.PanelUsers[restartRequest.AdminUsername]; !ok {
		log.Println("Attempt with wrong username.")
		utils.JSONRespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	_, err = utils.VerifyPassword(restartRequest.AdminPassword, utils.PanelUsers[restartRequest.AdminUsername])
	if err != nil {
		utils.JSONRespondError(w, http.StatusUnauthorized, "Unauthorized")
		log.Println("someone tries to restart the server with this password.", restartRequest.AdminPassword)
		return
	}

	// TODO: make the rate limit for each panel user and lock out the user for the wrong password.

	err = utils.ValidateConfig()
	if err != nil {
		utils.JSONRespondError(w, http.StatusInternalServerError, "Config validation failed.")
		log.Println("config failed.", err)
		return
	}

	err = utils.RestartService()
	if err != nil {
		utils.JSONRespondError(w, http.StatusInternalServerError, "Failed to restart v2ray server.")
		log.Println("restart server by web ui failed.", err)
		return
	}

	// prepare and send push notification
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		utils.RenderError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	title := *config.WebHost + " - v2ray server " + *config.WebHostIP + " restarted."
	message := "server " + *config.WebHostIP + " restarted by " + ip
	for _, key := range config.GotifyAPIKeys {
		utils.SendNoti(*config.GotifyServer, key, title, message, 9)
	}

	utils.JSONRespond(w, http.StatusOK, "V2ray service restarted successfully.")
}

func AccountEditPOST(w http.ResponseWriter, r *http.Request) {
	// load the config file.
	configData, err := os.ReadFile(*config.ConfigFile)
	if err != nil {
		fmt.Println("Error reading file:", err)
		utils.RenderError(w, "Unable to read the config file.", http.StatusInternalServerError)
		return
	}

	// load the users file.
	userData, err := os.ReadFile(*config.UserFile)
	if err != nil {
		fmt.Println("Error reading user data file: ", err)
		utils.RenderError(w, "Unable to read the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON config file into a map
	var configResult map[string]json.RawMessage
	err = json.Unmarshal(configData, &configResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in config:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the JSON users file into a map
	var userResult map[string]json.RawMessage
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		log.Println("Error unmarshalling JSON to map in users:", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "inbounds" key into a slice of Inbound structs
	var inbounds []utils.Inbound
	err = json.Unmarshal(configResult["inbounds"], &inbounds)
	if err != nil {
		log.Println("Error unmarshalling 'inbounds':", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// unmarshal the "users" key into a slice of clients.
	var users []utils.Client
	err = json.Unmarshal(userResult["clients"], &users)
	if err != nil {
		log.Println("Error unmarshalling 'users': ", err)
		utils.RenderError(w, "Error adding a new client to the users file.", http.StatusInternalServerError)
		return
	}

	// modify the inbounds by adding a new client with default AlterId of value 1.
	modifiedV2rayClient := utils.V2rayClient{
		Id:      r.FormValue("serverUUID"),
		AlterId: 1,
	}

	// modify the users by adding a new user entity to the users file.
	modifiedClient := utils.Client{
		Id:         r.FormValue("serverUUID"),
		AlterId:    1,
		Username:   r.FormValue("username"),
		DeviceId:   r.FormValue("deviceUUID"),
		StartDate:  r.FormValue("startDate"),
		ExpireDate: r.FormValue("expireDate"),
	}

	userNumber, err := strconv.Atoi(r.FormValue("userNumber"))
	if err != nil {
		log.Println("Error converting usernumber into integer", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	if userNumber >= len(inbounds[0].Settings.Clients) {
		log.Println("Invalid userNumber is assessed.")
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// change with the modified user.
	inbounds[0].Settings.Clients[userNumber] = modifiedV2rayClient
	users[userNumber] = modifiedClient

	// marshal the modified inbounds back to JSON
	inboundBytes, err := json.Marshal(inbounds)
	if err != nil {
		log.Println("Error marshalling modified inbounds:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}
	// update the original result map with the modified inbounds
	configResult["inbounds"] = inboundBytes

	// marshal the modified users back to JSON
	usersBytes, err := json.Marshal(users)
	if err != nil {
		log.Println("Error marshalling modified users:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}
	userResult["clients"] = usersBytes

	// marshal the entire result map back to JSON
	finalConfigJSON, err := json.MarshalIndent(configResult, "", "  ")
	if err != nil {
		log.Println("Error marshalling final config JSON:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// marshal the entire users result map back to JSON
	finalUserJSON, err := json.MarshalIndent(userResult, "", " ")
	if err != nil {
		log.Println("Error marshalling final users JSON:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.ConfigFile, finalConfigJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new client to the existings configuration.", http.StatusInternalServerError)
		return
	}

	// Optional: Write the modified JSON back to a file
	err = os.WriteFile(*config.UserFile, finalUserJSON, 0644)
	if err != nil {
		log.Println("Error writing modified JSON to file:", err)
		utils.RenderError(w, "Error adding a new user to the users file.", http.StatusInternalServerError)
		return
	}

	// prepare and send a push notification
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println("Error getting the ip address of the requester.")
		utils.RenderError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	title := *config.WebHost + " - User is updated"
	message := modifiedClient.Username + "@" + *config.WebHostIP + " with [[" + modifiedClient.Id + "]] is updated by " + ip
	for _, key := range config.GotifyAPIKeys {
		utils.SendNoti(*config.GotifyServer, key, title, message, 5)
	}

	// since this has to be relative path there shouldn't be any "/" infront of(admin) the current path.
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

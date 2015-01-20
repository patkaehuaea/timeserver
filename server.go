//  Copyright (C) Pat Kaehuaea - All Rights Reserved
//  Unauthorized copying of this file, via any medium is strictly prohibited
//  Proprietary and confidential
//  Written by Pat Kaehuaea, January 2015
//
// Package contains simple web server that binds to port 8080. Exectuable accepts
// two parameters, --port to designate listen port, and -V to output the version
// number of the program. Server provieds '/time' endpoint as well as '/login' '/logout'
// and root pages '/', 'index.html'. Pages are rendered from templates that must be
// located in a 'templates/' directory relative to the executable. This package uses
// adjacent people package to maintain state as it relates to visits. State is lost
// upon program termination.
package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/patkaehuaea/server/people"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	VERSION_NUMBER = "v1.0.8"
	TIME_LAYOUT    = "3:04:05 PM"
	COOKIE_NAME    = "uuid"
	COOKIE_MAX_AGE = 86400
)

var cwd, _ = os.Getwd()
var templates = template.Must(template.ParseGlob(filepath.Join(cwd, "templates", "*.html")))
var users = people.NewUsers()

// Debug(..) and Log(..) functions simply wrap log calls
// with fields. Possible to define custom formatter on logrus
// library at later time.
func debug(msg string, r *http.Request) {
	log.WithFields(log.Fields{
		"header":      r.Header["Cookie"],
		"remote addr": r.RemoteAddr,
		"method":      r.Method,
		"time":        time.Now().Format(TIME_LAYOUT),
		"url":         r.URL,
	}).Debug(msg)
}

func handleDefault(w http.ResponseWriter, r *http.Request) {
	info("Default handler called.", r)
	id, _ := idFromUUIDCookie(r)
	if name := users.Name(id); name != "" {
		log.Debug("User: " + name + " viewing site.")
		renderTemplate(w, "greetings", name)
	} else {
		log.Debug("No cookie found or value empty. Redirecting to login.")
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// Handling GET and POST methods can be implemented on separate /login
// handlers with mux. Left as-is for clarity of flow.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	info("Login handler called.", r)
	if r.Method == "GET" {
		log.Debug("Login GET method detected.")
		renderTemplate(w, "login", nil)
	} else if r.Method == "POST" {
		log.Debug("Login POST method detected.")
		name := r.FormValue("name")
		// Allows first name, or first and last name in English characters with intervening space.
		// Minimum length of name is two characters and maximum length of field is 71 characters
		// including space.
		if valid, _ := regexp.MatchString("^[a-zA-Z]{2,35} {0,1}[a-zA-Z]{0,35}$", name); valid {
			log.Debug("Name matched regex.")
			// uuid := uuid()
			person := people.NewPerson(name)
			users.Add(person)
			setCookie(w, person.ID, COOKIE_MAX_AGE)
			http.Redirect(w, r, "/", http.StatusFound)
			log.Debug("User: " + person.Name + " logged in.")
			return
		} else {
			log.Debug("Invalid username. Rendering login page.")
			w.WriteHeader(http.StatusBadRequest)
			renderTemplate(w, "login", "C'mon, I need a name.")
		}
	} else {
		log.Debug("Login request method not handled.")
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	info("Logout handler called.", r)
	// Invalidate data along and set MaxAge to avoid accidental persistence issues.
	setCookie(w, "deleted", -1)
	renderTemplate(w, "logged-out", nil)
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	info("Not found handler called.", r)
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404", nil)
}

func handleTime(w http.ResponseWriter, r *http.Request) {
	info("Time handler called.", r)
	id, _ := idFromUUIDCookie(r)
	// Personalized message will only display if user's cookie contains an id
	// and that id is found in the users table. Template handles display logic.
	params := map[string]interface{}{"time": time.Now().Format(TIME_LAYOUT), "name": users.Name(id)}
	renderTemplate(w, "time", params)
}

func idFromUUIDCookie(r *http.Request) (string, error) {
	log.Debug("Reading cookie 'uuid'.")
	cookie, err := r.Cookie(COOKIE_NAME)
	if err == http.ErrNoCookie {
		log.Debug("Cookie not found.")
		return "", http.ErrNoCookie
	}
	return cookie.Value, nil
}

func info(msg string, r *http.Request) {
	log.WithFields(log.Fields{
		"header":      r.Header["Cookie"],
		"remote addr": r.RemoteAddr,
		"method":      r.Method,
		"time":        time.Now().Format(TIME_LAYOUT),
		"url":         r.URL,
	}).Info(msg)
}

// credit: https://golang.org/doc/articles/wiki/#tmp_10
func renderTemplate(w http.ResponseWriter, templ string, d interface{}) {
	err := templates.ExecuteTemplate(w, templ+".html", d)
	if err != nil {
		log.Fatal("Error looking for template: " + templ)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// The maxAge parameter allows use of a single method to set and delete cookie.
// Default cookie valid for 1 day. Set age to -1 for deletion.
func setCookie(w http.ResponseWriter, uuid string, maxAge int) {
	c := http.Cookie{Name: COOKIE_NAME, Value: uuid, Path: "/", MaxAge: maxAge}
	http.SetCookie(w, &c)
}

func main() {
	portPtr := flag.String("port", "8080", "Web server binds to this port. Default is 8080.")
	verbosePtr := flag.Bool("V", false, "Prints version number of program.")
	flag.Parse()
	portParam := ":" + *portPtr

	if *verbosePtr {
		fmt.Printf("Version number: %s \n", VERSION_NUMBER)
		os.Exit(1)
	}

	log.SetLevel(log.InfoLevel)

	r := mux.NewRouter()
	r.HandleFunc("/", handleDefault)
	r.HandleFunc("/index.html", handleDefault)
	r.HandleFunc("/login", handleLogin)
	r.HandleFunc("/logout", handleLogout)
	r.HandleFunc("/time", handleTime)
	r.NotFoundHandler = http.HandlerFunc(handleNotFound)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(portParam, nil))
}

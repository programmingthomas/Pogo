package server

import (
	"bytes"
	"fmt"
	"github.com/programmingthomas/Pogo/catcher"
	"github.com/programmingthomas/Pogo/pogoutils"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"time"
)

//A generic type that is used when serving up pages that require the standard template
type Page struct {
	URL     string
	Content template.HTML
	Title   string
}

type Podcast struct {
	Name string
}

var templates = template.Must(template.ParseFiles("server/templates/index.html", "server/templates/welcome.html", "server/templates/about.html", "server/templates/addfeed.html", "server/templates/podcast.html", "server/templates/episode.html"))
var PodCatcher catcher.Catcher

//Handles the CSS, JS and Bootstrap resources
func resHandler(w http.ResponseWriter, r *http.Request) {
	_, filename := path.Split(r.URL.Path)
	fullPath := "server/res/" + filename
	fileExists := pogoutils.FileExists(fullPath)
	if fileExists {
		lastModTime := pogoutils.LastMod(fullPath)
		//This checks whether or not the Header was submitted with
		//If-Modified-Since, which reduces server IO, only do if Cache is enabled
		if r.Header["If-Modified-Since"] != nil && Cache {
			//RFC1123 is the standard date format used with HTTP
			headerTime, _ := time.Parse(time.RFC1123, r.Header["If-Modified-Since"][0])
			if !headerTime.Before(lastModTime) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		//Writer the header and content
		if Cache {
			w.Header().Add("Last-Modified", lastModTime.Format(time.RFC1123))
		}
		//Go has a function for serving files easily
		//I used this function because it reduces the complexity of the code
		//And it seems to do a good job handling MIME types
		http.ServeFile(w, r, fullPath)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Println(fullPath, "not found")
	}
}

//Serves the homepage
func homeHandler(w http.ResponseWriter, r *http.Request) {
	page := Page{URL: fmt.Sprintf("%s:%d", Path, Port), Title: "Pogo"}
	content := bytes.NewBufferString("")
	//Update podcast list
	PodCatcher.UpdateAll()
	templates.ExecuteTemplate(content, "welcome.html", PodCatcher.Podcasts)
	page.Content = template.HTML(content.String())
	pageHandler(page, "index.html", w)
}

//Serves the really exciting about page
func aboutHandler(w http.ResponseWriter, r *http.Request) {
	page := Page{URL: fmt.Sprintf("%s:%d", Path, Port), Title: "About - Pogo"}
	content := bytes.NewBufferString("")
	templates.ExecuteTemplate(content, "about.html", nil)
	page.Content = template.HTML(content.String())
	pageHandler(page, "index.html", w)
}

//Serves the add podcast page and if the request is a POST request it will subscribe to the
//podcast in the 'feedurl' parameter
func addPodcastHandler(w http.ResponseWriter, r *http.Request) {
	//I.e. add a podcast feed
	if r.Method == "POST" {
		if r.FormValue("feedurl") != "" {
			//Check if the URL is a valid URL
			feedURL, err := url.Parse(r.FormValue("feedurl"))
			if err == nil {
				//Concurrent???
				go PodCatcher.AddPodcastFeed(feedURL.String())
			}
		}
	}
	page := Page{URL: fmt.Sprintf("%s:%d", Path, Port), Title: "Add podcast - Pogo"}
	content := bytes.NewBufferString("")
	templates.ExecuteTemplate(content, "addfeed.html", nil)
	page.Content = template.HTML(content.String())
	pageHandler(page, "index.html", w)
}

//Generic page handler contains the main template
func pageHandler(page Page, template string, w http.ResponseWriter) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	err := templates.ExecuteTemplate(w, template, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//Allows you to download the configuration file in case you wanted to build something on
//top of Pogo (like a mobile app)
func pogoConfigHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "pogoconfig.json")
}

//Serves up a page with info for a certain podcast
func podcastHandler(w http.ResponseWriter, r *http.Request) {
	base := path.Base(r.URL.Path)
	if base != "podcasts" {
		for _, podcast := range PodCatcher.Podcasts {
			if podcast.ID == base {
				page := Page{URL: fmt.Sprintf("%s:%d", Path, Port), Title: podcast.Name + " - Pogo"}
				content := bytes.NewBufferString("")
				templates.ExecuteTemplate(content, "podcast.html", podcast)
				page.Content = template.HTML(content.String())
				pageHandler(page, "index.html", w)
				break
			}
		}
	}
}

//Serves up a page with the info for an individual podcast
func episodeHandler(w http.ResponseWriter, r *http.Request) {
	PodCatcher.UpdateAll()
	if r.FormValue("episode") != "" {
		for _, podcast := range PodCatcher.Podcasts {
			for _, episode := range podcast.PodcastEpisodes {
				if episode.URL == r.FormValue("episode") {
					page := Page{URL: fmt.Sprintf("%s:%d", Path, Port), Title: episode.Title + " - Pogo"}
					content := bytes.NewBufferString("")
					templates.ExecuteTemplate(content, "episode.html", episode)
					page.Content = template.HTML(content.String())
					pageHandler(page, "index.html", w)
					return
				}
			}
		}
	}
}

//Serves up video/audio
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	_, filename := path.Split(r.URL.Path)
	if pogoutils.FileExists("downloads/" + filename) {
		fmt.Println("Serving downloads/", filename)
		http.ServeFile(w, r, "downloads/"+filename)
	}
}

//Start the Pogo server
func Start() {
	fmt.Println("Starting Pogo server")
	PodCatcher = catcher.StartCatcher("pogoconfig.json")
	http.HandleFunc("/js/", resHandler)
	http.HandleFunc("/css/", resHandler)
	http.HandleFunc("/res/", resHandler)
	http.HandleFunc("/img/", resHandler)
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/home", homeHandler)
	http.HandleFunc("/index", homeHandler)
	http.HandleFunc("/episode/", episodeHandler)
	http.HandleFunc("/downloads/", downloadHandler)
	http.HandleFunc("/podcasts/add", addPodcastHandler)
	http.HandleFunc("/pogo.json", pogoConfigHandler)
	http.HandleFunc("/podcast/", podcastHandler)
	http.HandleFunc("/about", aboutHandler)
	http.ListenAndServe(fmt.Sprintf(":%d", Port), nil)
}

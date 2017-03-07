package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// GLOBALS DECLARED HERE

var filePaths = MediaDir{}
var tmplDir = "./templates/"
var templates map[string]*template.Template
var pageData = PageData{}
var firstStart = true

// THE MODEL CODE IS HERE

type MediaDir struct {
	MediaDirs []string
}

type Movie struct {
	FullFilePath string
	FileName     string
}

type PageData struct {
	MovieList   []Movie
	CurrentFilm string
	Player      Player
}

// LOOKS FOR FILES ON THE FILESYSTEM

var extensionList = [][]byte{
	{'.', 'm', 'k', 'v'},
	{'.', 'm', 'p', 'g'},
	{'.', 'a', 'v', 'i'},
	{'.', 'm', '4', 'v'},
	{'.', 'm', 'p', '4'}}

func visit(path string, f os.FileInfo, err error) error {
	bpath := []byte(strings.ToLower(path))
	bpath = bpath[len(bpath)-4:]
	for i := 0; i < len(extensionList); i++ {
		if reflect.DeepEqual(bpath, extensionList[i]) {
			movie := Movie{path, f.Name()}
			pageData.MovieList = append(pageData.MovieList, movie)
		}
	}
	return nil
}

func generateMovies(filePaths []string) error {
	if len(filePaths) > 0 {
		for _, path := range filePaths {
			err := filepath.Walk(path, visit)
			if err != nil {
				return err
			} else if len(pageData.MovieList) <= 0 {
				return fmt.Errorf("No media files were found in the given path: %s", path)
			}
		}
	} else {
		return fmt.Errorf("No file paths to process.")
	}
	fmt.Printf("file import complete: %d files imported\n", len(pageData.MovieList))
	return nil
}

// THE VIEW CODE IS HERE

func generateTemplates() {
	templates = make(map[string]*template.Template)
	modulus := template.FuncMap{"mod": func(i, j int) bool { return i%j == 0 }}
	templatesList := []string{"index.html", "about.html", "movie.html", "alreadyplaying.html", "setup.html", "nothingfound.html"}
	for _, tmpl := range templatesList {
		t := template.New("base.html").Funcs(modulus)
		templates[tmpl] = template.Must(t.ParseFiles(tmplDir+"base.html", tmplDir+tmpl))
	}
}

func renderTemplate(pageStruct interface{}, w http.ResponseWriter, tmpl string) {
	var err error
	if pageStruct == nil {
		err = templates[tmpl].Execute(w, pageData)
	} else {
		err = templates[tmpl].Execute(w, pageStruct)
	}
	if err != nil {
		log.Printf("The follwing error occurred: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func refreshList() error {
	if pageData.Player.Playing {
		player := pageData.Player
		currentFilm := pageData.CurrentFilm
		pageData = PageData{}
		err := generateMovies(filePaths.MediaDirs)
		pageData.CurrentFilm = currentFilm
		pageData.Player = player
		return err
	}
	pageData = PageData{}
	err := generateMovies(filePaths.MediaDirs)
	return err

}

// HANDLERS ARE HERE

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if firstStart {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	var tmpl string
	if pageData.Player.Playing {
		tmpl = "alreadyplaying.html"
	} else {
		tmpl = "index.html"
	}
	if r.Method == "POST" {
		err := refreshList()
		if err != nil {
			panic(err)
		}
	}
	renderTemplate(nil, w, tmpl)
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(nil, w, "about.html")
}

func setupHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := "setup.html"
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	if r.Method == "POST" {
		filePaths.MediaDirs = append(filePaths.MediaDirs, r.Form["filepath"][0])
		if err := refreshList(); err != nil {
			tmpl = "nothingfound.html"
		} else {
			firstStart = false
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}
	renderTemplate(filePaths, w, tmpl)
}

func movieHandler(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Query().Get("command")
	film := r.URL.Query().Get("movie")

	if pageData.Player.Playing == false {
		if film == "" {
			log.Println("No film was selected")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		err := pageData.Player.StartFilm(film)
		if err != nil {
			log.Printf("Following error occurred: %v\n", err)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		pageData.CurrentFilm = film
	} else if pageData.Player.Playing && (film == "" || pageData.Player.FilmName == film) {
		if command == "kill" {
			err := pageData.Player.EndFilm()
			if err != nil {
				log.Printf("Following error occurred: %v\n", err)
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		} else if command != "" {
			err := pageData.Player.SendCommandToFilm(command)
			if err != nil {
				log.Printf("Following error occurred: %v\n", err)
			}
		}
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	renderTemplate(nil, w, "movie.html")
}

// IT ALL STARTS HERE

func main() {
	generateTemplates()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/setup", setupHandler)
	http.HandleFunc("/movie", movieHandler)
	http.ListenAndServe(":8080", nil)
}

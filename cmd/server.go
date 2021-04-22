package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/sholiday/tinycast"
)

type embedFileSystem struct {
	http.FileSystem
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	_, err := e.Open(path)
	if err != nil {
		return false
	}
	return true
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	fsys, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return embedFileSystem{
		FileSystem: http.FS(fsys),
	}
}

func maybeStartStatusServer() {
	host := os.Getenv("DEBUG_HOST")
	port := os.Getenv("DEBUG_PORT")
	if port != "" {
		statusHostPort := fmt.Sprintf("%s:%s", host, port)
		log.Printf("Launching status server at '%s", statusHostPort)
		go http.ListenAndServe(statusHostPort, http.DefaultServeMux)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	maybeStartStatusServer()

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	serverBind := fmt.Sprintf("%s:%s", host, port)

	r := gin.Default()

	tmpl, err := template.ParseFS(tinycast.Templates, "templates/*.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	r.SetHTMLTemplate(tmpl)

	baseUrlString := os.Getenv("BASE_URL")
	if baseUrlString == "" {
		baseUrlString = fmt.Sprintf("http://localhost:%s/", port)
	}
	baseUrl, err := url.Parse(baseUrlString)
	if err != nil {
		log.Fatal(err)
	}
	apiKey := os.Getenv("API_KEY")
	app := tinycast.NewApp(*baseUrl, apiKey)
	r.GET("/", app.Home)
	r.GET("/convert.mp3", app.Get)
	r.GET("/feed", app.Feed)
	r.Use(static.Serve("/", EmbedFolder(tinycast.Templates, "templates/favicon")))

	log.Printf("About to start main server at '%s'", serverBind)

	r.Run(serverBind)
}

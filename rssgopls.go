package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gocolly/colly"
	"github.com/robfig/cron/v3"
)

type Post struct {
	Title string
	Link  string
	Date  string
}

type Channel struct {
	Title       string
	Link        string
	Date        string
	Description string
	Posts       []Post
}

var tpl = `
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>{{ .Title }}</title>
    <link>{{ .Link }}</link>
    <description>Recent content on {{ .Title }}</description>
    <generator>Hugo -- gohugo.io</generator>
    <language>en-us</language>
    <lastBuildDate>{{ .Date }}</lastBuildDate>
    <atom:link href="{{ .Link }}" rel="self" type="application/rss+xml"/>
    {{ range .Posts }}
    <item>
      <title>{{ .Title }}</title>
      <link>{{ .Link }}</link>
      <pubDate>{{ .Date }}</pubDate>
      <guid>{{ .Link }}</guid>
    </item>
    {{ end }}
  </channel>
</rss>
`
var serveDir string

func init() {
	flag.StringVar(&serveDir, "serve", "", "directory to serve")
	flag.Parse()
}

func main() {
	fetchRss()

	at := cron.New()

	if _, err := at.AddFunc("32 8,12,16,20 * * *", fetchRss); err != nil {
		fmt.Printf("unable to set cron: %s\n", err)
	}

	at.Start()

	serveFile()
}

// http serve static rss xml file
func serveFile() {
	fs := http.FileServer(http.Dir(filepath.Join(serveDir)))

	// Create a handler function that wraps the file server handler and logs the requests
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		fs.ServeHTTP(w, r)
	}

	// Start the server with the handler function
	log.Print("Listening on :3786...")
	err := http.ListenAndServe(":3786", http.HandlerFunc(handler))
	if err != nil {
		log.Fatal(err)
	}
}

func fetchRss() {
	t, err := template.New("rss").Parse(tpl)
	if err != nil {
		fmt.Println(err)
		return
	}

	posts := []Post{}

	c := colly.NewCollector()

	// Find and visit all links
	c.OnHTML(".crayons-story", func(e *colly.HTMLElement) {
		post := Post{}
		post.Title = e.ChildText(".crayons-story__title")
		post.Link = e.ChildAttr("a", "href")
		post.Date = e.ChildAttr("time", "datetime")
		posts = append(posts, post)
		// e.Request.Visit(e.Attr("href"))
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	err = c.Visit("https://dev.to/top/week")
	if err != nil {
		fmt.Println(err)
	}

	channel := Channel{
		Title: "dev.to - top of the week",
		Link:  "https://dev.to/top/week",
		Date:  time.Now().UTC().Format(time.RFC3339),
		Posts: posts,
	}

	f, err := os.Create(filepath.Join(serveDir, "rss.xml"))
	if err != nil {
		log.Printf("unable to create file: %s\n", err)
		return
	}
	defer f.Close()

	err = t.Execute(f, channel)
	if err != nil {
		fmt.Println(err)
		return
	}
}

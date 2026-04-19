// Command e64ec-server serves the embedded static site over HTTP.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"go.e64ec.com/e64ec/site"
)

func main() {
	addr := flag.String("addr", defaultAddr(), "listen address")
	flag.Parse()

	fsys, err := site.FS()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler(http.FS(fsys)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("e64ec serving on %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handler(fsys http.FileSystem) http.Handler {
	return http.FileServer(fsys)
}

func defaultAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8080"
}

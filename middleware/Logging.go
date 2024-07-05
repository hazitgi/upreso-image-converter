package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println()
		start := time.Now()
		log.Printf("Started %s \t %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
		log.Printf("Completed \t\t %s in %v", r.URL.Path, time.Since(start))
	})
}

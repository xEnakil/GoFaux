package server

import (
	"fmt"
	"net/http"
	"sync"

	"GoFaux/api"
)

var serverRunning sync.Once

func StartServer() {
	serverRunning.Do(func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			apis := api.GetAllAPIs()
			for _, mockAPI := range apis {
				if r.URL.Path == mockAPI.Endpoint && r.Method == mockAPI.Method {
					api.IncrementRequestCount(mockAPI.Endpoint)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(mockAPI.Response))
					return
				}
			}
			http.NotFound(w, r)
		})
		fmt.Println("\n🚀 Mock API Server running at http://localhost:8080")
		http.ListenAndServe(":8080", nil)
	})
}
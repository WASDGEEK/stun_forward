// signal_server.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

var (
	dataStore = make(map[string]map[string]string)
	mutex     = &sync.Mutex{}
)

func signalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var data SignalData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		mutex.Lock()
		if _, ok := dataStore[data.Room]; !ok {
			dataStore[data.Room] = make(map[string]string)
		}
		dataStore[data.Room][data.Role] = data.Data
		mutex.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	} else if r.Method == "GET" {
		room := r.URL.Query().Get("room")
		role := r.URL.Query().Get("role")
		if room == "" || role == "" {
			http.Error(w, "Missing room or role", http.StatusBadRequest)
			return
		}
		peer := "sender"
		if role == "sender" {
			peer = "receiver"
		}
		mutex.Lock()
		data, ok := dataStore[room][peer]
		mutex.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(data))
	} else {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func runServer(port string) {
	http.HandleFunc("/", signalHandler)
	log.Printf("Signal server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

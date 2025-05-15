package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/query", queryHandler)

	fmt.Println("Slave is running on port 8001")
	http.ListenAndServe(":8001", nil)
}

type QueryRequest struct {
	DBName string `json:"dbname"`
	Query  string `json:"query"`
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to encode request", http.StatusInternalServerError)
		return
	}

	fmt.Println("Slave is sending a query to the master")

	resp, err := http.Post("http://192.168.137.61:8000/execute_query", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		http.Error(w, "Failed to contact master: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read master response", http.StatusInternalServerError)
		return
	}

	w.Write(body)
}

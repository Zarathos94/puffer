package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Zarathos94/puffer/utils"
)

func RegisterRateRoutes(rs *utils.RateService) {
	http.HandleFunc("/sse/rate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ctx := r.Context()
		enc := json.NewEncoder(w)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			update, err := rs.GetLatest()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
				w.(http.Flusher).Flush()
				time.Sleep(15 * time.Second)
				continue
			}
			fmt.Fprintf(w, "data: ")
			enc.Encode(update)
			fmt.Fprintf(w, "\n")
			w.(http.Flusher).Flush()
			time.Sleep(15 * time.Second)
		}
	})

	http.HandleFunc("/rate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		update, err := rs.GetLatest()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(update)
	})

	http.HandleFunc("/rate/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		to := time.Now().Unix()
		from := to - 24*60*60
		history, err := rs.GetHistory(from, to)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(history)
	})
}

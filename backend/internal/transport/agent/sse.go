package agent

import (
	"fmt"
	"net/http"
)

func WriteSSEConnected(w http.ResponseWriter) {
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func WriteSSEHeartbeat(w http.ResponseWriter) {
	fmt.Fprintf(w, ": heartbeat\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

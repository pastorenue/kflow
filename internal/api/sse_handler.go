package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var execIDRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func (s *Server) handleExecutionStream(w http.ResponseWriter, r *http.Request) {
	execID := r.PathValue("id")
	if !execIDRe.MatchString(execID) {
		writeError(w, http.StatusBadRequest, "invalid execution id", "bad_request")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported", "internal")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := s.Hub.subscribe()
	defer s.Hub.unsubscribe(ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.Type != "state_transition" {
				continue
			}
			p, ok := ev.Payload.(StateTransitionPayload)
			if !ok || p.ExecutionID != execID {
				continue
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

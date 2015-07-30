package gossip

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

var defaultNumPollinationsToReturn = flag.Int("default_num_pollinations_to_return", 10,
	"Number of randomly selected STH pollination entries to return for sth-pollination requests.")

// Handler for the gossip HTTP requests.
type Handler struct {
	storage *Storage
}

// HandleSCTFeedback handles requests POSTed to .../sct-feedback.
// It attempts to store the provided SCT Feedback
func (h *Handler) HandleSCTFeedback(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		rw.Header().Add("Allow", "POST")
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var feedback SCTFeedback
	if err := decoder.Decode(&feedback); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(fmt.Sprintf("Invalid SCT Feedback received: %v", err)))
		return
	}

	// TODO(alcutter): 5.1.1 Validate leaf chains up to a trusted root
	// TODO(alcutter): 5.1.1/2 Verify each SCT is valid and from a known log, discard those which aren't
	// TODO(alcutter): 5.1.1/3 Discard leaves for domains other than ours.
	if err := h.storage.AddSCTFeedback(feedback); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("Unable to store feedback: %v", err)))
		return
	}
	rw.WriteHeader(http.StatusOK)
}

// HandleSTHPollination handles requests POSTed to .../sth-pollination.
// It attempts to store the provided pollination info, and returns a random set of
// pollination data from the last 14 days (i.e. "fresh" by the definition of the gossip RFC.)
func (h *Handler) HandleSTHPollination(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		rw.Header().Add("Allow", "POST")
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var p STHPollination
	if err := decoder.Decode(&p); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(fmt.Sprintf("Invalid STH Pollination received: %v", err)))
		return
	}

	err := h.storage.AddSTHPollination(p)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("Couldn't store pollination: %v", err)))
		return
	}

	rp, err := h.storage.GetRandomSTHPollination(*defaultNumPollinationsToReturn)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("Couldn't fetch pollination to return: %v", err)))
		return
	}

	json := json.NewEncoder(rw)
	if err := json.Encode(*rp); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("Couldn't get pollination to return: %v", err)))
		return
	}
}

// NewHandler creates a new Handler object, taking a pointer a Storage object to
// use for storing and retrieving feedback and pollination data.
func NewHandler(s *Storage) Handler {
	return Handler{storage: s}
}

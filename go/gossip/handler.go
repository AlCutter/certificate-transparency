package gossip

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/certificate-transparency/go/x509"
)

var defaultNumPollinationsToReturn = flag.Int("default_num_pollinations_to_return", 10,
	"Number of randomly selected STH pollination entries to return for sth-pollination requests.")

// TODO(alcutter): This probably needs to be a list at somepoint.
var myFQHostName = flag.String("public_hostname", "", "The fully qualified host.domain name that this server is reached at.")

// Handler for the gossip HTTP requests.
type Handler struct {
	storage *Storage
}

func certFromB64(b64 string) (*x509.Certificate, error) {
	chainB64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64))
	chain, err := ioutil.ReadAll(chainB64)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(chain)
}

func validateChain(chainB64 []string) (*x509.Certificate, error) {
	cert, err := certFromB64(chainB64[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert: %v", err)
	}
	pool := x509.NewCertPool()
	for _, i := range chainB64[1:] {
		iCert, err := certFromB64(i)
		if err != nil {
			return nil, fmt.Errorf("failed to parse intermediate cert: %v", err)
		}
		pool.AddCert(iCert)
	}
	opts := x509.VerifyOptions{Intermediates: pool}
	_, err = cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to verify cert: %v", err)
	}
	return cert, nil
}

func shouldAcceptCert(cert *x509.Certificate) bool {
	if err := cert.VerifyHostname(*myFQHostName); err != nil {
		log.Printf("Failed to verify hostname: %v", err)
		return false
	}
	return true
}

func validateSCT(sct string, cert *x509.Certificate) error {
	log.Printf("validateSCT unimplemented")
	return nil
}

// TODO(alcutter): 5.1.1 Validate leaf chains up to a trusted root
// TODO(alcutter): 5.1.1/2 Verify each SCT is valid and from a known log, discard those which aren't
// TODO(alcutter): 5.1.1/3 Discard leaves for domains other than ours.
func (h *Handler) verifyAndStripInvalidFeedback(f *SCTFeedback) {
	var filtered []SCTFeedbackEntry
	for _, entry := range f.Feedback {
		// Check the presented chain is valid and terminates in a trusted root
		cert, err := validateChain(entry.X509Chain)
		if err != nil {
			log.Printf("Failed to validate chain: %v", err)
			continue
		}

		// Check whether we should accept this cert based on its subject/SAN
		if !shouldAcceptCert(cert) {
			log.Printf("Rejecting cert for host: %s", cert.Subject.CommonName)
			continue
		}

		// Check each SCT and remove any which are invalid
		var filteredSCTs []string
		for _, sct := range entry.SCTData {
			if err := validateSCT(sct, cert); err != nil {
				log.Printf("Failed to validate SCT: %v", err)
				continue
			}
			filteredSCTs = append(filteredSCTs, sct)
		}
		entry.SCTData = filteredSCTs
		filtered = append(filtered, entry)
	}
	f.Feedback = filtered
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

	h.verifyAndStripInvalidFeedback(&feedback)

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

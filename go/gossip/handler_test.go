package gossip

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	addSCTFeedbackJSON = `
      {
        "sct_feedback": [
          { "x509_chain": [
            "CHAIN00",
            "CHAIN01"
            ],
            "sct_data": [
            "SCT00",
            "SCT01",
            "SCT02"
            ]
          }, {
            "x509_chain": [
            "CHAIN10",
            "CHAIN11"
            ],
            "sct_data": [
            "SCT10",
            "SCT11",
            "SCT12"
            ]
          }
        ]
      }`

	addSTHPollinationJSON = `
      {
        "sths": [
          {
            "sth_version": 0,
            "tree_size": 100,
            "timestamp": 1438254824,
            "sha256_root_hash": "HASH0",
            "tree_head_signature": "SIG0",
            "log_id": "LOG0"
          }, {
            "sth_version": 0,
            "tree_size": 100,
            "timestamp": 1438254825,
            "sha256_root_hash": "HASH1",
            "tree_head_signature": "SIG1",
            "log_id": "LOG0"
          }, {
            "sth_version": 0,
            "tree_size": 400,
            "timestamp": 1438254824,
            "sha256_root_hash": "HASH2",
            "tree_head_signature": "SIG2",
            "log_id": "LOG1"
          }
        ]
      }`
)

func CreateAndOpenStorage() *Storage {
	dir, err := ioutil.TempDir("", "handlertest")
	if err != nil {
		log.Fatalf("Failed to get temporary dir for test: %v", err)
	}
	*dbName = dir + "/gossip.db"
	s := &Storage{}
	if err := s.Open(); err != nil {
		log.Fatalf("Failed to Open() storage: %v", err)
	}
	return s
}

func CloseAndDeleteStorage(s *Storage, path string) {
	s.Close()
	if err := os.Remove(path); err != nil {
		log.Printf("Failed to remove test DB (%v): %v", path, err)
	}
}

func SCTFeedbackFromString(t *testing.T, s string) SCTFeedback {
	json := json.NewDecoder(strings.NewReader(s))
	var f SCTFeedback
	if err := json.Decode(&f); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	return f
}

func STHPollinationFromString(t *testing.T, s string) STHPollination {
	json := json.NewDecoder(strings.NewReader(s))
	var f STHPollination
	if err := json.Decode(&f); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	return f
}

func ExpectStorageHasFeedback(t *testing.T, s *Storage, chain []string, sct string) {
	sctID, err := s.getSCTID(sct)
	if err != nil {
		t.Fatalf("Failed to look up ID for SCT %v: %v", sct, err)
	}
	chainID, err := s.getChainID(chain)
	if err != nil {
		t.Fatalf("Failed to look up ID for Chain %v: %v", chain, err)
	}
	assert.True(t, s.hasFeedback(sctID, chainID))
}

func MustGet(t *testing.T, f func() (int64, error)) int64 {
	v, err := f()
	if err != nil {
		t.Fatalf("Got error while calling %v: %v", f, err)
	}
	return v
}

func TestHandlesValidSCTFeedback(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sct-feedback", strings.NewReader(addSCTFeedbackJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSCTFeedback(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	f := SCTFeedbackFromString(t, addSCTFeedbackJSON)
	for _, entry := range f.Feedback {
		for _, sct := range entry.SCTData {
			ExpectStorageHasFeedback(t, s, entry.X509Chain, sct)
		}
	}
}

func TestHandlesDuplicatedSCTFeedback(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sct-feedback", strings.NewReader(addSCTFeedbackJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	for i := 0; i < 10; i++ {
		h.HandleSCTFeedback(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	numExpectedChains := 0
	numExpectedSCTs := 0
	f := SCTFeedbackFromString(t, addSCTFeedbackJSON)
	for _, entry := range f.Feedback {
		numExpectedChains++
		for _, sct := range entry.SCTData {
			numExpectedSCTs++
			ExpectStorageHasFeedback(t, s, entry.X509Chain, sct)
		}
	}

	assert.EqualValues(t, numExpectedChains, MustGet(t, s.getNumChains))
	assert.EqualValues(t, numExpectedSCTs, MustGet(t, s.getNumSCTs))
	assert.EqualValues(t, numExpectedSCTs, MustGet(t, s.getNumFeedback)) // one feedback entry per SCT/Chain pair
}

func TestRejectsInvalidSCTFeedback(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sct-feedback", strings.NewReader("BlahBlah},"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSCTFeedback(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlesValidSTHPollination(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sth-pollination", strings.NewReader(addSTHPollinationJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSTHPollination(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	f := STHPollinationFromString(t, addSTHPollinationJSON)

	assert.EqualValues(t, len(f.STHs), MustGet(t, s.getNumSTHs))
	for _, sth := range f.STHs {
		assert.True(t, s.hasSTH(sth.STHVersion, sth.TreeSize, sth.Timestamp, sth.Sha256RootHashB64, sth.TreeHeadSignatureB64, sth.LogID))
	}
}

func TestHandlesDuplicateSTHPollination(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sth-pollination", strings.NewReader(addSTHPollinationJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	for i := 0; i < 10; i++ {
		h.HandleSTHPollination(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	f := STHPollinationFromString(t, addSTHPollinationJSON)

	assert.EqualValues(t, len(f.STHs), MustGet(t, s.getNumSTHs))
	for _, sth := range f.STHs {
		assert.True(t, s.hasSTH(sth.STHVersion, sth.TreeSize, sth.Timestamp, sth.Sha256RootHashB64, sth.TreeHeadSignatureB64, sth.LogID))
	}
}

func TestHandlesInvalidSTHPollination(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sth-pollination", strings.NewReader("blahblah,,}{"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSTHPollination(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReturnsSTHPollination(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sth-pollination", strings.NewReader(addSTHPollinationJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSTHPollination(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// since this is an empty DB, we should get back all of the pollination we sent
	// TODO(alcutter): We probably shouldn't blindly return stuff we were just given really, that's kinda silly, but it'll do for now.
	sentPollination := STHPollinationFromString(t, addSTHPollinationJSON)
	recvPollination := STHPollinationFromString(t, rr.Body.String())

	for _, sth := range sentPollination.STHs {
		assert.Contains(t, recvPollination.STHs, sth)
	}

	assert.Equal(t, len(sentPollination.STHs), len(recvPollination.STHs))
}

func TestLimitsSTHPollinationReturned(t *testing.T) {
	s := CreateAndOpenStorage()
	defer CloseAndDeleteStorage(s, *dbName)

	*defaultNumPollinationsToReturn = 1
	h := NewHandler(s)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/.well-known/ct/v1/sth-pollination", strings.NewReader(addSTHPollinationJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	h.HandleSTHPollination(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// since this is an empty DB, we should get back all of the pollination we sent
	// TODO(alcutter): We probably shouldn't blindly return stuff we were just given really, that's kinda silly, but it'll do for now.
	sentPollination := STHPollinationFromString(t, addSTHPollinationJSON)
	recvPollination := STHPollinationFromString(t, rr.Body.String())

	assert.Equal(t, 1, len(recvPollination.STHs))
	assert.Contains(t, sentPollination.STHs, recvPollination.STHs[0])
}

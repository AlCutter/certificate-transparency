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
            "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"
            ],
            "sct_data": [
            "SCT00",
            "SCT01",
            "SCT02"
            ]
          }, {
            "x509_chain": [
              "MIIFfzCCBGegAwIBAgIQX6A5SLE/idM1cLWXO4+OeDANBgkqhkiG9w0BAQUFADCBizELMAkGA1UEBhMCVVMxGzAZBgNVBAoTEk9yYWNsZSBDb3Jwb3JhdGlvbjEfMB0GA1UECxMWVmVyaVNpZ24gVHJ1c3QgTmV0d29yazEmMCQGA1UECxMdQ2xhc3MgMyBNUEtJIFNlY3VyZSBTZXJ2ZXIgQ0ExFjAUBgNVBAMTDU9yYWNsZSBTU0wgQ0EwHhcNMTIwNTA0MDAwMDAwWhcNMTQwNTA0MjM1OTU5WjB3MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEXMBUGA1UEBxQOUmVkd29vZCBTaG9yZXMxGzAZBgNVBAoUEk9yYWNsZSBDb3Jwb3JhdGlvbjEdMBsGA1UEAxQUd3d3LnN1bnNwb3R3b3JsZC5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDPliq6ecLgdtIn5gsseA3wL953WuB7FjDHNUOXnzkdT2ejJSbVTnH3qr5x17tX/ynYei/4eLn+Pydr9kWhU54pOZ+gMyQNhova33SMYlSgANoXSvoE85u6oDl/GNsFNrIROFncJ9YVDIWlwhu3kZcLMajOpUqYhFKUWNmZcOQmIZqZOOjQrZe34oIVCYHujrdjQSHKf/jek+eie0g9muI27Vo+apfi3DecF3elqRkh+SdWfRBMvV9mJMZBc9Mh1vT/uIYhkOn/d15zB7xjVAglTeea6NqaCi2o7kgglKb+7W0IkTLbTd1zHyXKZRZvEM/aV6oJgPdD489WKMqSid/fAgMBAAGjggHwMIIB7DAfBgNVHREEGDAWghR3d3cuc3Vuc3BvdHdvcmxkLmNvbTAJBgNVHRMEAjAAMB0GA1UdDgQWBBSMkjgHi5AvD/ru4fyo+MYGpDS99TAfBgNVHSMEGDAWgBTM+LtlR2pSFsTsfpsnnPwuqcLwDzAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMEQGA1UdIAQ9MDswOQYLYIZIAYb4RQEHFwMwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cudmVyaXNpZ24uY29tL3JwYTBhBgNVHR8EWjBYMFagVKBShlBodHRwOi8vU1ZSU2VjdXJlLW9yYWNsZS1jcmwudmVyaXNpZ24uY29tL09yYWNsZUNvcnBvcmF0aW9uVW5pZmllZC9TVlItT3JhY2xlLmNybDCBpQYIKwYBBQUHAQEEgZgwgZUwNQYIKwYBBQUHMAGGKWh0dHA6Ly9TVlJTZWN1cmUtb3JhY2xlLW9jc3AudmVyaXNpZ24uY29tMFwGCCsGAQUFBzAChlBodHRwOi8vU1ZSU2VjdXJlLW9yYWNsZS1haWEudmVyaXNpZ24uY29tL09yYWNsZUNvcnBvcmF0aW9uVW5pZmllZC9TVlItT3JhY2xlLmNlcjANBgkqhkiG9w0BAQUFAAOCAQEAd9OrHs2CoVSYakzB+3q11HPI3+ONQSF2/U3EUhiBxJk6WdYkeY9xxo0Hiy7JlnFGE8NpoQGKz0RBwdpXNn0CkWHm0anoTOWCOUAc+2M7AP8p+1Tv503zCJgZlGOrxfIYkg/WfgAvWCNAggY78JIaWIc0IsCp6NOcPINQ8q2Cw8RelSArWoCL+qBW4KhGL2gHlnYSIeXo0/QOge3DppIgQr9g5sgm4qvz9IUYk97KS8hWMv5B3KVaSp0zTmaqhYs1JHugFWoArXGlNxZ6QVzhQ18FMqriW6XJ934TDCtVLFutQCV511xwXJfdBT1r/Hzlk1iGF3VsvzxQ3Cyf5mKuqA==",
              "MIIF7DCCBNSgAwIBAgIQbsx6pacDIAm4zrz06VLUkTANBgkqhkiG9w0BAQUFADCByjELMAkGA1UEBhMCVVMxFzAVBgNVBAoTDlZlcmlTaWduLCBJbmMuMR8wHQYDVQQLExZWZXJpU2lnbiBUcnVzdCBOZXR3b3JrMTowOAYDVQQLEzEoYykgMjAwNiBWZXJpU2lnbiwgSW5jLiAtIEZvciBhdXRob3JpemVkIHVzZSBvbmx5MUUwQwYDVQQDEzxWZXJpU2lnbiBDbGFzcyAzIFB1YmxpYyBQcmltYXJ5IENlcnRpZmljYXRpb24gQXV0aG9yaXR5IC0gRzUwHhcNMTAwMjA4MDAwMDAwWhcNMjAwMjA3MjM1OTU5WjCBtTELMAkGA1UEBhMCVVMxFzAVBgNVBAoTDlZlcmlTaWduLCBJbmMuMR8wHQYDVQQLExZWZXJpU2lnbiBUcnVzdCBOZXR3b3JrMTswOQYDVQQLEzJUZXJtcyBvZiB1c2UgYXQgaHR0cHM6Ly93d3cudmVyaXNpZ24uY29tL3JwYSAoYykxMDEvMC0GA1UEAxMmVmVyaVNpZ24gQ2xhc3MgMyBTZWN1cmUgU2VydmVyIENBIC0gRzMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCxh4QfwgxF9byrJZenraI+nLr2wTm4i8rCrFbG5btljkRPTc5v7QlK1K9OEJxoiy6Ve4mbE8riNDTB81vzSXtig0iBdNGIeGwCU/m8f0MmV1gzgzszChew0E6RJK2GfWQS3HRKNKEdCuqWHQsV/KNLO85jiND4LQyUhhDKtpo9yus3nABINYYpUHjoRWPNGUFP9ZXse5jUxHGzUL4os4+guVOc9cosI6n9FAboGLSa6Dxugf3kzTU2s1HTaewSulZub5tXxYsU5w7HnO1KVGrJTcW/EbGuHGeBy0RVM5l/JJs/U0V/hhrzPPptf4H1uErT9YU3HLWm0AnkGHs4TvoPAgMBAAGjggHfMIIB2zA0BggrBgEFBQcBAQQoMCYwJAYIKwYBBQUHMAGGGGh0dHA6Ly9vY3NwLnZlcmlzaWduLmNvbTASBgNVHRMBAf8ECDAGAQH/AgEAMHAGA1UdIARpMGcwZQYLYIZIAYb4RQEHFwMwVjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cudmVyaXNpZ24uY29tL2NwczAqBggrBgEFBQcCAjAeGhxodHRwczovL3d3dy52ZXJpc2lnbi5jb20vcnBhMDQGA1UdHwQtMCswKaAnoCWGI2h0dHA6Ly9jcmwudmVyaXNpZ24uY29tL3BjYTMtZzUuY3JsMA4GA1UdDwEB/wQEAwIBBjBtBggrBgEFBQcBDARhMF+hXaBbMFkwVzBVFglpbWFnZS9naWYwITAfMAcGBSsOAwIaBBSP5dMahqyNjmvDz4Bq1EgYLHsZLjAlFiNodHRwOi8vbG9nby52ZXJpc2lnbi5jb20vdnNsb2dvLmdpZjAoBgNVHREEITAfpB0wGzEZMBcGA1UEAxMQVmVyaVNpZ25NUEtJLTItNjAdBgNVHQ4EFgQUDURcFlNEwYJ+HSCrJfQBY9i+eaUwHwYDVR0jBBgwFoAUf9Nlp8Ld7LvwMAnzQzn6Aq8zMTMwDQYJKoZIhvcNAQEFBQADggEBAAyDJO/dwwzZWJz+NrbrioBL0aP3nfPMU++CnqOh5pfBWJ11bOAdG0z60cEtBcDqbrIicFXZIDNAMwfCZYP6j0M3m+oOmmxw7vacgDvZN/R6bezQGH1JSsqZxxkoor7YdyT3hSaGbYcFQEFn0Sc67dxIHSLNCwuLvPSxe/20majpdirhGi2HbnTTiN0eIsbfFrYrghQKlFzyUOyvzv9iNw2tZdMGQVPtAhTItVgooazgW+yzf5VK+wPIrSbb5mZ4EkrZn0L74ZjmQoObj49nJOhhGbXdzbULJgWOw27EyHW4Rs/iGAZeqa6ogZpHFt4MKGwlJ7net4RYxh84HqTEy2Y=",
              "MIIE0DCCBDmgAwIBAgIQJQzo4DBhLp8rifcFTXz4/TANBgkqhkiG9w0BAQUFADBfMQswCQYDVQQGEwJVUzEXMBUGA1UEChMOVmVyaVNpZ24sIEluYy4xNzA1BgNVBAsTLkNsYXNzIDMgUHVibGljIFByaW1hcnkgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkwHhcNMDYxMTA4MDAwMDAwWhcNMjExMTA3MjM1OTU5WjCByjELMAkGA1UEBhMCVVMxFzAVBgNVBAoTDlZlcmlTaWduLCBJbmMuMR8wHQYDVQQLExZWZXJpU2lnbiBUcnVzdCBOZXR3b3JrMTowOAYDVQQLEzEoYykgMjAwNiBWZXJpU2lnbiwgSW5jLiAtIEZvciBhdXRob3JpemVkIHVzZSBvbmx5MUUwQwYDVQQDEzxWZXJpU2lnbiBDbGFzcyAzIFB1YmxpYyBQcmltYXJ5IENlcnRpZmljYXRpb24gQXV0aG9yaXR5IC0gRzUwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCvJAgIKXo1nmAMqudLO07cfLw8RRy7K+D+KQL5VwijZIUVJ/XxrcgxiV0i6CqqpkKzj/i5Vbext0uz/o9+B1fs70PbZmIVYc9gDaTY3vjgw2IIPVQT60nKWVSFJuUrjxuf6/WhkcIzSdhDY2pSS9KP6HBRTdGJaXvHcPaz3BJ023tdS1bTlr8Vd6Gw9KIl8q8ckmcY5fQGBO+QueQA5N06tRn/Arr0PO7gi+s3i+z016zy9vA9r911kTMZHRxAy3QkGSGT2RT+rCpSx4/VBEnkjWNHiDxpg8v+R70rfk/Fla4OndTRQ8Bnc+MUCH7lP59zuDMKz10/NIeWiu5T6CUVAgMBAAGjggGbMIIBlzAPBgNVHRMBAf8EBTADAQH/MDEGA1UdHwQqMCgwJqAkoCKGIGh0dHA6Ly9jcmwudmVyaXNpZ24uY29tL3BjYTMuY3JsMA4GA1UdDwEB/wQEAwIBBjA9BgNVHSAENjA0MDIGBFUdIAAwKjAoBggrBgEFBQcCARYcaHR0cHM6Ly93d3cudmVyaXNpZ24uY29tL2NwczAdBgNVHQ4EFgQUf9Nlp8Ld7LvwMAnzQzn6Aq8zMTMwbQYIKwYBBQUHAQwEYTBfoV2gWzBZMFcwVRYJaW1hZ2UvZ2lmMCEwHzAHBgUrDgMCGgQUj+XTGoasjY5rw8+AatRIGCx7GS4wJRYjaHR0cDovL2xvZ28udmVyaXNpZ24uY29tL3ZzbG9nby5naWYwNAYIKwYBBQUHAQEEKDAmMCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC52ZXJpc2lnbi5jb20wPgYDVR0lBDcwNQYIKwYBBQUHAwEGCCsGAQUFBwMCBggrBgEFBQcDAwYJYIZIAYb4QgQBBgpghkgBhvhFAQgBMA0GCSqGSIb3DQEBBQUAA4GBABMC3fjohgDyWvj4IAxZiGIHzs73Tvm7WaGY5eE43U68ZhjTresY8g3JbT5KlCDDPLq9ZVTGr0SzEK0saz6r1we2uIFjxfleLuUqZ87NMwwq14lWAyMfs77oOghZtOxFNfeKW/9mz1Cvxm1XjRl4t7mi0VfqH5pLr7rJjhJ+xr3/"
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

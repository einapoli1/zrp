package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// decodeEnvelope decodes an API response envelope and extracts the data
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode API envelope: %v", err)
	}
	dataBytes, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(dataBytes, v); err != nil {
		t.Fatalf("Failed to decode data from envelope: %v", err)
	}
}

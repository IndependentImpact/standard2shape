package tracer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHTTPGuidanceFlow(t *testing.T) {
	session, err := NewSession(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)
	server := httptest.NewServer(Handler(session, "missing-web-assets"))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/api/workspace")
	if err != nil {
		t.Fatal(err)
	}
	var initial Snapshot
	if err := json.NewDecoder(response.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d", response.StatusCode)
	}

	next := "State the project's purpose and intended outcome for an independent reviewer."
	payload, _ := json.Marshal(map[string]string{"guidance": next})
	response, err = http.Post(server.URL+"/api/guidance", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	var updated Snapshot
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST status = %d", response.StatusCode)
	}
	if updated.Document.Sections[0].Guidance != next {
		t.Fatalf("guidance = %q", updated.Document.Sections[0].Guidance)
	}
	if updated.Change == nil || !updated.Change.UnsupportedPreserved {
		t.Fatalf("change = %#v", updated.Change)
	}
	if initial.Unsupported[0].Digest != updated.Unsupported[0].Digest {
		t.Fatal("unsupported digest changed through HTTP flow")
	}
}

func TestHTTPRejectsEmptyGuidance(t *testing.T) {
	session, err := NewSession(filepath.Join("..", "..", "fixtures", "tracer"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)
	request := httptest.NewRequest(http.MethodPost, "/api/guidance", bytes.NewBufferString(`{"guidance":" "}`))
	recorder := httptest.NewRecorder()

	Handler(session, "missing-web-assets").ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

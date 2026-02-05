package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/challenge-github-events/internal/store"
	"go.uber.org/mock/gomock"
)

func TestServer_Health(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := store.NewMockStore(ctrl)
	mockStore.EXPECT().Ping(gomock.Any()).Return(nil)

	srv := NewServer(":0", mockStore)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status want 200 got %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("body.status want ok got %s", body["status"])
	}
}

func TestServer_Stats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := store.NewMockStore(ctrl)
	mockStore.EXPECT().GlobalNetLines(gomock.Any()).Return(int64(42), nil)
	mockStore.EXPECT().EventsSeenCount(gomock.Any()).Return(int64(10), nil)

	srv := NewServer(":0", mockStore)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()
	srv.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status want 200 got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if n, _ := body["global_net_lines_current"].(float64); n != 42 {
		t.Errorf("global_net_lines_current want 42 got %v", body["global_net_lines_current"])
	}
	if n, _ := body["events_seen_since_start"].(float64); n != 10 {
		t.Errorf("events_seen_since_start want 10 got %v", body["events_seen_since_start"])
	}
}

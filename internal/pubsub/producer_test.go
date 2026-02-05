package pubsub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/challenge-github-events/internal/github"
	"github.com/challenge-github-events/internal/store"
	"go.uber.org/mock/gomock"
)

func TestProducer_EnqueuesJobsForInsertedEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := store.NewMockStore(ctrl)
	payload := github.PushEventPayload{Commits: []github.PushCommit{{SHA: "sha1"}, {SHA: "sha2"}}}
	payloadJSON, _ := json.Marshal(payload)
	events := []github.Event{
		{
			ID:         "e1",
			Type:       "PushEvent",
			Repo:       &github.Repo{FullName: "owner/repo"},
			RawPayload: payloadJSON,
		},
		{ID: "e2", Type: "PushEvent", Repo: &github.Repo{FullName: "other/repo"}},
	}

	mockFetcher := github.NewMockEventsFetcher(ctrl)
	mockFetcher.EXPECT().FetchEvents(gomock.Any(), gomock.Any()).Return(events, "etag1", nil)
	mockStore.EXPECT().InsertPushEvent(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, event *store.PushEventRow) (bool, error) {
		return event.ID == "e1", nil
	}).Times(2)

	jobs := make(chan CommitJob, 4)
	prod := NewProducer(mockStore, mockFetcher, jobs, 10*time.Hour)
	go prod.Run(ctx)

	var got []CommitJob
	for i := 0; i < 2; i++ {
		select {
		case j := <-jobs:
			got = append(got, j)
		case <-time.After(2 * time.Second):
			break
		}
	}
	cancel()
	time.Sleep(100 * time.Millisecond)

	if len(got) != 2 {
		t.Fatalf("want 2 jobs got %d", len(got))
	}
	if got[0].SHA != "sha1" || got[1].SHA != "sha2" {
		t.Errorf("jobs want sha1, sha2 got %s, %s", got[0].SHA, got[1].SHA)
	}
	if got[0].EventID != "e1" || got[0].Owner != "owner" || got[0].Repo != "repo" {
		t.Errorf("job0 want event=e1 owner=owner repo=repo got event=%s owner=%s repo=%s", got[0].EventID, got[0].Owner, got[0].Repo)
	}
}

func TestProducer_EnqueuesTipCommitWhenCommitsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := store.NewMockStore(ctrl)
	// Public /events API omits "commits" but includes "head".
	payload := github.PushEventPayload{Head: "abc123tip"}
	payloadJSON, _ := json.Marshal(payload)
	events := []github.Event{
		{ID: "e1", Type: "PushEvent", Repo: &github.Repo{FullName: "owner/repo"}, RawPayload: payloadJSON},
	}

	mockFetcher := github.NewMockEventsFetcher(ctrl)
	mockFetcher.EXPECT().FetchEvents(gomock.Any(), gomock.Any()).Return(events, "", nil)
	mockStore.EXPECT().InsertPushEvent(gomock.Any(), gomock.Any()).Return(true, nil)

	jobs := make(chan CommitJob, 2)
	prod := NewProducer(mockStore, mockFetcher, jobs, 10*time.Hour)
	go prod.Run(ctx)

	var got CommitJob
	select {
	case got = <-jobs:
	case <-time.After(2 * time.Second):
		t.Fatal("expected one job for tip commit")
	}
	cancel()

	if got.SHA != "abc123tip" || got.Owner != "owner" || got.Repo != "repo" {
		t.Errorf("want SHA=abc123tip owner=owner repo=repo got SHA=%s owner=%s repo=%s", got.SHA, got.Owner, got.Repo)
	}
}

func TestProducer_DoesNotEnqueueWhenInsertReturnsFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockStore := store.NewMockStore(ctrl)
	payload := github.PushEventPayload{Commits: []github.PushCommit{{SHA: "sha1"}}}
	payloadJSON, _ := json.Marshal(payload)
	events := []github.Event{
		{ID: "e1", Type: "PushEvent", Repo: &github.Repo{FullName: "o/r"}, RawPayload: payloadJSON},
	}

	mockFetcher := github.NewMockEventsFetcher(ctrl)
	mockFetcher.EXPECT().FetchEvents(gomock.Any(), gomock.Any()).Return(events, "", nil)
	mockStore.EXPECT().InsertPushEvent(gomock.Any(), gomock.Any()).Return(false, nil)

	jobs := make(chan CommitJob, 1)
	prod := NewProducer(mockStore, mockFetcher, jobs, 10*time.Hour)
	go prod.Run(ctx)

	select {
	case <-jobs:
		t.Error("should not enqueue when InsertPushEvent returns false")
	case <-time.After(500 * time.Millisecond):
	}
	cancel()
}

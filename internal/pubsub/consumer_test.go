package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/challenge-github-events/internal/github"
	"github.com/challenge-github-events/internal/store"
	"go.uber.org/mock/gomock"
)

func TestConsumer_ProcessJob_InsertsStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := store.NewMockStore(ctrl)
	mockFetcher := github.NewMockCommitStatsFetcher(ctrl)
	ctx := context.Background()

	var capturedRow *store.CommitStatsRow
	mockFetcher.EXPECT().GetCommitStats(gomock.Any(), "o", "r", "sha1").Return(&github.CommitStats{
		SHA:         "sha1",
		Additions:   10,
		Deletions:   3,
		Total:       13,
		Net:         7,
		Author:      "author",
		CommittedAt: time.Now(),
	}, nil)
	mockStore.EXPECT().InsertCommitStats(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, row *store.CommitStatsRow) (bool, error) {
		capturedRow = row
		return true, nil
	})

	jobs := make(chan CommitJob, 1)
	cons := NewConsumer(mockStore, mockFetcher, jobs)
	jobs <- CommitJob{EventID: "e1", Owner: "o", Repo: "r", SHA: "sha1"}
	close(jobs)

	cons.Run(ctx)

	if capturedRow == nil {
		t.Fatal("InsertCommitStats was not called")
	}
	if capturedRow.Sha != "sha1" || capturedRow.Repo != "o/r" || capturedRow.Additions != 10 || capturedRow.Deletions != 3 || capturedRow.Net != 7 {
		t.Errorf("inserted want sha=sha1 repo=o/r add=10 del=3 net=7 got sha=%s repo=%s add=%d del=%d net=%d",
			capturedRow.Sha, capturedRow.Repo, capturedRow.Additions, capturedRow.Deletions, capturedRow.Net)
	}
}

func TestConsumer_ProcessJob_Skips404(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := store.NewMockStore(ctrl)
	mockFetcher := github.NewMockCommitStatsFetcher(ctrl)
	ctx := context.Background()

	mockFetcher.EXPECT().GetCommitStats(gomock.Any(), "o", "r", "sha").Return(nil, github.ErrNotFound)
	// InsertCommitStats must not be called

	jobs := make(chan CommitJob, 1)
	cons := NewConsumer(mockStore, mockFetcher, jobs)
	jobs <- CommitJob{Owner: "o", Repo: "r", SHA: "sha"}
	close(jobs)

	cons.Run(ctx)
}

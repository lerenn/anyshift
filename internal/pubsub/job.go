package pubsub

// CommitJob is a unit of work for the consumer: fetch stats for this commit and persist.
type CommitJob struct {
	EventID string
	Owner   string
	Repo    string
	SHA     string
}

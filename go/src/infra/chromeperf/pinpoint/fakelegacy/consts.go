package fakelegacy

// The Legacy Pinpoint service has a lot of constant strings in its API;
// this file contains named constants to avoid hard-coding them throughout,
// along with relevant types to help avoid mistakes.

type Status string

const (
	CompletedStatus Status = "COMPLETED"
)

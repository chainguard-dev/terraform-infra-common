package check

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v75/github"
)

// Docs for Check Run API: https://docs.github.com/en/rest/checks/runs?apiVersion=2022-11-28

const (
	maxCheckOutputLength = 65536
	truncationMessage    = "\n\n⚠️ _Summary has been truncated_"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusWaiting    Status = "waiting"
	StatusRequested  Status = "requested"
	StatusPending    Status = "pending"
)

type Conclusion string

const (
	ConclusionActionRequired Conclusion = "action_required"
	ConclusionCancelled      Conclusion = "cancelled"
	ConclusionFailure        Conclusion = "failure"
	// ConclusionNeutral is the default, and is sufficient to pass a required check.
	ConclusionNeutral  Conclusion = "neutral"
	ConclusionSuccess  Conclusion = "success"
	ConclusionTimedOut Conclusion = "timed_out"
	// ConclusionSkipped is not sufficient to pass a required check.
	ConclusionSkipped Conclusion = "skipped"
)

type Builder struct {
	md            strings.Builder
	name, headSHA string
	Summary       string
	Status        Status
	Conclusion    Conclusion
}

func NewBuilder(name, headSHA string) *Builder {
	return &Builder{
		name:    name,
		headSHA: headSHA,
	}
}

// Writef appends a formatted string to the CheckRun output.
//
// If the output exceeds the maximum length, it will be truncated and a message will be appended.
func (b *Builder) Writef(format string, args ...any) {
	if b.md.Len() <= maxCheckOutputLength {
		b.md.WriteString(fmt.Sprintf(format, args...))
		b.md.WriteRune('\n')
	}

	if b.md.Len() > maxCheckOutputLength {
		out := b.md.String()
		out = out[:maxCheckOutputLength-len(truncationMessage)]
		out += truncationMessage
		b.md = strings.Builder{}
		b.md.WriteString(out)
	}
}

// CheckRun returns a GitHub CheckRun object with the current state of the Builder.
//
// If the Summary field is empty, it will be set to the name field.
// If the Conclusion field is set, the CheckRun will be marked as completed.
func (b *Builder) CheckRunCreate() *github.CreateCheckRunOptions {
	if b.Summary == "" {
		b.Summary = b.name
	}
	cr := &github.CreateCheckRunOptions{
		Name:    b.name,
		HeadSHA: b.headSHA,
		Status:  github.Ptr(string(StatusInProgress)),
		Output: &github.CheckRunOutput{
			Title:   &b.Summary,
			Summary: &b.Summary,
			Text:    github.Ptr(b.md.String()),
		},
		// Fields we don't set:
		// - DetailsURL: sets the URL of the "Details" link at the bottom of the Check Run page. Defaults to the app's installation URL.
		// - ExternalID: sets a unique identifier of the check run on the external system. Not used by this SDK.
		// - Actions: sets actions that a user can perform on the check run. Not used by this SDK.
		// - StartedAt: sets the time that the check run began. Automatically set by GitHub the first time the check run is created if it's in-progress.
		// - CompletedAt: sets the time that the check run completed. Automatically set by GitHub the first time the check run is completed.
		// - Output.Annotations: sets annotations that are used to provide more information about a line of code. Not used by this SDK.
	}
	// Providing conclusion will automatically set the status parameter to completed.
	if b.Conclusion != "" {
		cr.Conclusion = github.Ptr(string(b.Conclusion))
		cr.Status = github.Ptr(string(StatusCompleted))
	}
	return cr
}

func (b *Builder) CheckRunUpdate() *github.UpdateCheckRunOptions {
	create := b.CheckRunCreate()
	return &github.UpdateCheckRunOptions{
		Name:       create.Name,
		Status:     create.Status,
		Conclusion: create.Conclusion,
		Output: &github.CheckRunOutput{
			Title:   create.GetOutput().Title,
			Summary: create.GetOutput().Summary,
			Text:    create.GetOutput().Text,
		},
	}
}

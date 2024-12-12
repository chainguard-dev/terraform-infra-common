package check

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v61/github"
)

func TestCheckRun(t *testing.T) {
	b := NewBuilder("name", "headSHA")
	b.Writef("test %d", 123)

	if diff := cmp.Diff(b.CheckRun(), &github.CheckRun{
		Name:    github.String("name"),
		HeadSHA: github.String("headSHA"),
		Output: &github.CheckRunOutput{
			Title:   github.String("name"),
			Summary: github.String("name"),
			Text:    github.String("test 123\n"),
		},
	}); diff != "" {
		t.Errorf("CheckRun() mismatch (-want +got):\n%s", diff)
	}

	b.Summary = "summary"
	b.Conclusion = ConclusionSuccess
	b.Writef("test %t", true)
	if diff := cmp.Diff(b.CheckRun(), &github.CheckRun{
		Name:       github.String("name"),
		HeadSHA:    github.String("headSHA"),
		Status:     github.String("completed"),
		Conclusion: github.String("success"),
		Output: &github.CheckRunOutput{
			Title:   github.String("summary"),
			Summary: github.String("summary"),
			Text:    github.String("test 123\ntest true\n"),
		},
	}); diff != "" {
		t.Errorf("CheckRun() mismatch (-want +got):\n%s", diff)
	}
}

func TestWritef(t *testing.T) {
	b := NewBuilder("name", "headSHA")

	// append 1 KB 100 times
	for i := 0; i < 100; i++ {
		b.Writef(strings.Repeat("a", 1024)) //nolint:govet

		// The output should never exceed maxCheckOutputLength, even internally.
		if b.md.Len() > maxCheckOutputLength {
			t.Fatalf("CheckRun().Output.Text length = %d, want <= %d", b.md.Len(), maxCheckOutputLength)
		}
	}

	gotText := b.CheckRun().GetOutput().GetText()
	wantLength := maxCheckOutputLength
	if len(gotText) != wantLength {
		t.Fatalf("CheckRun().Output.Text length = %d, want %d", len(gotText), wantLength)
	}
	if !strings.HasSuffix(gotText, truncationMessage) {
		last100 := gotText[len(gotText)-100:]
		t.Errorf("CheckRun().Output.Text does not have truncation message, ends with %q", last100)
	}
}

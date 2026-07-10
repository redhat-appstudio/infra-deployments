package changelog_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

// fakeComparer implements RepoComparer for testing.
// failTimes controls how many calls return err before the call succeeds.
type fakeComparer struct {
	files     []*gh.CommitFile
	commits   []*gh.RepositoryCommit // returned in CommitsComparison.Commits
	aheadBy   int                    // returned in CommitsComparison.AheadBy
	err       error
	failTimes int // number of initial calls that return err before succeeding
	calls     int // tracks total calls made
}

func (f *fakeComparer) CompareCommits(_ context.Context, _, _, _, _ string, _ *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error) {
	f.calls++
	if f.failTimes > 0 && f.calls <= f.failTimes {
		return nil, nil, f.err
	}
	return &gh.CommitsComparison{
		Files:   f.files,
		Commits: f.commits,
		AheadBy: gh.Ptr(f.aheadBy),
	}, nil, nil
}

func TestFetchOperatorCompare_ReturnsConvertedFiles(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr("-old\n+new"),
			},
			{Filename: gh.Ptr("other/file.yaml"), Patch: gh.Ptr("unchanged")},
		},
	}
	result, err := changelog.FetchOperatorCompare(context.Background(), fake, "oldref", "newref")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Files).To(HaveLen(2))
	g.Expect(result.Truncated).To(BeFalse())
	g.Expect(result.Files[0].Filename).To(Equal("operator/upstream-kustomizations/build-service/kustomization.yaml"))
	g.Expect(result.Files[0].Patch).To(Equal("-old\n+new"))
}

func TestFetchOperatorCompare_APIError(t *testing.T) {
	g := NewWithT(t)
	// Use 40-char SHAs so the error message exercises refShort's truncation path.
	oldSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newSHA := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	// failTimes=3 means all 3 attempts fail — the error should be returned.
	fake := &fakeComparer{err: errors.New("rate limited"), failTimes: 3}
	_, err := changelog.FetchOperatorCompare(context.Background(), fake, oldSHA, newSHA)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("konflux-ci/konflux-ci"))
	g.Expect(err.Error()).To(ContainSubstring("rate limited"))
	// refShort truncates to 12 chars — verify the error contains the truncated SHA.
	g.Expect(err.Error()).To(ContainSubstring("aaaaaaaaaaaa"))
	g.Expect(fake.calls).To(Equal(3))
}

func TestFetchOperatorCompare_RetriesOnTransientError(t *testing.T) {
	g := NewWithT(t)
	// Fail the first attempt, succeed on the second.
	fake := &fakeComparer{
		err:       errors.New("transient"),
		failTimes: 1,
		files: []*gh.CommitFile{
			{Filename: gh.Ptr("file.yaml"), Patch: gh.Ptr("diff")},
		},
	}
	result, err := changelog.FetchOperatorCompare(context.Background(), fake, "a", "b")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Files).To(HaveLen(1))
	g.Expect(fake.calls).To(Equal(2))
}

func TestFetchOperatorCompare_TruncatedWhenMaxFiles(t *testing.T) {
	g := NewWithT(t)
	// Build exactly 300 files — the API maximum.
	files := make([]*gh.CommitFile, 300)
	for i := range files {
		files[i] = &gh.CommitFile{Filename: gh.Ptr("file.yaml"), Patch: gh.Ptr("diff")}
	}
	fake := &fakeComparer{files: files}
	result, err := changelog.FetchOperatorCompare(context.Background(), fake, "a", "b")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Truncated).To(BeTrue())
}

func TestFetchOperatorCompare_NotTruncatedBeforeMax(t *testing.T) {
	g := NewWithT(t)
	files := make([]*gh.CommitFile, 299)
	for i := range files {
		files[i] = &gh.CommitFile{Filename: gh.Ptr("file.yaml"), Patch: gh.Ptr("diff")}
	}
	fake := &fakeComparer{files: files}
	result, err := changelog.FetchOperatorCompare(context.Background(), fake, "a", "b")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Truncated).To(BeFalse())
}

func TestFetchOperatorCompare_NilFilesIgnored(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeComparer{
		files: []*gh.CommitFile{nil, {Filename: gh.Ptr("file.yaml"), Patch: gh.Ptr("diff")}},
	}
	result, err := changelog.FetchOperatorCompare(context.Background(), fake, "a", "b")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Files).To(HaveLen(1))
}

func TestFetchOperatorCompare_NonRetryableErrorFastFails(t *testing.T) {
	g := NewWithT(t)
	// A 404 is a permanent error — retrying won't help and burns time.
	notFound := &gh.ErrorResponse{
		Response: &http.Response{StatusCode: http.StatusNotFound},
	}
	fake := &fakeComparer{err: notFound, failTimes: 3}
	_, err := changelog.FetchOperatorCompare(context.Background(), fake, "a", "b")
	g.Expect(err).To(HaveOccurred())
	g.Expect(fake.calls).To(Equal(1)) // only one attempt — no retries for 404
}

func TestFetchOperatorCompare_ContextCancelledDuringRetry(t *testing.T) {	g := NewWithT(t)
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the retry wait is interrupted.
	cancel()
	fake := &fakeComparer{err: errors.New("transient"), failTimes: 3}
	_, err := changelog.FetchOperatorCompare(ctx, fake, "a", "b")
	g.Expect(err).To(HaveOccurred())
	// Either context error or the API error — both are acceptable.
	isExpected := strings.Contains(err.Error(), "context") ||
		strings.Contains(err.Error(), "transient") ||
		strings.Contains(err.Error(), "konflux-ci")
	g.Expect(isExpected).To(BeTrue())
}

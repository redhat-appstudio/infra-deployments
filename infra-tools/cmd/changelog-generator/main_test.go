package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
	. "github.com/onsi/gomega"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/changelog"
)

const (
	oldRef = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newRef = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// fakeCommenter records the most recent UpsertCommentByMarker call.
type fakeCommenter struct {
	prNumber int
	body     string
	marker   string
}

func (f *fakeCommenter) UpsertCommentByMarker(_ context.Context, prNumber int, body, marker string) error {
	f.prNumber = prNumber
	f.body = body
	f.marker = marker
	return nil
}

// fakeRepoComparer implements changelog.RepoComparer for testing.
// The operator compare call (repo == "konflux-ci") uses err and files.
// Service compare calls (any other repo) use commitErr and commits/aheadBy.
var _ changelog.RepoComparer = &fakeRepoComparer{}

type fakeRepoComparer struct {
	files     []*gh.CommitFile
	commits   []*gh.RepositoryCommit
	aheadBy   int
	err       error // returned for the operator compare call
	commitErr error // returned for service compare calls
}

func (f *fakeRepoComparer) CompareCommits(_ context.Context, _, repo, _, _ string, _ *gh.ListOptions) (*gh.CommitsComparison, *gh.Response, error) {
	if repo == "konflux-ci" {
		if f.err != nil {
			return nil, nil, f.err
		}
		return &gh.CommitsComparison{Files: f.files}, nil, nil
	}
	// Service compare call
	if f.commitErr != nil {
		return nil, nil, f.commitErr
	}
	return &gh.CommitsComparison{
		Commits: f.commits,
		AheadBy: gh.Ptr(f.aheadBy),
	}, nil, nil
}

func writeTempKustomization(t *testing.T, ref string) string {
	t.Helper()
	content := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n" +
		"  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=" + ref + "\n"
	f, err := os.CreateTemp(t.TempDir(), "kustomization-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// TestPost_PassesCorrectMarkerAndBody verifies that post wires the permanent
// commentMarker through to UpsertCommentByMarker, not some other value.
func TestPost_PassesCorrectMarkerAndBody(t *testing.T) {
	g := NewWithT(t)
	fake := &fakeCommenter{}
	err := post(context.Background(), fake, 42, "hello")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.prNumber).To(Equal(42))
	g.Expect(fake.body).To(Equal("hello"))
	g.Expect(fake.marker).To(Equal(commentMarker))
}

// TestFormatCompare_ShortRef verifies refs under 12 chars (e.g. branch names)
// are not truncated — the short() helper takes a different branch.
func TestFormatCompare_ShortRef(t *testing.T) {
	g := NewWithT(t)
	body := formatCompare("main", "feature-x", nil)
	g.Expect(body).To(ContainSubstring("main"))
	g.Expect(body).To(ContainSubstring("feature-x"))
}

// TestComputeBody_Unchanged verifies that identical kustomization files produce
// the no-change comment body with the required marker and heading.
func TestComputeBody_Unchanged(t *testing.T) {
	g := NewWithT(t)
	path := writeTempKustomization(t, oldRef)
	body, err := computeBody(context.Background(), path, path, &fakeRepoComparer{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(commentMarker))
	g.Expect(body).To(ContainSubstring("### Operator Changelog"))
	g.Expect(body).To(ContainSubstring("No operator ref change"))
}

// TestComputeBody_Changed verifies that different kustomization files produce
// a compare comment containing both refs and a compare URL. No service bumps
// in this case because the fake comparer returns no files.
func TestComputeBody_Changed(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, &fakeRepoComparer{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(commentMarker))
	g.Expect(body).To(ContainSubstring(oldRef))
	g.Expect(body).To(ContainSubstring(newRef))
	g.Expect(body).To(ContainSubstring(oldRef[:12]))
	g.Expect(body).To(ContainSubstring("konflux-ci/konflux-ci/compare/" + oldRef + "..." + newRef))
	g.Expect(body).To(ContainSubstring("No upstream service refs changed"))
	g.Expect(body).NotTo(ContainSubstring("No operator ref change"))
}

// TestComputeBody_WithServiceBumps verifies that a fake comparer returning a
// build-service ref change causes "build-service" and its compare link to
// appear in the comment body.
func TestComputeBody_WithServiceBumps(t *testing.T) {
	g := NewWithT(t)
	buildOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	buildNewSHA := "dddddddddddddddddddddddddddddddddddddddd"

	patch := "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildNewSHA + "\n"

	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(patch),
			},
		},
	}

	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("build-service"))
	g.Expect(body).To(ContainSubstring(buildOldSHA[:12]))
	g.Expect(body).To(ContainSubstring(buildNewSHA[:12]))
	g.Expect(body).To(ContainSubstring("build-service/compare/" + buildOldSHA + "..." + buildNewSHA))
}

// TestComputeBody_APIFailureDegrades verifies that when the comparer returns an
// error, the comment still includes the operator compare link (not an error page),
// and notes that service bump detection was unavailable.
func TestComputeBody_APIFailureDegrades(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	fake := &fakeRepoComparer{err: errors.New("rate limited")}
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("compare/" + oldRef + "..." + newRef))
	g.Expect(body).To(ContainSubstring("unavailable"))
}

// TestComputeBody_TruncatedDegrades verifies that when the compare API signals
// truncation (≥300 files), the comment still includes the operator compare link
// but notes that service bump detection was unavailable.
func TestComputeBody_TruncatedDegrades(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)

	// Return exactly 300 files to trigger the truncated flag.
	files := make([]*gh.CommitFile, 300)
	for i := range files {
		files[i] = &gh.CommitFile{Filename: gh.Ptr("file.yaml"), Patch: gh.Ptr("diff")}
	}
	fake := &fakeRepoComparer{files: files}

	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("compare/" + oldRef + "..." + newRef))
	g.Expect(body).To(ContainSubstring("unavailable"))
}

// TestComputeBody_EmptyPatchDegrades verifies that when an upstream kustomization
// file is returned by the compare API but has no patch data, the comment still
// includes the operator compare link but notes that service bump detection was
// unavailable — preventing a misleading "no upstream service refs changed" message.
func TestComputeBody_EmptyPatchDegrades(t *testing.T) {
	g := NewWithT(t)
	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)

	// The file is an upstream kustomization but its patch is empty — GitHub omits
	// patch data for very large or renamed files.
	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(""),
			},
		},
	}

	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("compare/" + oldRef + "..." + newRef))
	g.Expect(body).To(ContainSubstring("unavailable"))
	g.Expect(body).NotTo(ContainSubstring("No upstream service refs changed"))
}

// TestComputeBody_Error verifies that an unreadable file returns an error.
func TestComputeBody_Error(t *testing.T) {
	g := NewWithT(t)
	_, err := computeBody(context.Background(), "/nonexistent/kustomization.yaml", "/nonexistent/kustomization.yaml", &fakeRepoComparer{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("extracting operator refs"))
}

// TestComputeBody_WithServiceCommits verifies that feat/fix conventional commits
// fetched for a bumped service appear in the comment body, while chore/docs are
// excluded.
func TestComputeBody_WithServiceCommits(t *testing.T) {
	g := NewWithT(t)
	buildOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	buildNewSHA := "dddddddddddddddddddddddddddddddddddddddd"

	patch := "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildNewSHA + "\n"

	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(patch),
			},
		},
		commits: []*gh.RepositoryCommit{
			{SHA: gh.Ptr("e1e1e1e1e1e1"), Commit: &gh.Commit{Message: gh.Ptr("feat: add multi-arch support")}},
			{SHA: gh.Ptr("f2f2f2f2f2f2"), Commit: &gh.Commit{Message: gh.Ptr("fix(pipeline): correct timeout handling")}},
			{SHA: gh.Ptr("a3a3a3a3a3a3"), Commit: &gh.Commit{Message: gh.Ptr("chore: update deps")}},
		},
	}

	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("build-service"))
	g.Expect(body).To(ContainSubstring("feat: add multi-arch support"))
	g.Expect(body).To(ContainSubstring("fix(pipeline): correct timeout handling"))
	g.Expect(body).NotTo(ContainSubstring("chore:"))
}

// TestComputeBody_ServiceCommitsFail verifies that when the commit fetch fails
// for a service, the bump header is still shown with a "unavailable" note,
// while the operator-level compare link is unaffected.
func TestComputeBody_ServiceCommitsFail(t *testing.T) {
	g := NewWithT(t)
	buildOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	buildNewSHA := "dddddddddddddddddddddddddddddddddddddddd"

	patch := "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildNewSHA + "\n"

	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(patch),
			},
		},
		commitErr: errors.New("rate limited"),
	}

	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("build-service"))
	g.Expect(body).To(ContainSubstring("compare/" + oldRef + "..." + newRef))
	g.Expect(body).To(ContainSubstring("Commit details unavailable"))
	g.Expect(body).NotTo(ContainSubstring("check manually")) // operator-level degradation message absent
}

// TestComputeBody_ServiceCommitsTruncated verifies that when AheadBy hits the
// 250-commit API limit, a truncation note appears after the commit list.
func TestComputeBody_ServiceCommitsTruncated(t *testing.T) {
	g := NewWithT(t)
	buildOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	buildNewSHA := "dddddddddddddddddddddddddddddddddddddddd"

	patch := "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildNewSHA + "\n"

	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(patch),
			},
		},
		commits: []*gh.RepositoryCommit{
			{SHA: gh.Ptr("e1e1e1e1e1e1"), Commit: &gh.Commit{Message: gh.Ptr("feat: add feature")}},
		},
		aheadBy: 250,
	}

	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("feat: add feature"))
	g.Expect(body).To(ContainSubstring("250 commits"))
}

// TestComputeBody_NoNotableCommits verifies that when service commits exist but
// none are feat/fix conventional commits, a clear "no notable commits" note is shown.
func TestComputeBody_NoNotableCommits(t *testing.T) {
	g := NewWithT(t)
	buildOldSHA := "cccccccccccccccccccccccccccccccccccccccc"
	buildNewSHA := "dddddddddddddddddddddddddddddddddddddddd"

	patch := "-  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildOldSHA + "\n" +
		"+  - https://github.com/konflux-ci/build-service/config/default?ref=" + buildNewSHA + "\n"

	fake := &fakeRepoComparer{
		files: []*gh.CommitFile{
			{
				Filename: gh.Ptr("operator/upstream-kustomizations/build-service/kustomization.yaml"),
				Patch:    gh.Ptr(patch),
			},
		},
		commits: []*gh.RepositoryCommit{
			{SHA: gh.Ptr("a1a1a1"), Commit: &gh.Commit{Message: gh.Ptr("chore: bump go version")}},
			{SHA: gh.Ptr("b2b2b2"), Commit: &gh.Commit{Message: gh.Ptr("Merge pull request #42")}},
		},
	}

	basePath := writeTempKustomization(t, oldRef)
	headPath := writeTempKustomization(t, newRef)
	body, err := computeBody(context.Background(), basePath, headPath, fake)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring("build-service"))
	g.Expect(body).To(ContainSubstring("No notable commits"))
}

// TestBuildBody_Integration creates a real git repo with two commits that have
// different operator refs, then calls buildBody to validate the full git path:
// worktree creation, kustomization parsing, and comment body selection.
func TestBuildBody_Integration(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test")

	// Commit 1: kustomization with oldRef (this becomes the base).
	kustPath := filepath.Join(dir, kustomizationPath)
	g.Expect(os.MkdirAll(filepath.Dir(kustPath), 0755)).To(Succeed())
	writeKustomizationFile(t, kustPath, oldRef)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial")
	baseCommit := strings.TrimSpace(gitOutput(t, dir, "rev-parse", "HEAD"))

	// Commit 2: kustomization with newRef (this becomes HEAD).
	writeKustomizationFile(t, kustPath, newRef)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "bump operator ref")

	// Use a fake comparer so the test does not make real GitHub API calls.
	body, err := buildBody(context.Background(), dir, baseCommit, &fakeRepoComparer{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(body).To(ContainSubstring(oldRef))
	g.Expect(body).To(ContainSubstring(newRef))
	g.Expect(body).To(ContainSubstring("compare"))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

func writeKustomizationFile(t *testing.T, path, ref string) {
	t.Helper()
	content := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n" +
		"  - https://github.com/konflux-ci/konflux-ci/operator/config/default?ref=" + ref + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing kustomization: %v", err)
	}
}

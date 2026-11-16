package kustomize //nolint:testpackage // tests unexported retry and stderr helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestFilterBuildStderr(t *testing.T) {
	g := NewWithT(t)

	raw := `
Warning: 'patchesStrategicMerge' is deprecated. Please use 'patches' instead.
# Warning: 'commonLabels' is deprecated. Please use 'labels' instead.

Error: failed to download chart "foo"
fatal: unable to access remote
::error:: injected workflow command ##[warning]also injected
`
	got := filterBuildStderr(raw)
	g.Expect(got).To(ContainSubstring("failed to download chart"))
	g.Expect(got).To(ContainSubstring("unable to access remote"))
	g.Expect(got).To(ContainSubstring(": :error: : injected workflow command # #[warning]also injected"))
	g.Expect(got).NotTo(ContainSubstring("deprecated"))
	g.Expect(got).NotTo(HavePrefix("\n"))
}

func TestFilterBuildStderrTruncatesTail(t *testing.T) {
	g := NewWithT(t)

	raw := strings.Repeat("x", buildStderrLimit+100) + "\nFINAL ERROR"
	got := filterBuildStderr(raw)
	g.Expect(got).To(HavePrefix("[... truncated ...]"))
	g.Expect(got).To(ContainSubstring("FINAL ERROR"))
	g.Expect(len(got)).To(BeNumerically("<=", buildStderrLimit+len("[... truncated ...]\n")))
}

func TestIsTransientBuildError(t *testing.T) {
	g := NewWithT(t)

	for _, msg := range []string{
		"i/o timeout",
		"context deadline exceeded",
		"connection reset by peer",
		"temporary failure in name resolution",
		"unexpected EOF",
		"HTTP 503 Service Unavailable",
		"HTTP 500 Internal Server Error",
		"status code 500",
		"HTTP/1.1 503 Service Unavailable",
		"dial tcp: connection timed out",
		"network is unreachable",
		"rate limited",
	} {
		g.Expect(isTransientBuildError(errors.New(msg))).To(BeTrue(), msg)
	}
	for _, msg := range []string{
		"unable to find one of 'kustomization.yaml'",
		"invalid YAML",
		"401 Unauthorized",
		"invalid field timeoutSeconds",
		"error at line 504",
	} {
		g.Expect(isTransientBuildError(errors.New(msg))).To(BeFalse(), msg)
	}
	g.Expect(isTransientBuildError(nil)).To(BeFalse())
}

func TestBuildWithRetry_SucceedsOnRetry(t *testing.T) {
	g := NewWithT(t)

	first := func(string) ([]byte, error) {
		return nil, errors.New("temporary failure in name resolution")
	}
	retry := func(string) ([]byte, string, error) {
		return []byte("rendered"), "", nil
	}

	oldDelay := buildRetryDelay
	buildRetryDelay = 0
	defer func() { buildRetryDelay = oldDelay }()

	out, err := buildWithRetry("dir", first, retry)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(out)).To(Equal("rendered"))
}

func TestBuildWithRetry_ReportsCapturedStderr(t *testing.T) {
	g := NewWithT(t)

	first := func(string) ([]byte, error) {
		return nil, errors.New("kustomize build dir: failed to download chart")
	}
	retry := func(string) ([]byte, string, error) {
		return nil, "Error: failed to download chart\nWarning: old field is deprecated\n", errors.New("kustomize build dir: failed to download chart again")
	}

	out, err := buildWithRetry("dir", first, retry)
	g.Expect(out).To(BeNil())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("kustomize build dir"))
	g.Expect(err.Error()).To(ContainSubstring("build stderr:"))
	g.Expect(err.Error()).To(ContainSubstring("failed to download chart"))
	g.Expect(err.Error()).NotTo(ContainSubstring("deprecated"))
}

func TestBuildWithRetry_DoesNotRetrySuccess(t *testing.T) {
	g := NewWithT(t)

	retryCalled := false
	first := func(string) ([]byte, error) {
		return []byte("rendered"), nil
	}
	retry := func(string) ([]byte, string, error) {
		retryCalled = true
		return nil, "", errors.New("should not be called")
	}

	out, err := buildWithRetry("dir", first, retry)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(out)).To(Equal("rendered"))
	g.Expect(retryCalled).To(BeFalse())
}

func TestBuildWithRetry_IncludesRetryDetail(t *testing.T) {
	g := NewWithT(t)

	first := func(string) ([]byte, error) {
		return nil, errors.New("failed to download chart A")
	}
	retry := func(string) ([]byte, string, error) {
		return nil, "", errors.New("helm pull: HTTP 500 Internal Server Error: token=secret-token")
	}

	_, err := buildWithRetry("dir", first, retry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to download chart A"))
	g.Expect(err.Error()).To(ContainSubstring("HTTP 500"))
	g.Expect(err.Error()).NotTo(ContainSubstring("secret-token"))
}

func TestBuildWithRetry_DownloadErrorRetries(t *testing.T) {
	g := NewWithT(t)

	called := false
	first := func(string) ([]byte, error) {
		return nil, errors.New("unable to run: helm pull failed")
	}
	retry := func(string) ([]byte, string, error) {
		called = true
		return []byte("rendered"), "", nil
	}

	out, err := buildWithRetry("dir", first, retry)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(out)).To(Equal("rendered"))
	g.Expect(called).To(BeTrue())
}

func TestBuildWithRetry_DeterministicKustomizeError(t *testing.T) {
	g := NewWithT(t)

	called := false
	first := func(string) ([]byte, error) {
		return nil, errors.New("accumulating resources: recursed accumulation: invalid YAML")
	}
	retry := func(string) ([]byte, string, error) {
		called = true
		return nil, "", errors.New("should not be called")
	}

	_, err := buildWithRetry("dir", first, retry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid YAML"))
	g.Expect(called).To(BeFalse())
}

func TestSecretEnvName(t *testing.T) {
	g := NewWithT(t)

	for _, key := range []string{"GITHUB_TOKEN", "MY_API_KEY", "CLIENT_SECRET", "DB_PASSWORD", "REGISTRY_CREDENTIALS", "GITLAB_PAT"} {
		g.Expect(secretEnvName(key)).To(BeTrue(), key)
	}
	for _, key := range []string{"PATH", "GOPATH", "GITHUB_PATH", "SSH_AUTH_SOCK", "XAUTHORITY"} {
		g.Expect(secretEnvName(key)).To(BeFalse(), key)
	}
}

func TestBuildWithRetry_DoesNotRetryDeterministic(t *testing.T) {
	g := NewWithT(t)

	retryCalled := false
	first := func(string) ([]byte, error) {
		return nil, errors.New("invalid YAML: bad indentation")
	}
	retry := func(string) ([]byte, string, error) {
		retryCalled = true
		return nil, "", errors.New("should not be called")
	}

	_, err := buildWithRetry("dir", first, retry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid YAML"))
	g.Expect(retryCalled).To(BeFalse())
}

func TestBuildWithRetry_KeepsOriginalError(t *testing.T) {
	g := NewWithT(t)

	first := func(string) ([]byte, error) {
		return nil, errors.New("failed to download chart A")
	}
	retry := func(string) ([]byte, string, error) {
		return nil, "", errors.New("creating stderr capture: no space")
	}

	_, err := buildWithRetry("dir", first, retry)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to download chart A"))
}

func TestBuildRetryDelayIsSmall(t *testing.T) {
	g := NewWithT(t)
	// Sanity check that the default retry delay is bounded; a diagnostic retry
	// must not add minutes to a failing job.
	g.Expect(buildRetryDelay).To(BeNumerically("<=", 5*time.Second))
}

func TestCaptureTailKeepsLastBytes(t *testing.T) {
	g := NewWithT(t)

	got, err := captureTail(strings.NewReader("0123456789ABCDEF"), 10)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("6789ABCDEF"))
}

func TestRedactSecrets(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("TEST_API_TOKEN", "supersecretvalue")

	raw := "failed https://user:pass@example.com/chart?token=abc123\n" +
		"Authorization: Bearer abc.def.ghi\n" +
		"token=supersecretvalue"
	got := filterBuildStderr(raw)

	g.Expect(got).NotTo(ContainSubstring("user:pass"))
	g.Expect(got).NotTo(ContainSubstring("abc123"))
	g.Expect(got).NotTo(ContainSubstring("Bearer abc.def.ghi"))
	g.Expect(got).NotTo(ContainSubstring("supersecretvalue"))
	g.Expect(got).To(ContainSubstring("<redacted>"))
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestCaptureTailReturnsReadError(t *testing.T) {
	g := NewWithT(t)

	_, err := captureTail(failingReader{}, 10)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("read failed"))
}

func TestBuildCaptured_Success(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("resources: []\n"), 0o600)).To(Succeed())

	_, stderr, err := buildCaptured(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderr).To(BeEmpty())
}

func TestBuildCaptured_BuildError(t *testing.T) {
	g := NewWithT(t)
	dir := t.TempDir()

	_, _, err := buildCaptured(dir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("kustomize build"))
}

func TestBuildCaptured_CapturesStderr(t *testing.T) {
	g := NewWithT(t)

	oldBuild := buildOnce
	defer func() { buildOnce = oldBuild }()
	buildOnce = func(string) ([]byte, error) {
		fmt.Fprintln(os.Stderr, "helm: failed to download chart")
		return nil, errors.New("kustomize build failed")
	}

	_, stderr, err := buildCaptured(t.TempDir())
	g.Expect(err).To(HaveOccurred())
	g.Expect(stderr).To(ContainSubstring("helm: failed to download chart"))
}

func TestRedactSecrets_EnvironmentValue(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("MY_API_KEY", "bare-secret-value")

	got := redactSecrets("failure: bare-secret-value")
	g.Expect(got).NotTo(ContainSubstring("bare-secret-value"))
	g.Expect(got).To(ContainSubstring("<redacted>"))
}

func TestBuildCaptured_WaitsForQuietBuildsAndIsolatesStderr(t *testing.T) {
	g := NewWithT(t)
	original := os.Stderr

	oldBuild := buildOnce
	defer func() { buildOnce = oldBuild }()
	buildOnce = func(dir string) ([]byte, error) {
		switch dir {
		case "quiet1", "quiet2":
			fmt.Fprintln(os.Stderr, dir+"-marker")
			time.Sleep(10 * time.Millisecond)
			return []byte("ok"), nil
		case "capture":
			fmt.Fprintln(os.Stderr, "capture-marker")
			return nil, errors.New("helm pull failed")
		default:
			return nil, nil
		}
	}

	var wg sync.WaitGroup
	for _, dir := range []string{"quiet1", "quiet2"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = buildQuiet(dir)
		}()
	}
	_, stderr, err := buildCaptured("capture")
	wg.Wait()

	g.Expect(err).To(HaveOccurred())
	g.Expect(stderr).To(ContainSubstring("capture-marker"))
	g.Expect(stderr).NotTo(ContainSubstring("quiet1-marker"))
	g.Expect(stderr).NotTo(ContainSubstring("quiet2-marker"))
	g.Expect(os.Stderr).To(BeIdenticalTo(original))
}

func TestTailStringSingleLongLine(t *testing.T) {
	g := NewWithT(t)

	got := tailString(strings.Repeat("x", buildStderrLimit+100), buildStderrLimit)
	g.Expect(got).To(HavePrefix("[... truncated ...] "))
	g.Expect(len(got)).To(BeNumerically("<=", buildStderrLimit+len("[... truncated ...] ")))
}

// Package kustomize provides helpers for building kustomize overlays using the
// krusty library.
package kustomize

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/redhat-appstudio/infra-deployments/infra-tools/internal/logging"
)

const buildStderrLimit = 8 * 1024

var buildRetryDelay = time.Second

// newKustomizer builds a kustomizer configured the way every build needs it.
// A new instance per build is cheap and keeps the options immutable, so builds
// can run concurrently.
func newKustomizer() *krusty.Kustomizer {
	opts := krusty.MakeDefaultOptions()
	// Allow loading files from outside the kustomization root since overlays
	// reference ../../base/ paths.
	opts.LoadRestrictions = 0
	// Enable Helm chart inflation so components using
	// HelmChartInflationGenerator can be built.
	opts.PluginConfig.HelmConfig.Enabled = true
	opts.PluginConfig.HelmConfig.Command = "helm"
	return krusty.MakeKustomizer(opts)
}

// Build runs kustomize build on the given directory and returns the rendered
// YAML output as a byte slice.
//
// Deprecation warnings are normally silenced. If the build fails with a
// download/Helm/git error, it is retried once with stderr captured so the
// developer sees the error that kustomize would otherwise write only to stderr.
// Capture covers writes that look up os.Stderr; it does not redirect file
// descriptor 2, so a subprocess with its own stderr is not captured.
func Build(dir string) ([]byte, error) {
	return buildWithRetry(dir, buildQuiet, buildCaptured)
}

func buildWithRetry(
	dir string,
	first func(string) ([]byte, error),
	retry func(string) ([]byte, string, error),
) ([]byte, error) {
	yaml, err := first(dir)
	if err == nil {
		return yaml, nil
	}

	// Deterministic manifest errors have nothing to gain from a second build,
	// and a retry under the exclusive stderr gate stalls every other build.
	if !isRetryableBuildError(err) {
		return nil, redactedError(err)
	}
	if isTransientBuildError(err) {
		time.Sleep(buildRetryDelay)
	}
	slog.Warn("kustomize build failed, retrying with stderr capture",
		"dir", logging.Sanitize(dir),
		"err", logging.Sanitize(redactSecrets(tailString(err.Error(), buildStderrLimit))))

	yaml, stderr, retryErr := retry(dir)
	if retryErr == nil {
		slog.Info("kustomize build succeeded on retry", "dir", logging.Sanitize(dir))
		return yaml, nil
	}
	// Keep the first error for classification (render-diff matches its text to
	// skip non-kustomization directories), but include the retry's own text:
	// kustomize wraps Helm's stderr into the returned error rather than writing
	// it to the capture pipe.
	slog.Debug("kustomize retry failed", "dir", logging.Sanitize(dir),
		"err", logging.Sanitize(redactSecrets(tailString(retryErr.Error(), buildStderrLimit))))
	firstErr := sanitizeDiagnostic(redactSecrets(tailString(err.Error(), buildStderrLimit)))
	detail := filterBuildStderr(retryErr.Error())
	if detail != "" && detail != firstErr {
		firstErr += "\n\nretry: " + detail
	}
	return nil, withBuildStderr(&safeError{msg: firstErr, err: err}, stderr)
}

// buildQuiet runs a build with stderr muted. Normal builds use a shared
// read-lock so they stay concurrent.
func buildQuiet(dir string) ([]byte, error) {
	stderrGate.RLock()
	defer stderrGate.RUnlock()

	restoreStderr := muteStderr()
	defer restoreStderr()
	return run(dir)
}

// buildCaptured runs one build with stderr captured to a bounded in-memory
// tail. It takes an exclusive gate so no concurrent build can contaminate the
// capture. krusty has no cancellation API, so a hanging remote cannot be
// interrupted here; the job-level timeout is the hard bound.
func buildCaptured(dir string) ([]byte, string, error) {
	stderrGate.Lock()
	defer stderrGate.Unlock()

	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, "", fmt.Errorf("creating stderr capture: %w", err)
	}
	defer reader.Close()

	saved := os.Stderr
	os.Stderr = writer
	captured := make(chan captureResult, 1)
	go func() {
		data, readErr := captureTail(reader, 2*buildStderrLimit)
		captured <- captureResult{data: data, err: readErr}
	}()

	yaml, runErr := buildOnce(dir)
	os.Stderr = saved
	closeErr := writer.Close()
	result := <-captured
	stderr := string(result.data)

	if closeErr != nil {
		slog.Warn("closing stderr capture", "err", logging.Sanitize(closeErr.Error()))
	}
	if result.err != nil {
		slog.Warn("reading build stderr", "err", logging.Sanitize(result.err.Error()))
	}
	return yaml, stderr, runErr
}

type captureResult struct {
	data []byte
	err  error
}

// captureTail drains r and keeps only the last limit bytes, so a build that
// writes a lot of diagnostics cannot fill the runner's disk or memory.
func captureTail(r io.Reader, limit int) ([]byte, error) {
	var tail []byte
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			tail = append(tail, buf[:n]...)
			if len(tail) > limit {
				tail = tail[len(tail)-limit:]
			}
		}
		if err != nil {
			if err == io.EOF {
				return tail, nil
			}
			return tail, err
		}
	}
}

// buildOnce performs one kustomize build and wraps its errors with the
// directory being built. It is a variable so tests can stall a retry.
var buildOnce = func(dir string) ([]byte, error) {
	resMap, err := newKustomizer().Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build %s: %w", dir, err)
	}
	yamlBytes, err := resMap.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("converting resmap to yaml for %s: %w", dir, err)
	}
	return yamlBytes, nil
}

// run performs the actual kustomize build.
func run(dir string) ([]byte, error) {
	return buildOnce(dir)
}

// withBuildStderr appends filtered, size-capped build stderr to err.
func withBuildStderr(err error, raw string) error {
	stderr := filterBuildStderr(raw)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w\n\nbuild stderr:\n%s", err, stderr)
}

// safeError preserves the original error for errors.Is/As while exposing a
// redacted, size-capped message.
type safeError struct {
	msg string
	err error
}

func (e *safeError) Error() string { return e.msg }
func (e *safeError) Unwrap() error { return e.err }

// redactedError bounds and redacts an error message without losing the original
// error chain or the text render-diff uses to classify it.
func redactedError(err error) error {
	if err == nil {
		return nil
	}
	msg := sanitizeDiagnostic(redactSecrets(tailString(err.Error(), buildStderrLimit)))
	if msg == err.Error() {
		return err
	}
	return &safeError{msg: msg, err: err}
}

// tailString keeps the last limit bytes of text on a complete line, so a cut
// cannot split a credential such as token=abc into oken=abc.
func tailString(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := len(text) - limit
	for cut < len(text) && !utf8.RuneStart(text[cut]) {
		cut++
	}
	if cut > 0 {
		if next := strings.IndexByte(text[cut:], '\n'); next >= 0 {
			cut += next + 1
		} else {
			// A single very long line: keep a rune-aligned tail rather than
			// dropping the only diagnostic we have.
			return "[... truncated ...] " + text[cut:]
		}
	}
	return text[cut:]
}

// sanitizeDiagnostic neutralizes workflow commands in each line without
// collapsing the line structure of a build error.
func sanitizeDiagnostic(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = logging.Sanitize(lines[i])
	}
	return strings.Join(lines, "\n")
}

var (
	urlUserInfoRe  = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^\s]*@`)
	querySecretRe  = regexp.MustCompile(`(?i)\b((?:token|access_token|private_token|password|passwd|secret|api_key|apikey)=)[^&\s]+`)
	bearerTokenRe  = regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=-]+`)
	authHeaderRe   = regexp.MustCompile(`(?i)(authorization:\s*(?:basic|token|bearer)\s+)[^\s]+`)
	privateTokenRe = regexp.MustCompile(`(?i)(private-token:\s*)[^\s]+`)
	githubTokenRe  = regexp.MustCompile(`\b(?:ghp|gho|ghs|ghu|github_pat)_[A-Za-z0-9_]{20,}\b`)
)

// secretEnvName matches whole underscore-separated segments, so PATH and
// SSH_AUTH_SOCK are not treated as secrets while API_KEY and CLIENT_SECRET are.
func secretEnvName(key string) bool {
	upper := strings.ToUpper(key)
	if strings.HasSuffix(upper, "_PATH") || strings.HasSuffix(upper, "_SOCK") ||
		strings.Contains(upper, "AUTHORITY") {
		return false
	}
	for _, segment := range strings.Split(upper, "_") {
		switch segment {
		case "TOKEN", "SECRET", "PASSWORD", "PASSWD", "KEY", "CREDENTIAL", "CREDENTIALS", "AUTH", "PAT":
			return true
		}
	}
	return false
}

// redactSecrets removes credentials that a Helm/git failure may print in a URL
// or header. It also masks values of secret-looking environment variables.
func redactSecrets(s string) string {
	s = urlUserInfoRe.ReplaceAllString(s, `${1}<redacted>@`)
	s = querySecretRe.ReplaceAllString(s, `${1}<redacted>`)
	s = bearerTokenRe.ReplaceAllString(s, `${1} <redacted>`)
	s = authHeaderRe.ReplaceAllString(s, `${1}<redacted>`)
	s = privateTokenRe.ReplaceAllString(s, `${1}<redacted>`)
	s = githubTokenRe.ReplaceAllString(s, "<redacted>")
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || len(value) < 8 || !secretEnvName(key) {
			continue
		}
		s = strings.ReplaceAll(s, value, "<redacted>")
	}
	return s
}

// filterBuildStderr drops empty lines and kustomize deprecation warnings,
// neutralizes workflow-command syntax, then keeps only the tail of the output so
// a verbose Helm/git failure cannot flood the PR comment or job summary.
func filterBuildStderr(raw string) string {
	// Bound before splitting: kustomize can put large Helm stderr into the
	// returned error, and processing the whole string would defeat the cap.
	raw = redactSecrets(tailString(raw, 2*buildStderrLimit))
	var kept []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		deprecation := strings.TrimLeft(trimmed, "#- ")
		if strings.HasPrefix(deprecation, "Warning:") &&
			strings.Contains(strings.ToLower(deprecation), "deprecated") {
			continue
		}
		// Keep the line structure for readability, redact credentials, and
		// neutralize workflow commands and control characters from
		// PR-controlled build output.
		kept = append(kept, logging.Sanitize(line))
	}
	out := strings.TrimSpace(strings.Join(kept, "\n"))
	if len(out) > buildStderrLimit {
		out = "[... truncated ...]\n" + tailString(out, buildStderrLimit)
	}
	return out
}

var transientBuildErrorRe = regexp.MustCompile(`(?i)\b(?:timeout|timed out|deadline exceeded|connection reset|connection refused|temporary failure in name resolution|no such host|tls handshake timeout|i/o timeout|unexpected eof|rate limit(?:ed)?|network is unreachable|cannot be reached|service unavailable|internal server error|bad gateway|gateway timeout|too many requests)\b`)

var httpStatusRe = regexp.MustCompile(`(?i)\b(?:http/[\d.]+|http(?:\s+status)?|status(?:\s+code)?)\s*:?\s*(?:429|500|502|503|504)\b`)

// isTransientBuildError reports whether err looks like a retryable network or
// dependency-download failure.
func isTransientBuildError(err error) bool {
	if err == nil {
		return false
	}
	if httpStatusRe.MatchString(err.Error()) || transientBuildErrorRe.MatchString(err.Error()) {
		return true
	}
	return false
}

// isRetryableBuildError limits the diagnostic retry to failures that may be
// caused by a remote download or chart extraction. Other failures are
// deterministic and would only stall the build pool.
func isRetryableBuildError(err error) bool {
	if err == nil {
		return false
	}
	if isTransientBuildError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, pattern := range []string{
		"helm pull",
		"failed to untar",
		"failed to download",
		"failed to fetch",
		"unable to access",
		"couldn't find remote ref",
		"could not find remote ref",
		"unable to run",
	} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// stderrGate lets normal builds share stderr muting while a diagnostic retry
// takes exclusive ownership to capture that build's stderr.
var stderrGate sync.RWMutex

// Stderr muting is process-global because kustomize writes deprecation warnings
// to os.Stderr directly. Builds run concurrently (overlay builds in the
// detector and component builds in the render-diff engine), so the swap is
// reference-counted: the first build redirects stderr, the last one restores it.
//
// Only writes that look up os.Stderr while a build runs are muted. Our slog
// handlers bind os.Stderr when logging.Setup runs, so INFO/WARN diagnostics are
// unaffected; the muted output is kustomize's own deprecation warnings, which
// this package intentionally discards. Capturing those per build would require
// running kustomize as a subprocess or serializing builds.
var (
	stderrMu      sync.Mutex
	stderrRefs    int
	stderrSaved   *os.File
	stderrDevNull *os.File
)

// muteStderr redirects stderr to /dev/null until the returned function is
// called. It is safe for concurrent use.
func muteStderr() func() {
	stderrMu.Lock()
	defer stderrMu.Unlock()

	if stderrRefs == 0 {
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			// Can't mute: return a no-op restore rather than failing the build.
			return func() {}
		}
		stderrSaved = os.Stderr
		stderrDevNull = devNull
		os.Stderr = devNull
	}
	stderrRefs++

	var once sync.Once
	return func() {
		once.Do(func() {
			stderrMu.Lock()
			defer stderrMu.Unlock()

			stderrRefs--
			if stderrRefs > 0 {
				return
			}
			if stderrSaved != nil {
				os.Stderr = stderrSaved
				stderrSaved = nil
			}
			if stderrDevNull != nil {
				_ = stderrDevNull.Close()
				stderrDevNull = nil
			}
		})
	}
}

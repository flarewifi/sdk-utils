package sdkutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitOpTimeout bounds a single clone+checkout so a hung or throttled remote can't
// block an install goroutine (and the HTTP request waiting on it) indefinitely.
// Generous because a large plugin repo over a slow on-device link is legitimate.
const gitOpTimeout = 10 * time.Minute

type GitRepoSource struct {
	URL string
	Ref string // Can be branch, tag, commit, or empty
}

func getGitCachePath(repo GitRepoSource) string {
	// Slugify the credential-stripped URL so an embedded token never lands in a
	// predictable on-disk cache path (Slugify preserves alphanumerics verbatim).
	return filepath.Join(PathTmpDir, "git-cache", Slugify(stripGitCredentials(repo.URL), "_"), repo.Ref)
}

func GitIsCached(repo GitRepoSource) bool {
	// Only immutable refs (full commit SHAs) are cacheable. A branch or tag can
	// move, so a cached copy would silently serve a stale tree on the next install.
	return isImmutableGitRef(repo.Ref) && FsExists(getGitCachePath(repo))
}

func MakeGitCache(repo GitRepoSource, clonePath string) error {
	cachePath := getGitCachePath(repo)
	if err := FsEmptyDir(cachePath); err != nil {
		return err
	}
	// Copy the cloned repository to the cache directory
	if err := FsCopyDir(clonePath, cachePath, nil); err != nil {
		return err
	}
	return nil
}

func GitClone(repo GitRepoSource, clonePath string) error {
	// Ensure the parent exists without disturbing sibling clones (this dir may be
	// shared by concurrent installs); clear only our own target below.
	parentDir := filepath.Dir(clonePath)
	if err := FsEnsureDir(parentDir); err != nil {
		return err
	}

	// Defense in depth against argument injection: a URL or ref beginning with "-"
	// would be parsed by git as an option. We also terminate options with "--".
	if strings.HasPrefix(repo.URL, "-") {
		return errors.New("invalid git URL")
	}
	if strings.HasPrefix(repo.Ref, "-") {
		return errors.New("invalid git ref")
	}

	if GitIsCached(repo) {
		if err := FsEmptyDir(clonePath); err != nil {
			return err
		}
		return FsCopyDir(getGitCachePath(repo), clonePath, nil)
	}

	// git clone needs a nonexistent (or empty) target.
	if err := os.RemoveAll(clonePath); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitOpTimeout)
	defer cancel()

	// Clone the repository. Errors are scrubbed of any credentials embedded in the
	// URL before they propagate (git echoes the remote URL in its stderr).
	var stderr strings.Builder
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--", repo.URL, clonePath)
	cloneCmd.Stderr = &stderr
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %s: %s", err.Error(), redactGitSecrets(stderr.String(), repo.URL))
	}

	// If a specific ref (branch, tag, commit) is provided, check it out.
	if repo.Ref != "" {
		var coErr strings.Builder
		checkoutCmd := exec.CommandContext(ctx, "git", "-C", clonePath, "checkout", repo.Ref)
		checkoutCmd.Stderr = &coErr
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("git checkout %s failed: %s: %s", repo.Ref, err.Error(), redactGitSecrets(coErr.String(), repo.URL))
		}

		// Cache only immutable refs — a moved branch/tag must be re-cloned, not
		// served stale from the cache (see GitIsCached).
		if isImmutableGitRef(repo.Ref) {
			if err := MakeGitCache(repo, clonePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func DownloadGitHubTarball(repoUrl string, outputFile string) error {
	repo, err := ParseGitSource(repoUrl)
	if err != nil {
		return err
	}

	// Construct the GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", repo.Owner, repo.Repo, repo.Ref)

	// Create an HTTP client and a request
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if repo.Token != "" {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", repo.Token))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Copy the response body to the output file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	return nil
}

func NeutralizeGitURL(url string) string {
	if strings.HasSuffix(url, ".git") {
		return strings.Replace(url, ".git", "", 1)
	}
	return url
}

// GitSource represents the parsed components of a Git source URL.
type GitSource struct {
	Owner string
	Repo  string
	Ref   string
	Token string // Field for the access token
}

// ParseGitSource parses a Git source URL into a GitSource struct.
func ParseGitSource(sourceUrl string) (source GitSource, err error) {
	// Parse the URL
	u, err := url.Parse(sourceUrl)
	if err != nil {
		return source, errors.New("invalid URL format")
	}

	// Check the URL scheme
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "git" {
		return source, errors.New("unsupported URL scheme: " + sourceUrl)
	}

	// Extract the path components
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		return source, errors.New("URL must include at least owner and repository")
	}

	// Extract owner and repo
	source.Owner = pathParts[0]
	source.Repo = strings.TrimSuffix(pathParts[1], ".git") // Remove .git if present

	// Extract ref from query parameters if present
	queryRef := u.Query().Get("ref")
	if queryRef != "" {
		source.Ref = queryRef
	} else if len(pathParts) > 2 {
		// Assume ref if a third path component exists
		source.Ref = pathParts[2]
	}

	// Extract access token if present in the User part of the URL
	if u.User != nil {
		username := u.User.Username()
		if strings.HasPrefix(username, "oauth2:") {
			source.Token = strings.TrimPrefix(username, "oauth2:")
		}

		password, ok := u.User.Password()
		if source.Token == "" && ok {
			source.Token = password
		}
	}

	return source, nil
}

// stripGitCredentials returns rawURL with any embedded user-info (a token or
// user:password) removed. It falls back to the original string only when the URL
// is unparseable — in which case a clone would fail on it anyway.
func stripGitCredentials(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = nil
	return u.String()
}

// redactGitSecrets scrubs any credentials that a git error message may echo: the
// full credential-bearing URL is swapped for its stripped form and the bare
// user-info token is masked. rawURL is the source URL the message may quote.
func redactGitSecrets(msg, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return msg
	}
	msg = strings.ReplaceAll(msg, rawURL, stripGitCredentials(rawURL))
	if ui := u.User.String(); ui != "" {
		msg = strings.ReplaceAll(msg, ui, "***")
	}
	return msg
}

// isImmutableGitRef reports whether ref is a full Git object name (40 hex chars
// for SHA-1, 64 for SHA-256). Such a ref is immutable and safe to cache. Branch
// names, tags and abbreviated SHAs are treated as mutable and never cached.
func isImmutableGitRef(ref string) bool {
	if len(ref) != 40 && len(ref) != 64 {
		return false
	}
	for _, c := range ref {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

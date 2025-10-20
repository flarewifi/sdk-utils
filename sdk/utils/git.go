package sdkutils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitRepoSource struct {
	URL string
	Ref string // Can be branch, tag, commit, or empty
}

func getGitCachePath(repo GitRepoSource) string {
	return filepath.Join(PathTmpDir, "git-cache", Slugify(repo.URL, "_"), repo.Ref)
}

func GitIsCached(repo GitRepoSource) bool {
	cachePath := getGitCachePath(repo)
	return repo.Ref != "" && FsExists(cachePath)
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
	log.Printf("Repository cached to %s", cachePath)
	return nil
}

func GitClone(repo GitRepoSource, clonePath string) error {
	// Ensure the parent directory of clonePath exists
	parentDir := filepath.Dir(clonePath)
	if err := FsEmptyDir(parentDir); err != nil {
		return err
	}

	if GitIsCached(repo) {
		cachePath := getGitCachePath(repo)
		if err := FsCopyDir(cachePath, clonePath, nil); err != nil {
			return err
		}
	} else {
		// Clone the repository using the "git clone" command with the provided URL
		var stderr strings.Builder
		cmd := exec.Command("git", "clone", repo.URL, clonePath)
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Error: %s\nStderr: %s", err.Error(), stderr.String())
		}

		log.Printf("Repository cloned to %s", clonePath)

		// If a specific ref (branch, tag, commit) is provided, checkout that ref
		if repo.Ref != "" {
			// Prepare the checkout command
			checkoutCmd := exec.Command("git", "checkout", repo.Ref)
			checkoutCmd.Stdout = os.Stdout
			checkoutCmd.Stderr = os.Stderr
			checkoutCmd.Dir = clonePath // Set the working directory for the command
			if err := checkoutCmd.Run(); err != nil {
				return err
			}

			if err := MakeGitCache(repo, clonePath); err != nil {
				return err
			}

			log.Printf("Checked out ref %s", repo.Ref)
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

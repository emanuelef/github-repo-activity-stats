package repostats

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// RepoMentionREST represents a mention found via REST API
type RepoMentionREST struct {
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	URL        string    `json:"html_url"`
	Repository string    `json:"repository"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	State      string    `json:"state"`
	Author     string    `json:"author"`
	Body       string    `json:"body_preview"`
}

// RepoMentionResultREST contains search results from REST API
type RepoMentionResultREST struct {
	TargetRepo        string            `json:"target_repo"`
	TotalMentions     int               `json:"total_mentions"`
	IssuesCount       int               `json:"issues_count"`
	PullRequestsCount int               `json:"pull_requests_count"`
	Mentions          []RepoMentionREST `json:"mentions"`
}

// GitHubSearchResponse represents the GitHub API search response
type GitHubSearchResponse struct {
	TotalCount        int  `json:"total_count"`
	IncompleteResults bool `json:"incomplete_results"`
	Items             []struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		HTMLURL   string `json:"html_url"`
		State     string `json:"state"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
		User      struct {
			Login string `json:"login"`
		} `json:"user"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository_url"` // This needs special handling
		RepositoryURL string `json:"repository_url"`
		PullRequest   *struct {
			URL string `json:"url"`
		} `json:"pull_request,omitempty"`
	} `json:"items"`
}

// GetRepoMentionsREST searches for mentions using GitHub REST API
// This is an alternative to the GraphQL version
// Results are sorted by creation date (most recent first)
func (c *Client) GetRepoMentionsREST(repo string, limitPerType int) (*RepoMentionResultREST, error) {
	return c.GetRepoMentionsRESTWithTimeRange(repo, limitPerType, nil, nil)
}

// GetRepoMentionsRESTWithTimeRange searches for mentions with optional time range filtering
// startDate and endDate are optional (pass nil to ignore)
// Results are sorted by creation date (most recent first)
// Example: GetRepoMentionsRESTWithTimeRange("kubernetes/kubernetes", 30, &startDate, &endDate)
func (c *Client) GetRepoMentionsRESTWithTimeRange(repo string, limitPerType int, startDate, endDate *time.Time) (*RepoMentionResultREST, error) {
	if limitPerType <= 0 {
		limitPerType = 30
	}
	if limitPerType > 100 {
		limitPerType = 100 // GitHub API limit
	}

	// Build date range string for query
	dateRange := ""
	if startDate != nil {
		dateRange += fmt.Sprintf(" created:>=%s", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		dateRange += fmt.Sprintf(" created:<=%s", endDate.Format("2006-01-02"))
	}

	result := &RepoMentionResultREST{
		TargetRepo: repo,
		Mentions:   []RepoMentionREST{},
	}

	// Search for issues (excluding PRs)
	issuesQuery := fmt.Sprintf("%s in:body,title type:issue is:public -repo:%s%s", repo, repo, dateRange)
	issues, err := c.searchGitHubREST(issuesQuery, limitPerType, "Issue")
	if err != nil {
		return nil, fmt.Errorf("error searching issues: %w", err)
	}
	result.Mentions = append(result.Mentions, issues...)
	result.IssuesCount = len(issues)

	// Search for pull requests
	prsQuery := fmt.Sprintf("%s in:body,title type:pr is:public -repo:%s%s", repo, repo, dateRange)
	prs, err := c.searchGitHubREST(prsQuery, limitPerType, "PullRequest")
	if err != nil {
		return nil, fmt.Errorf("error searching pull requests: %w", err)
	}
	result.Mentions = append(result.Mentions, prs...)
	result.PullRequestsCount = len(prs)

	result.TotalMentions = len(result.Mentions)

	return result, nil
}

// searchGitHubREST performs the actual REST API search
func (c *Client) searchGitHubREST(query string, limit int, itemType string) ([]RepoMentionREST, error) {
	searchResp := &GitHubSearchResponse{}

	resp, err := c.restyClient.R().
		SetResult(searchResp).
		SetQueryParams(map[string]string{
			"q":        query,
			"per_page": fmt.Sprintf("%d", limit),
			"sort":     "created",
			"order":    "desc",
		}).
		Get(fmt.Sprintf("%s/search/issues", apiGHUrl))

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	mentions := []RepoMentionREST{}
	for _, item := range searchResp.Items {
		// Extract repository name from repository_url
		// Format: https://api.github.com/repos/owner/repo
		repoName := extractRepoFromURL(item.RepositoryURL)

		createdAt, _ := time.Parse(time.RFC3339, item.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, item.UpdatedAt)

		body := item.Body
		if len(body) > 200 {
			body = body[:200] + "..."
		}

		mention := RepoMentionREST{
			Type:       itemType,
			Title:      item.Title,
			URL:        item.HTMLURL,
			Repository: repoName,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			State:      item.State,
			Author:     item.User.Login,
			Body:       body,
		}

		mentions = append(mentions, mention)
	}

	log.Printf("Found %d %s mentions", len(mentions), itemType)
	return mentions, nil
}

// extractRepoFromURL extracts owner/repo from GitHub API URL
// Example: https://api.github.com/repos/kubernetes/kubernetes -> kubernetes/kubernetes
func extractRepoFromURL(url string) string {
	// URL format: https://api.github.com/repos/owner/repo
	const prefix = "https://api.github.com/repos/"
	if len(url) > len(prefix) {
		return url[len(prefix):]
	}
	return ""
}

// GetRepoMentionsSummaryREST returns a quick summary of mentions
func (c *Client) GetRepoMentionsSummaryREST(repo string) (map[string]int, error) {
	summary := make(map[string]int)

	// Quick search with limit of 1 just to get total_count
	for _, searchType := range []struct {
		name  string
		query string
	}{
		{"issues", fmt.Sprintf("%s in:body,title type:issue is:public -repo:%s", repo, repo)},
		{"pull_requests", fmt.Sprintf("%s in:body,title type:pr is:public -repo:%s", repo, repo)},
	} {
		searchResp := &GitHubSearchResponse{}

		resp, err := c.restyClient.R().
			SetResult(searchResp).
			SetQueryParams(map[string]string{
				"q":        searchType.query,
				"per_page": "1",
			}).
			Get(fmt.Sprintf("%s/search/issues", apiGHUrl))

		if err != nil {
			return nil, err
		}

		if resp.StatusCode() == http.StatusOK {
			summary[searchType.name] = searchResp.TotalCount
		}
	}

	summary["total"] = summary["issues"] + summary["pull_requests"]

	return summary, nil
}

package repostats

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

// RepoMention represents a mention of a repository in an issue, PR, or discussion
type RepoMention struct {
	Type       string // "Issue", "PullRequest", or "Discussion"
	Title      string
	URL        string
	Repository string // owner/name of the repo where the mention was found
	CreatedAt  time.Time
	UpdatedAt  time.Time
	State      string
	Author     string
	Body       string // First 200 chars of the body
	IsClosed   bool
}

// RepoMentionResult contains all mentions found for a repository
type RepoMentionResult struct {
	TargetRepo        string
	TotalMentions     int
	IssuesCount       int
	PullRequestsCount int
	DiscussionsCount  int
	Mentions          []RepoMention
}

// GetRepoMentions searches for mentions of a specific repository across all public GitHub repos
// It searches in Issues, Pull Requests, and Discussions, sorted by most recent
// Example: repo = "kubernetes/kubernetes"
// Note: Results are sorted by creation date (most recent first)
func (c *ClientGQL) GetRepoMentions(ctx context.Context, repo string, limit int) (*RepoMentionResult, error) {
	return c.GetRepoMentionsWithTimeRange(ctx, repo, limit, nil, nil)
}

// GetRepoMentionsWithTimeRange searches for mentions with optional time range filtering
// startDate and endDate are optional (pass nil to ignore)
// Results are sorted by creation date (most recent first)
// Time format: "2024-01-01" or use time.Time
// Example: GetRepoMentionsWithTimeRange(ctx, "kubernetes/kubernetes", 50, &startDate, &endDate)
func (c *ClientGQL) GetRepoMentionsWithTimeRange(ctx context.Context, repo string, limit int, startDate, endDate *time.Time) (*RepoMentionResult, error) {
	if limit <= 0 {
		limit = 100
	}

	// Build date range string for query
	dateRange := ""
	if startDate != nil {
		dateRange += fmt.Sprintf(" created:>=%s", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		dateRange += fmt.Sprintf(" created:<=%s", endDate.Format("2006-01-02"))
	}

	result := &RepoMentionResult{
		TargetRepo: repo,
		Mentions:   []RepoMention{},
	}

	// Search for Issues
	issues, err := c.searchIssues(ctx, repo, limit, dateRange)
	if err != nil {
		return nil, fmt.Errorf("error searching issues: %w", err)
	}
	result.Mentions = append(result.Mentions, issues...)
	result.IssuesCount = len(issues)

	// Search for Pull Requests
	prs, err := c.searchPullRequests(ctx, repo, limit, dateRange)
	if err != nil {
		return nil, fmt.Errorf("error searching pull requests: %w", err)
	}
	result.Mentions = append(result.Mentions, prs...)
	result.PullRequestsCount = len(prs)

	// Search for Discussions
	discussions, err := c.searchDiscussions(ctx, repo, limit, dateRange)
	if err != nil {
		return nil, fmt.Errorf("error searching discussions: %w", err)
	}
	result.Mentions = append(result.Mentions, discussions...)
	result.DiscussionsCount = len(discussions)

	result.TotalMentions = len(result.Mentions)

	return result, nil
}

// searchIssues searches for issues mentioning the target repo
// Results are sorted by creation date (most recent first)
func (c *ClientGQL) searchIssues(ctx context.Context, repo string, limit int, dateRange string) ([]RepoMention, error) {
	var query struct {
		Search struct {
			IssueCount int
			Edges      []struct {
				Node struct {
					Issue struct {
						Title     string
						URL       string
						CreatedAt time.Time
						UpdatedAt time.Time
						State     string
						Body      string
						Closed    bool
						Author    struct {
							Login string
						}
						Repository struct {
							NameWithOwner string
						}
					} `graphql:"... on Issue"`
				}
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"search(query: $searchQuery, type: ISSUE, first: $first, after: $cursor)"`
	}

	variables := map[string]any{
		"searchQuery": githubv4.String(fmt.Sprintf("%s in:body,title type:issue is:public -repo:%s sort:created-desc%s", repo, repo, dateRange)),
		"first":       githubv4.Int(min(limit, 100)),
		"cursor":      (*githubv4.String)(nil),
	}

	mentions := []RepoMention{}

	for {
		err := c.query(ctx, &query, variables)
		if err != nil {
			return mentions, err
		}

		for _, edge := range query.Search.Edges {
			issue := edge.Node.Issue
			body := issue.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}

			mentions = append(mentions, RepoMention{
				Type:       "Issue",
				Title:      issue.Title,
				URL:        issue.URL,
				Repository: issue.Repository.NameWithOwner,
				CreatedAt:  issue.CreatedAt,
				UpdatedAt:  issue.UpdatedAt,
				State:      issue.State,
				Author:     issue.Author.Login,
				Body:       body,
				IsClosed:   issue.Closed,
			})
		}

		if !query.Search.PageInfo.HasNextPage || len(mentions) >= limit {
			break
		}

		variables["cursor"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	return mentions, nil
}

// searchPullRequests searches for pull requests mentioning the target repo
// Results are sorted by creation date (most recent first)
func (c *ClientGQL) searchPullRequests(ctx context.Context, repo string, limit int, dateRange string) ([]RepoMention, error) {
	var query struct {
		Search struct {
			IssueCount int
			Edges      []struct {
				Node struct {
					PullRequest struct {
						Title     string
						URL       string
						CreatedAt time.Time
						UpdatedAt time.Time
						State     string
						Body      string
						Closed    bool
						Author    struct {
							Login string
						}
						Repository struct {
							NameWithOwner string
						}
					} `graphql:"... on PullRequest"`
				}
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"search(query: $searchQuery, type: ISSUE, first: $first, after: $cursor)"`
	}

	variables := map[string]any{
		"searchQuery": githubv4.String(fmt.Sprintf("%s in:body,title type:pr is:public -repo:%s sort:created-desc%s", repo, repo, dateRange)),
		"first":       githubv4.Int(min(limit, 100)),
		"cursor":      (*githubv4.String)(nil),
	}

	mentions := []RepoMention{}

	for {
		err := c.query(ctx, &query, variables)
		if err != nil {
			return mentions, err
		}

		for _, edge := range query.Search.Edges {
			pr := edge.Node.PullRequest
			body := pr.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}

			mentions = append(mentions, RepoMention{
				Type:       "PullRequest",
				Title:      pr.Title,
				URL:        pr.URL,
				Repository: pr.Repository.NameWithOwner,
				CreatedAt:  pr.CreatedAt,
				UpdatedAt:  pr.UpdatedAt,
				State:      pr.State,
				Author:     pr.Author.Login,
				Body:       body,
				IsClosed:   pr.Closed,
			})
		}

		if !query.Search.PageInfo.HasNextPage || len(mentions) >= limit {
			break
		}

		variables["cursor"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	return mentions, nil
}

// searchDiscussions searches for discussions mentioning the target repo
// Results are sorted by creation date (most recent first)
func (c *ClientGQL) searchDiscussions(ctx context.Context, repo string, limit int, dateRange string) ([]RepoMention, error) {
	var query struct {
		Search struct {
			DiscussionCount int
			Edges           []struct {
				Node struct {
					Discussion struct {
						Title     string
						URL       string
						CreatedAt time.Time
						UpdatedAt time.Time
						Body      string
						Closed    bool
						Author    struct {
							Login string
						}
						Repository struct {
							NameWithOwner string
						}
					} `graphql:"... on Discussion"`
				}
			}
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"search(query: $searchQuery, type: DISCUSSION, first: $first, after: $cursor)"`
	}

	variables := map[string]any{
		"searchQuery": githubv4.String(fmt.Sprintf("%s in:body,title -repo:%s sort:created-desc%s", repo, repo, dateRange)),
		"first":       githubv4.Int(min(limit, 100)),
		"cursor":      (*githubv4.String)(nil),
	}

	mentions := []RepoMention{}

	for {
		err := c.query(ctx, &query, variables)
		if err != nil {
			return mentions, err
		}

		for _, edge := range query.Search.Edges {
			discussion := edge.Node.Discussion
			body := discussion.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}

			state := "OPEN"
			if discussion.Closed {
				state = "CLOSED"
			}

			mentions = append(mentions, RepoMention{
				Type:       "Discussion",
				Title:      discussion.Title,
				URL:        discussion.URL,
				Repository: discussion.Repository.NameWithOwner,
				CreatedAt:  discussion.CreatedAt,
				UpdatedAt:  discussion.UpdatedAt,
				State:      state,
				Author:     discussion.Author.Login,
				Body:       body,
				IsClosed:   discussion.Closed,
			})
		}

		if !query.Search.PageInfo.HasNextPage || len(mentions) >= limit {
			break
		}

		variables["cursor"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}

	return mentions, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

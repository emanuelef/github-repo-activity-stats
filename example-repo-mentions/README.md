# GitHub Repository Mentions API

This API allows you to search for mentions of a specific GitHub repository across all public repositories on GitHub. It searches in:
- **Issues**
- **Pull Requests**
- **Discussions** (GraphQL only)

## Features

- Two implementation options: **GraphQL** and **REST API**
- Configurable result limits
- **Time range filtering** to search within specific date ranges
- **Results sorted by most recent** (creation date, descending)
- Detailed mention information including:
  - Title and body preview
  - Repository where the mention was found
  - Author, state, and timestamps
  - Direct URL to the mention

## Setup

1. Create a GitHub Personal Access Token (PAT):
   - Go to https://github.com/settings/tokens
   - Generate a new token with `public_repo` scope
   - Set it as an environment variable: `export PAT=your_token_here`

2. Install dependencies:
   ```bash
   go mod download
   ```

## Usage

### GraphQL API (Recommended)

The GraphQL API provides more detailed information and includes Discussions.

**Basic Usage (Most Recent Results):**

```go
import (
    "context"
    "github.com/emanuelef/github-repo-activity-stats/repostats"
    "golang.org/x/oauth2"
)

// Create authenticated client
tokenSource := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: "your_github_token"},
)
oauthClient := oauth2.NewClient(context.Background(), tokenSource)
client := repostats.NewClientGQL(oauthClient)

// Search for mentions (returns most recent first)
result, err := client.GetRepoMentions(
    context.Background(),
    "kubernetes/kubernetes", // repository to search for
    50,                      // limit per type (issues, PRs, discussions)
)

// Access results
fmt.Printf("Total mentions: %d\n", result.TotalMentions)
fmt.Printf("Issues: %d\n", result.IssuesCount)
fmt.Printf("Pull Requests: %d\n", result.PullRequestsCount)
fmt.Printf("Discussions: %d\n", result.DiscussionsCount)

for _, mention := range result.Mentions {
    fmt.Printf("[%s] %s - %s\n", mention.Type, mention.Title, mention.URL)
}
```

**With Time Range (Get Most Recent in Period):**

```go
import "time"

// Get mentions from the last 30 days
endDate := time.Now()
startDate := endDate.AddDate(0, 0, -30) // 30 days ago

result, err := client.GetRepoMentionsWithTimeRange(
    context.Background(),
    "kubernetes/kubernetes",
    50,
    &startDate,  // optional: pass nil for no start limit
    &endDate,    // optional: pass nil for no end limit
)

// Or custom date range
startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

result, err := client.GetRepoMentionsWithTimeRange(
    context.Background(),
    "kubernetes/kubernetes",
    100,
    &startDate,
    &endDate,
)
```

### REST API

The REST API is simpler but doesn't support Discussions search.

**Basic Usage (Most Recent Results):**

```go
import (
    "github.com/emanuelef/github-repo-activity-stats/repostats"
    "golang.org/x/oauth2"
)

// Create authenticated client
tokenSource := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: "your_github_token"},
)
oauthClient := oauth2.NewClient(context.Background(), tokenSource)
client := repostats.NewClient(&oauthClient.Transport)

// Quick summary (just counts)
summary, err := client.GetRepoMentionsSummaryREST("kubernetes/kubernetes")
fmt.Printf("Total: %d, Issues: %d, PRs: %d\n", 
    summary["total"], summary["issues"], summary["pull_requests"])

// Detailed results (sorted by most recent)
result, err := client.GetRepoMentionsREST(
    "kubernetes/kubernetes", // repository to search for
    30,                      // limit per type (max 100)
)

for _, mention := range result.Mentions {
    fmt.Printf("[%s] %s\n", mention.Type, mention.Title)
}
```

**With Time Range:**

```go
import "time"

// Last 7 days
endDate := time.Now()
startDate := endDate.AddDate(0, 0, -7)

result, err := client.GetRepoMentionsRESTWithTimeRange(
    "kubernetes/kubernetes",
    30,
    &startDate,  // optional: pass nil for no start limit
    &endDate,    // optional: pass nil for no end limit
)
```

## Run Examples

### GraphQL Example
```bash
cd example-repo-mentions
go run main.go
```

### GraphQL with Time Range Example
```bash
cd example-repo-mentions
go run main_with_timerange.go
```

### REST API Example
```bash
cd example-repo-mentions-rest
go run main.go
```

Both examples will:
1. Search for mentions of `kubernetes/kubernetes` (you can change this)
2. Display a summary and first 10 results (sorted by most recent)
3. Save all results to a JSON file

## How the Limit Works

**Important:** The limit parameter controls how many results are fetched **per type** (Issues, PRs, Discussions).

- If you set `limit = 50`, you'll get up to 50 issues + 50 PRs + 50 discussions = 150 total mentions
- Results are **sorted by creation date (most recent first)**
- This ensures you always get the newest mentions
- Use time range filtering to get recent mentions within a specific period

## API Response Structure

### GraphQL Response (`RepoMentionResult`)
```go
type RepoMentionResult struct {
    TargetRepo         string        // The repository you searched for
    TotalMentions      int           // Total number of mentions found
    IssuesCount        int           // Number of issue mentions
    PullRequestsCount  int           // Number of PR mentions
    DiscussionsCount   int           // Number of discussion mentions
    Mentions           []RepoMention // All mentions
}

type RepoMention struct {
    Type        string    // "Issue", "PullRequest", or "Discussion"
    Title       string    // Title of the issue/PR/discussion
    URL         string    // Direct URL to the mention
    Repository  string    // Repo where mention was found (owner/name)
    CreatedAt   time.Time // Creation timestamp
    UpdatedAt   time.Time // Last update timestamp
    State       string    // "OPEN", "CLOSED", etc.
    Author      string    // GitHub username of author
    Body        string    // First 200 chars of body
    IsClosed    bool      // Whether it's closed
}
```

### REST Response (`RepoMentionResultREST`)
```go
type RepoMentionResultREST struct {
    TargetRepo        string             // The repository you searched for
    TotalMentions     int                // Total mentions retrieved
    IssuesCount       int                // Number of issue mentions
    PullRequestsCount int                // Number of PR mentions
    Mentions          []RepoMentionREST  // All mentions
}
```

## JSON Response Example

Here's an example of what the API returns in JSON format:

```json
{
  "TargetRepo": "kubernetes/kubernetes",
  "TotalMentions": 4,
  "IssuesCount": 2,
  "PullRequestsCount": 1,
  "DiscussionsCount": 1,
  "Mentions": [
    {
      "Type": "Issue",
      "Title": "Add support for Kubernetes 1.28 compatibility",
      "URL": "https://github.com/example-org/cloud-platform/issues/1234",
      "Repository": "example-org/cloud-platform",
      "CreatedAt": "2026-02-03T14:22:33Z",
      "UpdatedAt": "2026-02-04T09:15:42Z",
      "State": "OPEN",
      "Author": "developer123",
      "Body": "We need to update our dependencies to support kubernetes/kubernetes v1.28. This includes updating the client-go library and testing against the new API versions...",
      "IsClosed": false
    },
    {
      "Type": "Discussion",
      "Title": "Best practices for scaling kubernetes/kubernetes clusters in production",
      "URL": "https://github.com/devops-community/infrastructure/discussions/42",
      "Repository": "devops-community/infrastructure",
      "CreatedAt": "2026-02-02T10:18:27Z",
      "UpdatedAt": "2026-02-03T15:42:09Z",
      "State": "OPEN",
      "Author": "sre-expert",
      "Body": "I'm looking for advice on scaling kubernetes/kubernetes clusters for production workloads. We're currently running 50+ nodes and experiencing some performance issues. Has anyone dealt wit...",
      "IsClosed": false
    },
    {
      "Type": "PullRequest",
      "Title": "Upgrade to kubernetes/kubernetes v1.27",
      "URL": "https://github.com/another-org/deployment-tool/pull/567",
      "Repository": "another-org/deployment-tool",
      "CreatedAt": "2026-02-01T08:45:12Z",
      "UpdatedAt": "2026-02-02T16:33:21Z",
      "State": "MERGED",
      "Author": "contributor456",
      "Body": "This PR upgrades our kubernetes/kubernetes dependencies from v1.26 to v1.27. Key changes include:\n- Updated client-go to v0.27.0\n- Fixed deprecated API usage\n- Added new feature suppor...",
      "IsClosed": true
    },
    {
      "Type": "Issue",
      "Title": "Question about migrating from docker-compose to kubernetes",
      "URL": "https://github.com/startup-project/backend/issues/89",
      "Repository": "startup-project/backend",
      "CreatedAt": "2026-01-28T11:30:45Z",
      "UpdatedAt": "2026-01-30T14:22:11Z",
      "State": "CLOSED",
      "Author": "newbie789",
      "Body": "We're considering migrating our application to kubernetes/kubernetes. What are the best practices for converting our docker-compose.yml files to Kubernetes manifests? Any recommended to...",
      "IsClosed": true
    }
  ]
}
```

**Note:** The mentions are sorted by creation date with the most recent first.

## Rate Limits

- **REST API**: 30 requests per minute (authenticated)
- **GraphQL API**: 5,000 points per hour

The GraphQL API is generally more efficient for this use case.

## Use Cases

1. **Monitor where your project is being discussed**
   - See which projects are mentioning your repository
   - Track adoption and usage patterns
   - Get **most recent mentions** to stay up-to-date

2. **Find integrations and comparisons**
   - Discover projects that integrate with yours
   - See where your project is being compared to alternatives

3. **Community engagement**
   - Find discussions about your project in other repositories
   - Respond to questions or issues mentioned elsewhere
   - Track mentions over specific time periods (e.g., after a major release)

4. **Market research**
   - See how competitors' projects are being discussed
   - Understand the ecosystem around similar projects
   - Analyze trends over time with date range filtering

5. **Release impact analysis**
   - Check mentions before and after a release
   - Use time ranges to compare different periods
   - Track community response to specific features

## Time Range Examples

```go
// Last 24 hours
endDate := time.Now()
startDate := endDate.Add(-24 * time.Hour)

// Last week
startDate := time.Now().AddDate(0, 0, -7)
endDate := time.Now()

// Last month
startDate := time.Now().AddDate(0, -1, 0)
endDate := time.Now()

// Specific month (January 2024)
startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

// Since a specific date (no end date)
startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
result, _ := client.GetRepoMentionsWithTimeRange(ctx, repo, 100, &startDate, nil)

// Until a specific date (no start date)
endDate := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC)
result, _ := client.GetRepoMentionsWithTimeRange(ctx, repo, 100, nil, &endDate)
```

## Limitations

- Maximum 1,000 search results per query (GitHub API limit)
- Search is limited to public repositories
- Some very old mentions might not be indexed by GitHub's search
- Discussions search only available via GraphQL API

## Tips

- Use authentication to get higher rate limits
- Start with smaller limit values to test
- For large-scale searches, implement pagination and rate limiting
- Consider caching results to reduce API calls

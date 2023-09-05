package repostats

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shurcooL/githubv4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/emanuelef/cncf-repos-stats")
}

type ClientGQL struct {
	ghClient    *githubv4.Client
	restyClient *resty.Client
}

func NewClientGQL(oauthClient *http.Client) *ClientGQL {
	ghClient := githubv4.NewClient(oauthClient)
	restyClient := resty.NewWithClient(
		&http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	)
	return &ClientGQL{ghClient: ghClient, restyClient: restyClient}
}

func (c *ClientGQL) getStarsHistory(ctx context.Context, owner, name string, totalStars int) (StarsHistory, error) {
	result := StarsHistory{}

	/*
		{
			repository(owner: "kubernetes", name: "kubernetes") {
			  stargazers(last: 100) {
				totalCount
				edges {
				  starredAt
				  cursor
				}
			  }
			}
			rateLimit {
			  limit
			  cost
			  remaining
			  resetAt
			}
		  }
	*/

	variablesStars := map[string]interface{}{
		"owner":       githubv4.String(owner),
		"name":        githubv4.String(name),
		"starsCursor": (*githubv4.String)(nil),
	}

	type starred struct {
		StarredAt time.Time
	}

	var queryStars struct {
		Repository struct {
			Stargazers struct {
				TotalCount int
				Edges      []starred
				PageInfo   struct {
					StartCursor     githubv4.String
					HasPreviousPage bool
				}
			} `graphql:"stargazers(last: 100, before: $starsCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		err := c.ghClient.Query(context.Background(), &queryStars, variablesStars)
		if err != nil {
			// Handle error.
		}

		// fmt.Println("Desc:", len(queryStars.Repository.Stargazers.Edges))

		res := queryStars.Repository.Stargazers.Edges

		currentTime := time.Now()
		slices.Reverse(res) // order from most recent to least

		if len(res) > 0 && result.LastStarDate.IsZero() {
			result.LastStarDate = res[0].StarredAt
		}

		moreThan30daysFlag := false

		for _, star := range res {
			if err == nil {
				days := currentTime.Sub(star.StarredAt).Hours() / 24

				if days <= 1 {
					result.AddedLast24H += 1
				}

				if days <= 7 {
					result.AddedLast7d += 1
				}

				if days <= 14 {
					result.AddedLast14d += 1
				}

				if days <= 30 {
					result.AddedLast30d += 1
				}

				if days > 30 {
					moreThan30daysFlag = true
					break
				}
			}
		}

		if !queryStars.Repository.Stargazers.PageInfo.HasPreviousPage || moreThan30daysFlag {
			break
		}

		variablesStars["starsCursor"] = githubv4.NewString(queryStars.Repository.Stargazers.PageInfo.StartCursor)
	}

	if totalStars > 0 {
		result.AddedPerMille30d = 1000 * (float32(result.AddedLast30d) / float32(totalStars))
	}

	return result, nil
}

func (c *ClientGQL) GetAllStats(ctx context.Context, ghRepo string) (*RepoStats, error) {
	result := RepoStats{}

	ctx, span := tracer.Start(ctx, "fetch-repo-stats")
	defer span.End()

	span.SetAttributes(attribute.String("github.repo", ghRepo))

	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(repoSplit[0]),
		"name":  githubv4.String(repoSplit[1]),
	}

	type commit struct {
		Node struct {
			CommittedDate time.Time
		}
	}

	type release struct {
		Node struct {
			CreatedAt   time.Time
			PublishedAt time.Time
			Name        string
		}
	}

	var query struct {
		Repository struct {
			Description     string
			StargazerCount  int
			CreatedAt       time.Time
			PrimaryLanguage struct {
				Name string
			}
			ForkCount        int
			IsArchived       bool
			DiskUsage        int
			MentionableUsers struct {
				TotalCount int
			}
			OpenIssues struct {
				TotalCount int
			} `graphql:"issues(states: OPEN)"`
			DefaultBranchRef struct {
				Name   string
				Target struct {
					Commit struct {
						History struct {
							TotalCount int
							Edges      []commit
						} `graphql:"history(first: 1)"`
					} `graphql:"... on Commit"`
				}
			}
			Releases struct {
				TotalCount int
				Edges      []release
			} `graphql:"releases(first: 1)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	err := c.ghClient.Query(ctx, &query, variables)
	if err != nil {
		log.Printf("%v\n", err)
	}
	fmt.Println("Desc:", query.Repository.Description)
	fmt.Println("Total Commit:", query.Repository.DefaultBranchRef.Target.Commit.History.TotalCount)

	result.GHPath = ghRepo
	result.CreatedAt = query.Repository.CreatedAt
	result.Stars = query.Repository.StargazerCount
	result.DefaultBranch = query.Repository.DefaultBranchRef.Name
	result.Archived = query.Repository.IsArchived
	result.Forks = query.Repository.ForkCount
	result.OpenIssues = query.Repository.OpenIssues.TotalCount
	result.Language = query.Repository.PrimaryLanguage.Name
	result.Size = query.Repository.DiskUsage
	result.MentionableUsers = query.Repository.MentionableUsers.TotalCount

	commits := query.Repository.DefaultBranchRef.Target.Commit.History.Edges
	if len(commits) > 0 {
		result.LastCommitDate = commits[0].Node.CommittedDate
	}
	releases := query.Repository.Releases.Edges
	if len(releases) > 0 {
		result.LastReleaseDate = releases[0].Node.CreatedAt
	}

	result.StarsHistory, _ = c.getStarsHistory(ctx, repoSplit[0], repoSplit[1], result.Stars)

	if result.Language == "Go" {
		GetGoStats(ctx, c.restyClient, ghRepo, &result)
	}

	return &result, nil
}

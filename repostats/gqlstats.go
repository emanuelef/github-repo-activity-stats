package repostats

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

type ClientGQL struct {
	ghClient *githubv4.Client
}

func NewClientGQL(oauthClient *http.Client) *ClientGQL {
	ghClient := githubv4.NewClient(oauthClient)

	return &ClientGQL{ghClient: ghClient}
}

func (c *ClientGQL) GetAllStats(ghRepo string) (*RepoStats, error) {
	result := RepoStats{}

	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(repoSplit[0]),
		"name":  githubv4.String(repoSplit[1]),
	}

	var query struct {
		Repository struct {
			Description string
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	err := c.ghClient.Query(context.Background(), &query, variables)
	if err != nil {
		// Handle error.
	}
	fmt.Println("Desc:", query.Repository.Description)

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
		"owner":       githubv4.String(repoSplit[0]),
		"name":        githubv4.String(repoSplit[1]),
		"starsCursor": (*githubv4.String)(nil),
	}

	type starred struct {
		StarredAt time.Time
		Cursor    string
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
		err = c.ghClient.Query(context.Background(), &queryStars, variablesStars)
		if err != nil {
			// Handle error.
		}
		fmt.Println("Desc:", len(queryStars.Repository.Stargazers.Edges))

		res := queryStars.Repository.Stargazers.Edges

		// result.LastStarDate = res[0].StarredAt

		currentTime := time.Now()
		slices.Reverse(res) // order from most recent to least

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

	return &result, nil
}

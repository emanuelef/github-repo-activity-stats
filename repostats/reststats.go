package repostats

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
)

const (
	apiGHUrl = "https://api.github.com"
	rawGHUrl = "https://raw.githubusercontent.com"
)

type Client struct {
	restyClient *resty.Client
}

func NewClient(transport *http.RoundTripper) *Client {
	restyClient := resty.New()
	restyClient.SetTransport(*transport)

	return &Client{restyClient: restyClient}
}

func (c *Client) getStarsHistory(ghRepo string, totalStars int) (stats.StarsHistory, error) {
	// https://api.github.com/repos/temporalio/temporal/stargazers?per_page=100&page=80
	// Accept: application/vnd.github.star+json

	result := stats.StarsHistory{}

	res := [](map[string]any){}
	restyReq := c.restyClient.R().SetResult(&res).SetHeader("Accept", "application/vnd.github.star+json")
	apiGithubUrl := fmt.Sprintf("%s/repos/%s/stargazers", apiGHUrl, ghRepo)

	// The stargazer endpoint allows only to reach page 400, and a maximum 100 results per page
	// It also doesn't seem to support sorting in reverse order

	perPage := strconv.Itoa(100)
	page := (totalStars / 100) + 1

	for i := 0; i < 2; i++ {

		currentPage := page - i

		if currentPage < 0 {
			break
		}

		resp, err := restyReq.
			SetQueryParams(map[string]string{
				"page":     strconv.Itoa(currentPage),
				"per_page": perPage,
			}).Get(apiGithubUrl)

		if resp.StatusCode() == http.StatusUnprocessableEntity {
			log.Println("Request over limit")
		}

		if err != nil {
			log.Println(err)
		}

		if resp.IsSuccess() {
			if len(res) == 0 {
				log.Println("No stars")
				return result, nil
			}

			result.LastStarDate, _ = time.Parse(time.RFC3339, res[0]["starred_at"].(string))

			currentTime := time.Now()

			slices.Reverse(res)
			for _, star := range res {
				// "2023-08-23T15:06:33Z"
				dateString := star["starred_at"].(string)
				output, err := time.Parse(time.RFC3339, dateString)
				if err == nil {
					days := currentTime.Sub(output).Hours() / 24

					if days < 1 {
						result.AddedLast24H += 1
					}

					if days < 7 {
						result.AddedLast7d += 1
					}

					if days < 14 {
						result.AddedLast14d += 1
					}

					if days < 30 {
						result.AddedLast30d += 1
					}
				}
			}
		}
	}

	return result, nil
}

// Refactor this and most of the code
func (c *Client) GetAllStats(ghRepo string) (*stats.RepoStats, error) {
	res := make(map[string]any)
	restyReq := c.restyClient.R().SetResult(&res)

	apiGithubUrl := fmt.Sprintf("%s/repos/%s", apiGHUrl, ghRepo)

	resp, err := restyReq.Get(apiGithubUrl)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		log.Println("Error getting repo infos")
		return nil, fmt.Errorf("%s Error getting repo infos", resp.Status())
	}

	language, ok := res["language"].(string)
	if !ok {
		language = ""
	}

	result := stats.RepoStats{
		GHPath:        ghRepo,
		Stars:         int(res["stargazers_count"].(float64)),
		Size:          int(res["size"].(float64)),
		Language:      language,
		OpenIssues:    int(res["open_issues_count"].(float64)),
		Forks:         int(res["forks_count"].(float64)),
		Archived:      res["archived"].(bool),
		DefaultBranch: res["default_branch"].(string),
	}

	result.StarsHistory, _ = c.getStarsHistory(ghRepo, result.Stars)

	return &result, nil
}

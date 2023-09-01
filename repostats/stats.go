package repostats

import (
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/mod/modfile"
)

const (
	apiGHUrl = "https://api.github.com"
	rawGHUrl = "https://raw.githubusercontent.com"
)

type Client struct {
	restyClient *resty.Client
}

type GoRepo struct {
	GoVersion  string
	DirectDeps []string
}

type StarsHistory struct {
	AddedLast24H int
	AddedLast7d  int
	AddedLast14d int
	AddedLast30d int
	LastStarDate time.Time
}

func (sh StarsHistory) String() string {
	return fmt.Sprintf(`Last Star Date: %s
AddedLast24H: %d
AddedLast7d: %d
AddedLast14d: %d
AddedLast30d: %d`, sh.LastStarDate,
		sh.AddedLast24H,
		sh.AddedLast7d,
		sh.AddedLast14d,
		sh.AddedLast30d)
}

type RepoStats struct {
	GHPath        string
	Stars         int
	Size          int
	Language      string
	OpenIssues    int
	Forks         int
	Archived      bool
	DefaultBranch string
	StarsHistory
	GoRepo
}

func (rs RepoStats) String() string {
	return fmt.Sprintf(`
GH Repo: %s
Stars: %d
Size: %d
Language: %s
Open Issues: %d
Forks: %d
Archived: %t
Default Branch: %s
%s
Go version: %s
Go Direct dependencies: %d
	`, rs.GHPath,
		rs.Stars,
		rs.Size,
		rs.Language,
		rs.OpenIssues,
		rs.Forks,
		rs.Archived,
		rs.DefaultBranch,
		rs.StarsHistory,
		rs.GoVersion,
		len(rs.DirectDeps))
}

func NewClient(transport *http.RoundTripper) *Client {
	restyClient := resty.New()
	restyClient.SetTransport(*transport)

	return &Client{restyClient: restyClient}
}

func (c *Client) getStarsHistory(ghRepo string, totalStars int) (StarsHistory, error) {
	// https://api.github.com/repos/temporalio/temporal/stargazers?per_page=100&page=80
	// Accept: application/vnd.github.star+json

	result := StarsHistory{}

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
					days := currentTime.Sub(output).Hours()

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

func (c *Client) GetAllStats(ghRepo string) (*RepoStats, error) {
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

	result := RepoStats{
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

	// get go.mod file
	if result.Language == "Go" {
		goModUrl := fmt.Sprintf("%s/%s/%s/go.mod", rawGHUrl, ghRepo, result.DefaultBranch)
		resp, err := c.restyClient.R().Get(goModUrl)

		if err == nil {
			f, err := modfile.Parse("go.mod", resp.Body(), nil)
			if err != nil {
				return &result, nil
			}
			result.GoVersion = f.Go.Version

			var directDeps []string

			for _, req := range f.Require {
				// only direct dependencies
				if !req.Indirect {
					directDeps = append(directDeps, req.Mod.Path)
				}
			}

			result.DirectDeps = directDeps
		}
	}

	return &result, nil
}

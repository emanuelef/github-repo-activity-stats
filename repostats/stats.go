package repostats

import (
	"fmt"
	"net/http"

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

type RepoStats struct {
	Stars         float64
	Language      string
	OpenIssues    float64
	Forks         float64
	Archived      bool
	DefaultBranch string
	GoRepo
}

func (rs RepoStats) String() string {
	return fmt.Sprintf(`
Stars: %d
Language: %s
Open Issues: %d
Forks: %d
Archived: %t
Default Branch: %s
Go version: %s
Direct dependencies: %d
	`, int(rs.Stars),
		rs.Language,
		int(rs.OpenIssues),
		int(rs.Forks),
		rs.Archived,
		rs.DefaultBranch,
		rs.GoVersion,
		len(rs.DirectDeps))
}

func NewClient(transport *http.RoundTripper) *Client {
	restyClient := resty.New()
	restyClient.SetTransport(*transport)

	return &Client{restyClient: restyClient}
}

func (c *Client) GetAllStats(ghRepo string) (*RepoStats, error) {
	res := make(map[string]any)
	restyReq := c.restyClient.R().SetResult(&res)

	apiGithubUrl := fmt.Sprintf("%s/repos/%s", apiGHUrl, ghRepo)

	_, _ = restyReq.Get(apiGithubUrl)

	result := RepoStats{
		Stars:         res["stargazers_count"].(float64),
		Language:      res["language"].(string),
		OpenIssues:    res["open_issues_count"].(float64),
		Forks:         res["forks_count"].(float64),
		Archived:      res["archived"].(bool),
		DefaultBranch: res["default_branch"].(string),
	}

	// get go.mod file
	if result.Language == "Go" {
		goModUrl := fmt.Sprintf("%s/%s/%s/go.mod", rawGHUrl, ghRepo, result.DefaultBranch)
		resp, err := c.restyClient.R().Get(goModUrl)

		if err == nil {
			f, err := modfile.Parse("go.mod", resp.Body(), nil)
			if err != nil {
				return nil, err
			}
			result.GoVersion = f.Go.Version

			var directDeps []string

			for _, req := range f.Require {
				// only direct dependencies
				if !req.Indirect {
					// fmt.Printf("%s\n", req.Mod.Path)
					directDeps = append(directDeps, req.Mod.Path)
				}
			}

			result.DirectDeps = directDeps
		}
	}

	return &result, nil
}

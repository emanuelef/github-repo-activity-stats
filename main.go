package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/mod/modfile"
	"golang.org/x/oauth2"
)

// https://api.github.com/repos/jasonrudolph/keyboard

// https://docs.github.com/en/rest/activity/starring?apiVersion=2022-11-28#alternative-response-with-star-creation-timestamps
// https://docs.github.com/en/rest/metrics/statistics?apiVersion=2022-11-28
// https://api.github.com/repos/kubernetes/kubernetes/releases

// https://pkg.go.dev/golang.org/x/mod@v0.5.1/modfile#Require
// https://go.dev/play/p/XETDzMcTwS_S // Test mod parsing

const ghRepo = "kubernetes/kubernetes"

// const ghRepo = "keptn/keptn" // no root go.mod

type TokenSource struct {
	AccessToken string
}

func main() {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	restyClient := resty.New()
	restyClient.SetTransport(oauthClient.Transport)

	res := make(map[string]any)

	restyReq := restyClient.R().SetResult(&res)

	apiGithubUrl := fmt.Sprintf("https://api.github.com/repos/%s", ghRepo)

	_, _ = restyReq.Get(apiGithubUrl)

	fmt.Println("Stars:", res["stargazers_count"])
	fmt.Println("Language:", res["language"])
	fmt.Println("Open Issues:", res["open_issues_count"])
	fmt.Println("Forks:", res["forks_count"])
	fmt.Println("Archived:", res["archived"])
	fmt.Println("Default branch:", res["default_branch"])

	goModUrl := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/go.mod", ghRepo, res["default_branch"])
	resp, err := restyClient.R().Get(goModUrl)

	if err == nil {
		f, err := modfile.Parse("go.mod", resp.Body(), nil)
		if err != nil {
			panic(err)
		}
		fmt.Println(f.Go.Version)

		var directDeps []string

		for _, req := range f.Require {
			// only direct dependencies
			if !req.Indirect {
				// fmt.Printf("%s\n", req.Mod.Path)
				directDeps = append(directDeps, req.Mod.Path)
			}
		}

		fmt.Printf("Direct dependencies %d\n", len(directDeps))
	}

}

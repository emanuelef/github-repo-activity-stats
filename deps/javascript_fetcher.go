package deps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
)

type JavascriptDepsFetcher struct{}

type PackageInfo struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (gdf JavascriptDepsFetcher) Create() JavascriptDepsFetcher {
	return JavascriptDepsFetcher{}
}

func (gdf JavascriptDepsFetcher) GetDepsList(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) error {
	packageJsonTomUrl := fmt.Sprintf("%s/%s/%s/package.json", rawGHUrl, ghRepo, result.DefaultBranch)

	restyReq := restyClient.R()
	restyReq.SetContext(ctx)
	resp, err := restyReq.Get(packageJsonTomUrl)

	if err == nil {

		var pkgInfo PackageInfo
		err := json.Unmarshal(resp.Body(), &pkgInfo)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		var directDeps []string

		fmt.Println("Dependencies:")
		for dep := range pkgInfo.Dependencies {
			directDeps = append(directDeps, dep)
		}

		fmt.Println("\nDev Dependencies:")
		for dep := range pkgInfo.DevDependencies {
			directDeps = append(directDeps, dep)
		}

		result.DirectDeps = directDeps
	}

	return nil
}

package deps

import (
	"context"
	"fmt"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
	"golang.org/x/mod/modfile"
)

type GoDepsFetcher struct{}

func (gdf GoDepsFetcher) Create() GoDepsFetcher {
	return GoDepsFetcher{}
}

func (gdf GoDepsFetcher) GetDepsList(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) error {
	goModUrl := fmt.Sprintf("%s/%s/%s/go.mod", rawGHUrl, ghRepo, result.DefaultBranch)

	restyReq := restyClient.R()
	restyReq.SetContext(ctx)
	resp, err := restyReq.Get(goModUrl)

	if err == nil {
		f, err := modfile.Parse("go.mod", resp.Body(), nil)
		if err != nil {
			return nil
		}

		if f.Go != nil {
			result.GoVersion = f.Go.Version
		}

		var directDeps []string

		for _, req := range f.Require {
			// only direct dependencies
			if !req.Indirect {
				directDeps = append(directDeps, req.Mod.Path)
			}
		}

		result.DirectDeps = directDeps
	}

	return nil
}

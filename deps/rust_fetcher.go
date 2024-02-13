package deps

import (
	"context"
	"fmt"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
	"github.com/pelletier/go-toml"
)

type RustDepsFetcher struct{}

func (gdf RustDepsFetcher) Create() RustDepsFetcher {
	return RustDepsFetcher{}
}

func (gdf RustDepsFetcher) GetDepsList(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) error {
	cargoTomUrl := fmt.Sprintf("%s/%s/%s/Cargo.toml", rawGHUrl, ghRepo, result.DefaultBranch)

	restyReq := restyClient.R()
	restyReq.SetContext(ctx)
	resp, err := restyReq.Get(cargoTomUrl)

	if err == nil {
		cfg, err := toml.Load(string(resp.Body()))
		if err != nil {
			return err
		}

		var directDeps []string

		// Check [dependencies] section
		if depSection, ok := cfg.Get("dependencies").(*toml.Tree); ok {
			for name := range depSection.ToMap() {
				directDeps = append(directDeps, name)
			}
		}

		// Check workspace.dependencies section
		if workspace, ok := cfg.Get("workspace").(*toml.Tree); ok {
			if workspaceDeps, ok := workspace.Get("dependencies").(*toml.Tree); ok {
				for name := range workspaceDeps.ToMap() {
					directDeps = append(directDeps, name)
				}
			}
		}

		// Remove duplicates by converting to a map and then back to a slice
		uniqueDirectDeps := make(map[string]struct{})
		for _, dep := range directDeps {
			uniqueDirectDeps[dep] = struct{}{}
		}

		result.DirectDeps = make([]string, 0, len(uniqueDirectDeps))
		for dep := range uniqueDirectDeps {
			result.DirectDeps = append(result.DirectDeps, dep)
		}
	}

	return nil
}

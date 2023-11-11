package deps

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
	"github.com/pelletier/go-toml"
)

type PythonDepsFetcher struct{}

func (gdf PythonDepsFetcher) Create() PythonDepsFetcher {
	return PythonDepsFetcher{}
}

func (gdf PythonDepsFetcher) GetDepsList(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) error {
	requirementsUrl := fmt.Sprintf("%s/%s/%s/requirements.txt", rawGHUrl, ghRepo, result.DefaultBranch)

	restyReq := restyClient.R()
	restyReq.SetContext(ctx)

	var directDeps []string

	resp, err := restyReq.Get(requirementsUrl)

	if resp.IsSuccess() && err == nil {
		reader := bytes.NewReader([]byte(resp.Body()))
		scanner := bufio.NewScanner(reader)

		for scanner.Scan() {
			line := scanner.Text()
			// Exclude lines starting with '#', which are comments
			if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-e") {
				// Use a regular expression to match the package name and version
				re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)[^a-zA-Z0-9_-]`)
				match := re.FindStringSubmatch(line)
				if len(match) >= 2 {
					directDeps = append(directDeps, match[1])
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	poetryUrl := fmt.Sprintf("%s/%s/%s/pyproject.toml", rawGHUrl, ghRepo, result.DefaultBranch)

	resp, err = restyReq.Get(poetryUrl)

	if resp.IsSuccess() && err == nil {
		cfg, err := toml.Load(string(resp.Body()))
		if err != nil {
			return err
		}

		if depSection, ok := cfg.Get("tool.poetry.dependencies").(*toml.Tree); ok {
			for name := range depSection.ToMap() {
				directDeps = append(directDeps, name)
			}
		}
	}

	result.DirectDeps = directDeps

	return nil
}

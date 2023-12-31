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
					directDeps = append(directDeps, strings.ToLower(match[1]))
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
				directDeps = append(directDeps, strings.ToLower(name))
			}
		}
	}

	setupUrl := fmt.Sprintf("%s/%s/%s/setup.py", rawGHUrl, ghRepo, result.DefaultBranch)

	resp, err = restyReq.Get(setupUrl)

	if resp.IsSuccess() && err == nil {
		reader := bytes.NewReader([]byte(resp.Body()))
		scanner := bufio.NewScanner(reader)

		dependencyRegex := regexp.MustCompile(`^\s*'([a-zA-Z0-9_-]+)'(?:[,)]|$)`)
		for scanner.Scan() {
			line := scanner.Text()

			// Match the line against the regex
			match := dependencyRegex.FindStringSubmatch(line)
			if len(match) >= 2 {
				dependencyName := match[1]
				directDeps = append(directDeps, strings.ToLower(dependencyName))
			}
		}

		// Check for errors during scanning
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	pipfileUrl := fmt.Sprintf("%s/%s/%s/Pipfile", rawGHUrl, ghRepo, result.DefaultBranch)

	resp, err = restyReq.Get(pipfileUrl)

	if resp.IsSuccess() && err == nil {
		reader := bytes.NewReader([]byte(resp.Body()))
		scanner := bufio.NewScanner(reader)

		dependencyRegex := regexp.MustCompile(`\b([\w\d_-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s]+))`)
		currentSection := ""

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Check if the line contains a section header
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				currentSection = line[1 : len(line)-1]
				continue
			}

			// Check if the line contains a dependency
			matches := dependencyRegex.FindStringSubmatch(line)
			if len(matches) >= 2 && (currentSection == "dev-packages" || currentSection == "packages") {
				dependencyName := matches[1]
				directDeps = append(directDeps, strings.ToLower(dependencyName))
			}
		}

		// Check for errors during scanning
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	result.DirectDeps = directDeps

	return nil
}

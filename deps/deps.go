package deps

import (
	"context"
	"strings"

	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
)

const (
	apiGHUrl = "https://api.github.com"
	rawGHUrl = "https://raw.githubusercontent.com"
)

type DepsFetcher interface {
	GetDepsList(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) error
}

func CreateFetcher(lang string) DepsFetcher {
	switch strings.ToLower(lang) {
	case "go":
		return GoDepsFetcher{}
	case "rust":
		return RustDepsFetcher{}
	default:
		return nil
	}
}

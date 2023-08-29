package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	CNCFProjectsYamlUrl = "https://raw.githubusercontent.com/cncf/devstats/master/projects.yaml"
)

type T struct {
	Projects struct {
		RenamedC int `yaml:"c"`
		Project  map[string]any
	}
}

func main() {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClient(&oauthClient.Transport)

	restyClient := resty.New()
	resp, err := restyClient.R().Get(CNCFProjectsYamlUrl)
	if err == nil {
		m := make(map[any]any)

		err = yaml.Unmarshal([]byte(resp.Body()), &m)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		for key, val := range m["projects"].(map[string]any) {
			p := val.(map[string]any)
			fmt.Printf("%s %s %s\n", key, p["main_repo"], p["status"])
			if p["status"].(string) != "-" {
				result, _ := client.GetAllStats(p["main_repo"].(string))
				fmt.Println(result)
			}
		}
	}
}

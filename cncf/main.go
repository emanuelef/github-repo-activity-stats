package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v3"
)

const (
	CNCFProjectsYamlUrl = "https://raw.githubusercontent.com/cncf/devstats/master/projects.yaml"
)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func writeGoDepsMapFile(deps map[string]int) {
	currentTime := time.Now()
	outputFile, err := os.Create(fmt.Sprintf("dep-repo-%s.csv", getEnv("FILE_SUFFIX", (currentTime.Format("02-01-2006")))))
	if err != nil {
		log.Fatal(err)
	}

	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	headerRow := []string{
		"dep", "go_cncf_repos_using_dep",
	}

	csvWriter.Write(headerRow)

	for k, v := range deps {
		if v > 20 {
			csvWriter.Write([]string{
				k,
				fmt.Sprintf("%d", v),
			})
		}
	}
}

func main() {
	var mutex sync.Mutex
	sem := semaphore.NewWeighted(100)
	var wg sync.WaitGroup

	currentTime := time.Now()
	outputFile, err := os.Create(fmt.Sprintf("analysis-%s.csv", getEnv("FILE_SUFFIX", currentTime.Format("02-01-2006"))))
	if err != nil {
		log.Fatal(err)
	}

	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	headerRow := []string{
		"repo", "stars", "new-stars-last-30d",
		"language",
		"archived", "dependencies",
		"status",
		"start_date", "join_date",
	}

	csvWriter.Write(headerRow)

	depsUse := map[string]int{}

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

		ctx := context.Background()

		for key, val := range m["projects"].(map[string]any) {
			wg.Add(1)

			go func(key string, val any) {
				sem.Acquire(ctx, 1)
				defer sem.Release(1)
				defer wg.Done()
				p := val.(map[string]any)
				fmt.Printf("%s %s %s\n", key, p["main_repo"], p["status"])
				if p["status"].(string) != "-" {
					result, err := client.GetAllStats(p["main_repo"].(string))

					if err != nil {
						log.Fatalf("Error getting all stats %v", err)
					}

					fmt.Println(result)

					mutex.Lock()
					csvWriter.Write([]string{
						fmt.Sprintf("%s", p["main_repo"]),
						fmt.Sprintf("%d", result.Stars),
						fmt.Sprintf("%d", result.AddedLast30d),
						result.Language,
						fmt.Sprintf("%t", result.Archived),
						fmt.Sprintf("%d", len(result.DirectDeps)),
						fmt.Sprintf("%s", p["status"]),
						fmt.Sprintf("%s", p["start_date"]),
						fmt.Sprintf("%s", p["join_date"]),
					})

					if len(result.DirectDeps) > 0 {
						for _, dep := range result.DirectDeps {
							depsUse[dep] += 1
						}
					}
					mutex.Unlock()
				}
			}(key, val)
		}
		wg.Wait()
		writeGoDepsMapFile(depsUse)
	}

	elapsed := time.Since(currentTime)
	log.Printf("Took %s\n", elapsed)
}

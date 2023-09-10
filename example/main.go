package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/sync/semaphore"
)

func main() {
	var mutex sync.Mutex
	sem := semaphore.NewWeighted(10)
	var wg sync.WaitGroup

	starsHistory := map[string][]repostats.StarsPerDay{}

	ctx := context.Background()

	currentTime := time.Now()
	outputFile, err := os.Create(fmt.Sprintf("analysis-latest.csv"))
	if err != nil {
		log.Fatal(err)
	}

	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	headerRow := []string{
		"repo", "stars", "new-stars-last-30d", "new-stars-last-14d",
		"new-stars-last-7d", "new-stars-last-24H", "stars-per-mille-0d",
		"language",
		"archived", "dependencies",
	}

	csvWriter.Write(headerRow)

	depsUse := map[string]int{}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClientGQL(oauthClient)

	file, err := os.Open("repos.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		mainRepo := scanner.Text()
		fmt.Println(mainRepo)

		wg.Add(1)

		go func() {
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer wg.Done()
			result, err := client.GetAllStats(ctx, mainRepo)
			if err != nil {
				log.Fatalf("Error getting all stats %v", err)
			}

			fmt.Println(result)

			mutex.Lock()
			csvWriter.Write([]string{
				fmt.Sprintf("%s", mainRepo),
				fmt.Sprintf("%d", result.Stars),
				fmt.Sprintf("%d", result.AddedLast30d),
				fmt.Sprintf("%d", result.AddedLast14d),
				fmt.Sprintf("%d", result.AddedLast7d),
				fmt.Sprintf("%d", result.AddedLast24H),
				fmt.Sprintf("%.3f", result.AddedPerMille30d),
				result.Language,
				fmt.Sprintf("%t", result.Archived),
				fmt.Sprintf("%d", len(result.DirectDeps)),
			})

			if len(result.DirectDeps) > 0 {
				for _, dep := range result.DirectDeps {
					depsUse[dep] += 1
				}
			}

			starsHistory[mainRepo] = result.StarsTimeline

			mutex.Unlock()
		}()
	}

	wg.Wait()

	jsonData, _ := json.MarshalIndent(starsHistory, "", " ")
	_ = os.WriteFile("stars-history-30d.json", jsonData, 0o644)

	repostats.WriteStarsHistoryCSV("stars-k8s-latest.csv", starsHistory["kubernetes/kubernetes"])

	elapsed := time.Since(currentTime)
	log.Printf("Took %s\n", elapsed)
}

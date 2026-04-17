package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/repostats"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

func main() {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	clientGQL := repostats.NewClientGQL(oauthClient)
	ctx := context.Background()

	yesterday := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -1)
	fmt.Printf("=== New Repos Count Comparison ===\n")
	fmt.Printf("Date: %s\n\n", yesterday.Format("2006-01-02"))

	newReposNoForks, err := clientGQL.GetNewReposCountHistory(ctx, yesterday, yesterday, false, nil)
	if err != nil {
		fmt.Printf("Error (no forks): %v\n", err)
		return
	}
	if len(newReposNoForks) > 0 {
		fmt.Printf("Without forks: %d repos\n", newReposNoForks[0].Count)
	}

	newReposWithForks, err := clientGQL.GetNewReposCountHistory(ctx, yesterday, yesterday, true, nil)
	if err != nil {
		fmt.Printf("Error (with forks): %v\n", err)
		return
	}
	if len(newReposWithForks) > 0 {
		fmt.Printf("With forks:    %d repos\n", newReposWithForks[0].Count)
	}

	if len(newReposNoForks) > 0 && len(newReposWithForks) > 0 {
		diff := newReposWithForks[0].Count - newReposNoForks[0].Count
		fmt.Printf("\nDifference (forks only): %d repos\n", diff)
		fmt.Printf("Forks as %% of total: %.1f%%\n", float64(diff)/float64(newReposWithForks[0].Count)*100)
	}
}

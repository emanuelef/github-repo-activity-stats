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

	endDate := time.Now().UTC().Truncate(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -13) // 14 days including today

	fmt.Printf("=== New Public PRs opened per day (%s to %s) ===\n",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	newPRs, err := clientGQL.GetNewPRsCountHistory(ctx, startDate, endDate, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for _, day := range newPRs {
		fmt.Printf("  %s: %d PRs (cumulative: %d)\n",
			time.Time(day.Day).Format("2006-01-02"),
			day.Count,
			day.TotalSeen)
	}

	if len(newPRs) > 0 {
		fmt.Printf("\nTotal PRs opened in the last 14 days: %d\n", newPRs[len(newPRs)-1].TotalSeen)
	}
}

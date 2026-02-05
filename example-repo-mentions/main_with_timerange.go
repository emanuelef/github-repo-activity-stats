package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

func exampleWithTimeRange() {
	// Get GitHub Personal Access Token from environment
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClientGQL(oauthClient)

	// Repository to search for mentions
	targetRepo := "voltagent/voltagent"

	// Get mentions from the last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // 30 days ago

	limit := 50

	fmt.Printf("Searching for mentions of %s in the last 30 days...\n", targetRepo)
	fmt.Printf("Date range: %s to %s\n\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Get all mentions with time range
	result, err := client.GetRepoMentionsWithTimeRange(context.Background(), targetRepo, limit, &startDate, &endDate)
	if err != nil {
		log.Fatalf("Error getting repo mentions: %v", err)
	}

	// Print summary
	fmt.Printf("Total mentions found in last 30 days: %d\n", result.TotalMentions)
	fmt.Printf("  - Issues: %d\n", result.IssuesCount)
	fmt.Printf("  - Pull Requests: %d\n", result.PullRequestsCount)
	fmt.Printf("  - Discussions: %d\n", result.DiscussionsCount)
	fmt.Println()

	// Print first 10 mentions (sorted by most recent)
	fmt.Println("Most recent mentions:")
	fmt.Println("=====================")
	for i, mention := range result.Mentions {
		if i >= 10 {
			break
		}
		daysAgo := int(time.Since(mention.CreatedAt).Hours() / 24)
		fmt.Printf("\n[%s] %s\n", mention.Type, mention.Title)
		fmt.Printf("Repository: %s\n", mention.Repository)
		fmt.Printf("Author: %s\n", mention.Author)
		fmt.Printf("State: %s\n", mention.State)
		fmt.Printf("URL: %s\n", mention.URL)
		fmt.Printf("Created: %s (%d days ago)\n", mention.CreatedAt.Format("2006-01-02 15:04:05"), daysAgo)
		if len(mention.Body) > 0 {
			fmt.Printf("Body preview: %s\n", mention.Body)
		}
	}

	// Save full results to JSON file
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	outputFile := "repo-mentions-30days.json"
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	fmt.Printf("\nFull results saved to %s\n", outputFile)
}

func exampleCustomDateRange() {
	// Get GitHub Personal Access Token from environment
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClientGQL(oauthClient)

	// Repository to search for mentions
	targetRepo := "kubernetes/kubernetes"

	// Custom date range: January 2024
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	limit := 100

	fmt.Printf("\n\n=== Custom Date Range Example ===\n")
	fmt.Printf("Searching for mentions of %s in January 2024...\n", targetRepo)

	// Get all mentions with time range
	result, err := client.GetRepoMentionsWithTimeRange(context.Background(), targetRepo, limit, &startDate, &endDate)
	if err != nil {
		log.Fatalf("Error getting repo mentions: %v", err)
	}

	// Print summary
	fmt.Printf("\nTotal mentions found in January 2024: %d\n", result.TotalMentions)
	fmt.Printf("  - Issues: %d\n", result.IssuesCount)
	fmt.Printf("  - Pull Requests: %d\n", result.PullRequestsCount)
	fmt.Printf("  - Discussions: %d\n", result.DiscussionsCount)
}

func exampleRESTAPIWithTimeRange() {
	token := os.Getenv("PAT")
	if token == "" {
		log.Fatal("PAT environment variable is required")
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClient(&oauthClient.Transport)

	targetRepo := "kubernetes/kubernetes"

	// Last 7 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	fmt.Printf("\n\n=== REST API with Time Range ===\n")
	fmt.Printf("Searching for mentions of %s in the last 7 days...\n", targetRepo)

	result, err := client.GetRepoMentionsRESTWithTimeRange(targetRepo, 30, &startDate, &endDate)
	if err != nil {
		log.Fatalf("Error getting repo mentions: %v", err)
	}

	fmt.Printf("\nMentions in last 7 days: %d\n", result.TotalMentions)
	fmt.Printf("  - Issues: %d\n", result.IssuesCount)
	fmt.Printf("  - Pull Requests: %d\n", result.PullRequestsCount)

	// Show most recent 5
	fmt.Println("\nMost recent 5:")
	for i, mention := range result.Mentions {
		if i >= 5 {
			break
		}
		daysAgo := int(time.Since(mention.CreatedAt).Hours() / 24)
		fmt.Printf("%d. [%s] %s (%d days ago)\n", i+1, mention.Type, mention.Title, daysAgo)
	}
}

func main() {
	// Run all examples
	exampleWithTimeRange()
	exampleCustomDateRange()
	exampleRESTAPIWithTimeRange()
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

func main() {
	// Get GitHub Personal Access Token from environment
	token := os.Getenv("PAT")
	if token == "" {
		log.Fatal("PAT environment variable is required")
	}

	// Create authenticated HTTP client
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)

	// Create REST API client
	client := repostats.NewClient(&oauthClient.Transport)

	// Repository to search for mentions
	targetRepo := "kubernetes/kubernetes"

	fmt.Printf("=== REST API Version ===\n")
	fmt.Printf("Searching for mentions of %s across GitHub...\n\n", targetRepo)

	// First, get a quick summary
	fmt.Println("Getting summary...")
	summary, err := client.GetRepoMentionsSummaryREST(targetRepo)
	if err != nil {
		log.Fatalf("Error getting summary: %v", err)
	}

	fmt.Printf("Summary of mentions:\n")
	fmt.Printf("  - Total: %d\n", summary["total"])
	fmt.Printf("  - Issues: %d\n", summary["issues"])
	fmt.Printf("  - Pull Requests: %d\n", summary["pull_requests"])
	fmt.Println()

	// Get detailed results (limit to 30 per type to avoid rate limiting)
	fmt.Println("Fetching detailed results...")
	result, err := client.GetRepoMentionsREST(targetRepo, 30)
	if err != nil {
		log.Fatalf("Error getting repo mentions: %v", err)
	}

	// Print results summary
	fmt.Printf("\nDetailed results retrieved:\n")
	fmt.Printf("  - Issues: %d\n", result.IssuesCount)
	fmt.Printf("  - Pull Requests: %d\n", result.PullRequestsCount)
	fmt.Printf("  - Total fetched: %d\n", result.TotalMentions)
	fmt.Println()

	// Print first 10 mentions
	fmt.Println("Recent mentions:")
	fmt.Println("================")
	for i, mention := range result.Mentions {
		if i >= 10 {
			break
		}
		fmt.Printf("\n[%s] %s\n", mention.Type, mention.Title)
		fmt.Printf("Repository: %s\n", mention.Repository)
		fmt.Printf("Author: %s\n", mention.Author)
		fmt.Printf("State: %s\n", mention.State)
		fmt.Printf("URL: %s\n", mention.URL)
		fmt.Printf("Created: %s\n", mention.CreatedAt.Format("2006-01-02 15:04:05"))
		if len(mention.Body) > 0 {
			fmt.Printf("Body preview: %s\n", mention.Body)
		}
	}

	// Save full results to JSON file
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	outputFile := "repo-mentions-rest.json"
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	fmt.Printf("\nFull results saved to %s\n", outputFile)
}

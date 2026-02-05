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
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClientGQL(oauthClient)

	// Repository to search for mentions
	targetRepo := "voltagent/voltagent"

	// Limit number of results per type (issues, PRs, discussions)
	limit := 50

	fmt.Printf("Searching for mentions of %s across GitHub...\n\n", targetRepo)

	// Get all mentions
	result, err := client.GetRepoMentions(context.Background(), targetRepo, limit)
	if err != nil {
		log.Fatalf("Error getting repo mentions: %v", err)
	}

	// Print summary
	fmt.Printf("Total mentions found: %d\n", result.TotalMentions)
	fmt.Printf("  - Issues: %d\n", result.IssuesCount)
	fmt.Printf("  - Pull Requests: %d\n", result.PullRequestsCount)
	fmt.Printf("  - Discussions: %d\n", result.DiscussionsCount)
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

	outputFile := "repo-mentions.json"
	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	fmt.Printf("\nFull results saved to %s\n", outputFile)
}

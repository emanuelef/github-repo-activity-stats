package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/repostats"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

const ghRepo = "kubernetes/kubernetes"

func main() {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	clientGQL := repostats.NewClientGQL(oauthClient)
	currentTime := time.Now()
	result, _ := clientGQL.GetAllStats(ctx, ghRepo)
	fmt.Println(result)

	allStars, _ := clientGQL.GetAllStarsHistory(ctx, ghRepo, result.CreatedAt)
	repostats.WriteStarsHistoryCSV("all-stars-k8s.csv", allStars)

	elapsed := time.Since(currentTime)
	log.Printf("Took %s\n", elapsed)
}

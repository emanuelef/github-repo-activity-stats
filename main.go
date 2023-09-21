package main

import (
	"context"
	"fmt"
	"os"

	"github.com/emanuelef/github-repo-activity-stats/repostats"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

// https://api.github.com/repos/jasonrudolph/keyboard

// https://docs.github.com/en/rest/activity/starring?apiVersion=2022-11-28#alternative-response-with-star-creation-timestamps
// https://docs.github.com/en/rest/metrics/statistics?apiVersion=2022-11-28
// https://api.github.com/repos/kubernetes/kubernetes/releases

// https://pkg.go.dev/golang.org/x/mod@v0.5.1/modfile#Require
// https://go.dev/play/p/XETDzMcTwS_S // Test mod parsing

const ghRepo = "kubernetes/kubernetes"

// const ghRepo = "keptn/keptn" // no root go.mod

func main() {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("PAT")},
	)

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := repostats.NewClient(&oauthClient.Transport)

	result, _ := client.GetAllStats(ghRepo)
	fmt.Println(result)
	/*
		result, _ = client.GetAllStats("temporalio/temporal")
		fmt.Println(result)

		result, _ = client.GetAllStats("envoyproxy/envoy")
		fmt.Println(result)

		result, _ = client.GetAllStats("fluent/fluentd")
		fmt.Println(result)

		result, _ = client.GetAllStats("keptn/keptn")
		fmt.Println(result)
	*/

	result, _ = client.GetAllStats("emanuelef/github-repo-activity-stats")
	fmt.Println(result)

	result, _ = client.GetAllStats("ceccopierangiolieugenio/pyTermTk")
	fmt.Println(result)

	clientGQL := repostats.NewClientGQL(oauthClient)

	ctx := context.Background()

	result, _ = clientGQL.GetAllStats(ctx, "ceccopierangiolieugenio/pyTermTk")
	fmt.Println(result)
	allStars, _ := clientGQL.GetAllStarsHistory(ctx, "ceccopierangiolieugenio/pyTermTk", result.CreatedAt)
	fmt.Println(allStars)

	result, _ = clientGQL.GetAllStats(ctx, ghRepo)
	fmt.Println(result)
	// fmt.Println(result.StarsHistory.StarsTimeline)

	result, _ = clientGQL.GetAllStats(ctx, "istio/istio")
	fmt.Println(result)

	resultRateLimit, _ := clientGQL.GetCurrentLimits(ctx)
	fmt.Printf("Limit: %d, Remaining: %d\n", resultRateLimit.Limit, resultRateLimit.Remaining)
}

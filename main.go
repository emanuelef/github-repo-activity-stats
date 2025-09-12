package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

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

	// Get top stargazers by followers for daily-stars-explorer
	fmt.Println("\nTop 10 stargazers by followers for metalbear-co/mirrord:")
	topStargazers, err := clientGQL.GetTopStargazersByFollowers(ctx, "metalbear-co/mirrord", 10)
	if err != nil {
		fmt.Printf("Error getting top stargazers: %v\n", err)
	} else {
		for i, user := range topStargazers {
			fmt.Printf("%d. %s - %d followers (starred at %s)\n",
				i+1, user.Login, user.FollowerCount, user.StarredAt.Format("2006-01-02"))
		}
	}

	resultRecent, _ := clientGQL.GetRecentStarsHistoryTwoWays(ctx, "langflow-ai/langflow", 10, nil)
	for _, val := range resultRecent {
		fmt.Println("Date:", time.Time(val.Day), "Stars:", val.Stars, "Total Stars:", val.TotalStars)
	}

	allCommits, defaultBranch, _ := clientGQL.GetAllCommitsHistory(ctx, "ceccopierangiolieugenio/pyTermTk", nil)
	fmt.Println(time.Time(allCommits[len(allCommits)-1].Day))
	fmt.Println(defaultBranch)

	allPRs, _ := clientGQL.GetAllPRsHistory(ctx, "temporalio/temporal", nil)
	fmt.Println(time.Time(allPRs[len(allPRs)-1].Day))

	allIssues, _ := clientGQL.GetAllIssuesHistory(ctx, "temporalio/temporal", nil)
	fmt.Println(time.Time(allIssues[len(allIssues)-1].Day))

	allForks, _ := clientGQL.GetAllForksHistory(ctx, "ceccopierangiolieugenio/pyTermTk", nil)
	fmt.Println(time.Time(allForks[len(allForks)-1].Day))

	allContributors, _ := clientGQL.GetNewContributorsHistory(ctx, "temporalio/temporal", nil)
	fmt.Println(time.Time(allContributors[len(allContributors)-1].Day))

	// Test our new releases feed function
	allReleasesFeed, _ := clientGQL.GetAllReleasesFeed(ctx, "kubernetes/kubernetes")
	if len(allReleasesFeed) > 0 {
		fmt.Println("Latest release date:", allReleasesFeed[0].PublishedAt)
		fmt.Println("Total releases:", allReleasesFeed[0].TotalReleases)
		fmt.Println("Latest release name:", allReleasesFeed[0].Name)
		fmt.Println("Latest release tag:", allReleasesFeed[0].TagName)
	}

	//
	result, _ = clientGQL.GetAllStats(ctx, "kubewarden/kubewarden-controller")
	fmt.Println(result)

	// no commits or stars in the last 30d, at least last time I checked
	result, _ = clientGQL.GetAllStats(ctx, "mengzhuo/cookiestxt")
	fmt.Println(result)

	result, _ = clientGQL.GetAllStats(ctx, "ceccopierangiolieugenio/pyTermTk")
	fmt.Println(result)

	// repoTest := "kubernetes/kubernetes"
	// repoTest := "agnivade/levenshtein"
	repoTest := "mattn/go-colorable"

	result, _ = clientGQL.GetAllStats(ctx, repoTest)
	fmt.Println(result)

	allStars, _ := clientGQL.GetAllStarsHistory(ctx, repoTest, result.CreatedAt, nil)
	// fmt.Println(allStars)
	fmt.Println(time.Time(allStars[len(allStars)-1].Day))

	allStars2, _ := clientGQL.GetAllStarsHistoryTwoWays(ctx, repoTest, nil)
	fmt.Println("Equal Stars", slices.Equal(allStars, allStars2))

	for i, val := range allStars {
		if val.Stars != allStars2[i].Stars || val.TotalStars != allStars2[i].TotalStars {
			fmt.Println("Not Equal", i, val.Stars, allStars2[i].Stars, val.TotalStars, allStars2[i].TotalStars)
		}
	}

	result, _ = clientGQL.GetAllStats(ctx, "dghubble/gologin")
	fmt.Println(result)

	allStars, _ = clientGQL.GetAllStarsHistory(ctx, "dghubble/gologin", result.CreatedAt, nil)
	// fmt.Println(allStars)
	fmt.Println(time.Time(allStars[len(allStars)-1].Day))

	result, _ = clientGQL.GetAllStats(ctx, ghRepo)
	fmt.Println(result)
	// fmt.Println(result.StarsHistory.StarsTimeline)

	/*
		result, _ = clientGQL.GetAllStats(ctx, "istio/istio")
		fmt.Println(result)
	*/

	resultRateLimit, _ := clientGQL.GetCurrentLimits(ctx)
	fmt.Printf("Limit: %d, Remaining: %d\n", resultRateLimit.Limit, resultRateLimit.Remaining)

	starsCount, createdAt, _ := clientGQL.GetTotalStars(ctx, "ceccopierangiolieugenio/pyTermTk")
	fmt.Println(starsCount, createdAt)
	/*
		result, _ = clientGQL.GetAllStats(ctx, "surrealdb/surrealdb")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "fractalide/fractalide")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "chaosprint/glicol")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "gen2brain/malgo")
		fmt.Println(result)

		// package.json
		result, _ = clientGQL.GetAllStats(ctx, "winstonjs/winston")
		fmt.Println(result)

		// requirements.txt
		result, _ = clientGQL.GetAllStats(ctx, "encode/uvicorn")
		fmt.Println(result)

		// poetry pyproject.toml
		result, _ = clientGQL.GetAllStats(ctx, "copier-org/copier")
		fmt.Println(result)

		// setup.py
		result, _ = clientGQL.GetAllStats(ctx, "httpie/cli")
		fmt.Println(result)

		// pipenv Pipfile
		result, _ = clientGQL.GetAllStats(ctx, "zappa/Zappa")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "confluentinc/confluent-kafka-go")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "1set/gut")
		fmt.Println(result)

		result, _ = clientGQL.GetAllStats(ctx, "google/google-api-go-client")
		fmt.Println(result)

	*/

	result, _ = clientGQL.GetAllStats(ctx, "influxdata/influxdb")
	fmt.Println(result)

	maxPeriods, maxPeaks, _ := repostats.FindMaxConsecutivePeriods(result.StarsHistory.StarsTimeline, 10)
	fmt.Printf("%v %v \n", maxPeriods, maxPeaks)

	last7DaysStars := repostats.NewStarsLastDays(result.StarsHistory.StarsTimeline, 7)
	fmt.Println(last7DaysStars)
}

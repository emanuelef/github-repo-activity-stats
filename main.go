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

	resultRecent, _ := clientGQL.GetRecentStarsHistoryTwoWays(ctx, "langflow-ai/langflow", 10, nil)
	for _, val := range resultRecent {
		fmt.Println("Date:", time.Time(val.Day), "Stars:", val.Stars, "Total Stars:", val.TotalStars)
	}

	// Test new hourly range methods
	fmt.Println("\n=== Testing Hourly Stars History ===")

	// Test 1: Fetch last 5 hours only
	fiveHoursAgo := time.Now().Add(-5 * time.Hour)
	hourlyStars, _ := clientGQL.GetRecentStarsHistoryByHourSince(ctx, "langflow-ai/langflow", fiveHoursAgo, nil)
	fmt.Printf("\nStars in last 5 hours (%d hours fetched):\n", len(hourlyStars))
	for _, val := range hourlyStars {
		fmt.Printf("Hour: %s, Stars: %d, Total: %d\n", val.Hour.Format("2006-01-02 15:04"), val.Stars, val.TotalStars)
	}

	// Test 2: Fetch specific time range (e.g., 10 hours starting from 12 hours ago)
	startTime := time.Now().Add(-12 * time.Hour)
	endTime := time.Now().Add(-2 * time.Hour)
	hourlyStarsRange, _ := clientGQL.GetRecentStarsHistoryByHourRange(ctx, "langflow-ai/langflow", startTime, endTime, nil)
	fmt.Printf("\nStars in custom range (%s to %s):\n", startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"))
	totalInRange := 0
	for _, val := range hourlyStarsRange {
		totalInRange += val.Stars
	}
	fmt.Printf("Total hours: %d, Total stars in range: %d\n", len(hourlyStarsRange), totalInRange)

	// Test 3: Compare day-based vs hour-based for last 2 days
	hourlyStars2d, _ := clientGQL.GetRecentStarsHistoryByHour(ctx, "langflow-ai/langflow", 2, nil)
	fmt.Printf("\nStars in last 2 days (hourly): %d hours of data\n", len(hourlyStars2d))

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

	// Test new repos count history
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	newRepos, _ := clientGQL.GetNewReposCountHistory(ctx, startDate, endDate, false, nil)
	if len(newRepos) > 0 {
		fmt.Println("\nNew Public Repos (excluding forks) - January 2025:")
		fmt.Printf("First day (%s): %d repos\n", time.Time(newRepos[0].Day).Format("2006-01-02"), newRepos[0].Count)
		fmt.Printf("Last day (%s): %d repos\n", time.Time(newRepos[len(newRepos)-1].Day).Format("2006-01-02"), newRepos[len(newRepos)-1].Count)
		fmt.Printf("Total new repos in period: %d\n", newRepos[len(newRepos)-1].TotalSeen)
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

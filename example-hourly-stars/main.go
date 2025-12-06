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

	// Example 1: Get hourly stars for the last 7 days for a popular repo
	fmt.Println("=== Example 1: Hourly stars for langflow-ai/langflow (last 7 days) ===")
	hourlyStars7d, err := clientGQL.GetRecentStarsHistoryByHour(ctx, "langflow-ai/langflow", 7, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Total hours tracked: %d\n", len(hourlyStars7d))
		
		// Show first few hours
		fmt.Println("\nFirst 5 hours:")
		for i := 0; i < 5 && i < len(hourlyStars7d); i++ {
			fmt.Printf("  Hour: %s, Stars: %d, Total: %d\n",
				hourlyStars7d[i].Hour.Format("2006-01-02 15:04"),
				hourlyStars7d[i].Stars,
				hourlyStars7d[i].TotalStars)
		}

		// Show last few hours (most recent)
		fmt.Println("\nLast 5 hours (most recent):")
		start := len(hourlyStars7d) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(hourlyStars7d); i++ {
			fmt.Printf("  Hour: %s, Stars: %d, Total: %d\n",
				hourlyStars7d[i].Hour.Format("2006-01-02 15:04"),
				hourlyStars7d[i].Stars,
				hourlyStars7d[i].TotalStars)
		}

		// Find peak hour
		maxStars := 0
		maxHour := time.Time{}
		for _, h := range hourlyStars7d {
			if h.Stars > maxStars {
				maxStars = h.Stars
				maxHour = h.Hour
			}
		}
		fmt.Printf("\nPeak hour: %s with %d stars\n", maxHour.Format("2006-01-02 15:04"), maxStars)
		fmt.Printf("Total stars in last 7 days: %d\n", hourlyStars7d[len(hourlyStars7d)-1].TotalStars)
	}

	// Example 2: Get hourly stars for the last 3 days for another repo
	fmt.Println("\n=== Example 2: Hourly stars for your repo (last 3 days) ===")
	hourlyStars3d, err := clientGQL.GetRecentStarsHistoryByHour(ctx, "emanuelef/github-repo-activity-stats", 3, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Total hours tracked: %d\n", len(hourlyStars3d))
		
		// Group by day and show daily totals
		dayTotals := make(map[string]int)
		for _, h := range hourlyStars3d {
			day := h.Hour.Format("2006-01-02")
			dayTotals[day] += h.Stars
		}

		fmt.Println("\nDaily breakdown:")
		for day, stars := range dayTotals {
			fmt.Printf("  %s: %d stars\n", day, stars)
		}

		if len(hourlyStars3d) > 0 {
			fmt.Printf("\nTotal stars in last 3 days: %d\n", hourlyStars3d[len(hourlyStars3d)-1].TotalStars)
		}
	}

	// Example 3: Compare daily vs hourly granularity
	fmt.Println("\n=== Example 3: Comparing daily vs hourly data (last 2 days) ===")
	
	// Get daily data
	dailyStars, err := clientGQL.GetRecentStarsHistoryTwoWays(ctx, "langflow-ai/langflow", 2, nil)
	if err != nil {
		fmt.Printf("Error getting daily data: %v\n", err)
	}

	// Get hourly data
	hourlyStars2d, err := clientGQL.GetRecentStarsHistoryByHour(ctx, "langflow-ai/langflow", 2, nil)
	if err != nil {
		fmt.Printf("Error getting hourly data: %v\n", err)
	}

	if err == nil {
		fmt.Println("\nDaily data:")
		for _, day := range dailyStars {
			fmt.Printf("  %s: %d stars (total: %d)\n",
				time.Time(day.Day).Format("2006-01-02"),
				day.Stars,
				day.TotalStars)
		}

		fmt.Printf("\nHourly data provides %d data points vs %d daily data points\n",
			len(hourlyStars2d), len(dailyStars))
		
		// Show hours with most activity
		fmt.Println("\nTop 5 most active hours in last 2 days:")
		type hourStat struct {
			hour  time.Time
			stars int
		}
		var topHours []hourStat
		for _, h := range hourlyStars2d {
			if h.Stars > 0 {
				topHours = append(topHours, hourStat{h.Hour, h.Stars})
			}
		}
		
		// Simple bubble sort for top 5
		for i := 0; i < len(topHours); i++ {
			for j := i + 1; j < len(topHours); j++ {
				if topHours[j].stars > topHours[i].stars {
					topHours[i], topHours[j] = topHours[j], topHours[i]
				}
			}
		}

		limit := 5
		if len(topHours) < limit {
			limit = len(topHours)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %d. %s: %d stars\n",
				i+1,
				topHours[i].hour.Format("2006-01-02 15:04"),
				topHours[i].stars)
		}
	}

	// Show rate limits
	fmt.Println("\n=== API Rate Limit Status ===")
	rateLimit, err := clientGQL.GetCurrentLimits(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Limit: %d, Remaining: %d, Resets at: %s\n",
			rateLimit.Limit,
			rateLimit.Remaining,
			rateLimit.ResetAt.Format("15:04:05"))
	}
}

package repostats

import (
	"sort"

	"github.com/emanuelef/github-repo-activity-stats/stats"
)

type MaxPeriod struct {
	StartDay   stats.JSONDay
	EndDay     stats.JSONDay
	TotalStars int
}

type PeakDay struct {
	Day   stats.JSONDay
	Stars int
}

func FindMaxConsecutivePeriods(starsData []stats.StarsPerDay, consecutiveDays int) ([]MaxPeriod, []PeakDay, error) {
	var maxPeriods []MaxPeriod
	var peakDays []PeakDay

	// Calculate maxSum and maxPeriods for consecutive periods
	maxSum := 0
	for i := 0; i <= len(starsData)-consecutiveDays; i++ {
		sum := 0
		for j := i; j < i+consecutiveDays; j++ {
			sum += starsData[j].Stars
		}

		if sum > maxSum {
			maxSum = sum
			maxPeriods = []MaxPeriod{
				{
					StartDay:   starsData[i].Day,
					EndDay:     starsData[i+consecutiveDays-1].Day,
					TotalStars: sum,
				},
			}
		} else if sum == maxSum {
			maxPeriods = append(maxPeriods, MaxPeriod{
				StartDay:   starsData[i].Day,
				EndDay:     starsData[i+consecutiveDays-1].Day,
				TotalStars: sum,
			})
		}
	}

	// Calculate peakDays
	starMap := make(map[stats.JSONDay]int)
	for _, data := range starsData {
		starMap[data.Day] = data.Stars
	}

	// Sort the days by stars in descending order
	var sortedDays []PeakDay
	for day, stars := range starMap {
		sortedDays = append(sortedDays, PeakDay{Day: day, Stars: stars})
	}
	sort.Slice(sortedDays, func(i, j int) bool {
		return sortedDays[i].Stars > sortedDays[j].Stars
	})

	// Find the days with maximum stars
	maxStars := sortedDays[0].Stars
	for _, day := range sortedDays {
		if day.Stars == maxStars {
			peakDays = append(peakDays, day)
		} else {
			break // Days are sorted, so no need to check further
		}
	}

	return maxPeriods, peakDays, nil
}

func NewStarsLastDays(starsData []stats.StarsPerDay, days int) int {
	sum := 0

	if days > len(starsData) {
		days = len(starsData)
	}

	endIndex := len(starsData) - days
	for i := endIndex; i < len(starsData); i++ {
		sum += starsData[i].Stars
	}

	return sum
}

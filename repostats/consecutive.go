package repostats

import (
	"github.com/emanuelef/github-repo-activity-stats/stats"
)

type MaxPeriod struct {
	StartDay   stats.JSONDay
	EndDay     stats.JSONDay
	TotalStars int
}

func FindMaxConsecutivePeriods(starsData []stats.StarsPerDay, consecutiveDays int) ([]MaxPeriod, error) {
	var maxPeriods []MaxPeriod
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

	return maxPeriods, nil
}

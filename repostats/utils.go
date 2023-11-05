package repostats

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/stats"
)

func WriteStarsHistoryCSV(filename string, history []stats.StarsPerDay) {
	outputFile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	defer outputFile.Close()

	csvWriter := csv.NewWriter(outputFile)
	defer csvWriter.Flush()

	headerRow := []string{
		"date", "daily-stars", "total-stars",
	}

	csvWriter.Write(headerRow)

	for _, v := range history {
		csvWriter.Write([]string{
			fmt.Sprintf("%s", time.Time(v.Day).Format("02-01-2006")),
			fmt.Sprintf("%d", v.Stars),
			fmt.Sprintf("%d", v.TotalStars),
		})
	}
}

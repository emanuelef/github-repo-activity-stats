package stats

import (
	"encoding/json"
	"fmt"
	"time"
)

type GoRepo struct {
	GoVersion  string
	DirectDeps []string
}

type JSONDay time.Time

func (t JSONDay) MarshalJSON() ([]byte, error) {
	dayFormatted := fmt.Sprintf("\"%s\"", time.Time(t).Format("02-01-2006"))
	return []byte(dayFormatted), nil
}

type StarsPerDay struct {
	Day        JSONDay
	Stars      int
	TotalStars int
}

type CommitsPerDay struct {
	Day          JSONDay
	Commits      int
	TotalCommits int
}

func (t StarsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Stars, t.TotalStars})
}

func (t CommitsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Commits, t.TotalCommits})
}

type StarsHistory struct {
	AddedLast24H     int
	AddedLast7d      int
	AddedLast14d     int
	AddedLast30d     int
	LastStarDate     time.Time
	AddedPerMille30d float32
	StarsTimeline    []StarsPerDay
}

type CommitsHistory struct {
	AddedLast24H     int
	AddedLast7d      int
	AddedLast14d     int
	AddedLast30d     int
	LastCommitDate   time.Time
	AddedPerMille30d float32
	CommitsTimeline  []CommitsPerDay
	DifferentAuthors int
}

func (sh CommitsHistory) String() string {
	return fmt.Sprintf(`Last Commit Date: %s
	Commits Different Authors %d
Commits AddedLast24H: %d
Commits AddedLast7d: %d
Commits AddedLast14d: %d
Commits AddedLast30d: %d
Commits AddedPerMille30d: %.2f`,
		sh.LastCommitDate,
		sh.DifferentAuthors,
		sh.AddedLast24H,
		sh.AddedLast7d,
		sh.AddedLast14d,
		sh.AddedLast30d,
		sh.AddedPerMille30d)
}

func (sh StarsHistory) String() string {
	return fmt.Sprintf(`Last Star Date: %s
Stars AddedLast24H: %d
Stars AddedLast7d: %d
Stars AddedLast14d: %d
Stars AddedLast30d: %d
Stars AddedPerMille30d: %.2f`, sh.LastStarDate,
		sh.AddedLast24H,
		sh.AddedLast7d,
		sh.AddedLast14d,
		sh.AddedLast30d,
		sh.AddedPerMille30d)
}

type RepoStats struct {
	GHPath           string
	Stars            int
	Commits          int
	Size             int
	Language         string
	OpenIssues       int
	Forks            int
	Archived         bool
	DefaultBranch    string
	MentionableUsers int
	CreatedAt        time.Time
	LastReleaseDate  time.Time
	LivenessScore    float32
	StarsHistory
	CommitsHistory
	GoRepo
}

func (rs RepoStats) String() string {
	return fmt.Sprintf(`
GH Repo: %s
Created: %s
Last Commit: %s
Last Release: %s
Stars: %d
Size: %d
Language: %s
Open Issues: %d
Forks: %d
Archived: %t
Mentionable Users: %d
Default Branch: %s
%s
%s
Liveness Score: %.2f
Go version: %s
Go Direct dependencies: %d
	`, rs.GHPath,
		rs.CreatedAt,
		rs.LastCommitDate,
		rs.LastReleaseDate,
		rs.Stars,
		rs.Size,
		rs.Language,
		rs.OpenIssues,
		rs.Forks,
		rs.Archived,
		rs.MentionableUsers,
		rs.DefaultBranch,
		rs.StarsHistory,
		rs.CommitsHistory,
		rs.LivenessScore,
		rs.GoVersion,
		len(rs.DirectDeps))
}

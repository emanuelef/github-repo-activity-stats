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

type IssuesPerDay struct {
	Day                JSONDay
	Opened             int
	Closed             int
	TotalOpened        int
	TotalClosed        int
	CurrentlyOpen      int
	TotalCurrentlyOpen int
}

type ForksPerDay struct {
	Day        JSONDay
	Forks      int
	TotalForks int
}

type PRsPerDay struct {
	Day                JSONDay
	Opened             int
	Merged             int
	Closed             int
	TotalOpened        int
	TotalMerged        int
	TotalClosed        int
	CurrentlyOpen      int
	TotalCurrentlyOpen int
}

// NewContributorsPerDay holds statistics about new contributors for a specific day.
type NewContributorsPerDay struct {
	Day                  JSONDay
	NewContributors      int
	TotalNewContributors int
}

func (t StarsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Stars, t.TotalStars})
}

func (t CommitsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Commits, t.TotalCommits})
}

func (t IssuesPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Opened, t.Closed, t.TotalOpened, t.TotalClosed, t.CurrentlyOpen, t.TotalCurrentlyOpen})
}

func (t ForksPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Forks, t.TotalForks})
}

func (t PRsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.Opened, t.Merged, t.Closed, t.TotalOpened, t.TotalMerged, t.TotalClosed, t.CurrentlyOpen, t.TotalCurrentlyOpen})
}

func (t NewContributorsPerDay) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{t.Day, t.NewContributors, t.TotalNewContributors})
}

// ReleaseInfo represents a single GitHub release with its metadata
type ReleaseInfo struct {
	CreatedAt     time.Time `json:"createdAt"`
	PublishedAt   time.Time `json:"publishedAt"`
	Name          string    `json:"name"`
	TagName       string    `json:"tagName"`
	IsPrerelease  bool      `json:"isPrerelease"`
	IsDraft       bool      `json:"isDraft"`
	URL           string    `json:"url"`
	AuthorLogin   string    `json:"authorLogin"`
	TotalReleases int       `json:"totalReleases"` // Cumulative count at this point
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

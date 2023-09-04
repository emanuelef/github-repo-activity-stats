package repostats

import (
	"fmt"
	"time"
)

type GoRepo struct {
	GoVersion  string
	DirectDeps []string
}

type StarsHistory struct {
	AddedLast24H     int
	AddedLast7d      int
	AddedLast14d     int
	AddedLast30d     int
	LastStarDate     time.Time
	AddedPerMille30d float32
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
	Size             int
	Language         string
	OpenIssues       int
	Forks            int
	Archived         bool
	DefaultBranch    string
	MentionableUsers int
	CreatedAt        time.Time
	LastCommitDate   time.Time
	LastReleaseDate  time.Time
	StarsHistory
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
		rs.GoVersion,
		len(rs.DirectDeps))
}

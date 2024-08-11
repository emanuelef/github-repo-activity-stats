package repostats

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emanuelef/github-repo-activity-stats/deps"
	"github.com/emanuelef/github-repo-activity-stats/stats"
	"github.com/go-resty/resty/v2"
	"github.com/shurcooL/githubv4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/emanuelef/cncf-repos-stats")
}

type Counter struct {
	mu      sync.Mutex
	counter int
}

func (c *Counter) Increment() {
	c.mu.Lock()
	c.counter++
	c.mu.Unlock()
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter
}

type ClientGQL struct {
	ghClient    *githubv4.Client
	restyClient *resty.Client
}

func NewClientGQL(oauthClient *http.Client) *ClientGQL {
	ghClient := githubv4.NewClient(oauthClient)
	restyClient := resty.NewWithClient(
		&http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	)
	return &ClientGQL{ghClient: ghClient, restyClient: restyClient}
}

func (c *ClientGQL) query(ctx context.Context, q any, variables map[string]any) error {
	ctx, span := tracer.Start(ctx, "graphql-query")
	defer span.End()
	err := c.ghClient.Query(ctx, q, variables)
	return err
}

func (c *ClientGQL) GetAllStarsHistory(ctx context.Context, ghRepo string, repoCreationDate time.Time, updateChannel chan<- int) ([]stats.StarsPerDay, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.StarsPerDay{}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.StarsPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	variablesStars := map[string]any{
		"owner":       githubv4.String(owner),
		"name":        githubv4.String(name),
		"starsCursor": (*githubv4.String)(nil),
	}

	type starred struct {
		StarredAt time.Time
	}

	var queryStars struct {
		Repository struct {
			Stargazers struct {
				Edges    []starred
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"stargazers(first: 100, after: $starsCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	i := 0
	for {
		err := c.query(ctx, &queryStars, variablesStars)
		if err != nil {
			break
		}

		res := queryStars.Repository.Stargazers.Edges

		if len(res) == 0 {
			break
		}

		for _, star := range res {
			days := star.StarredAt.Sub(repoCreationDate).Hours() / 24
			result[int(days)].Stars++
		}

		if !queryStars.Repository.Stargazers.PageInfo.HasNextPage {
			break
		}

		variablesStars["starsCursor"] = githubv4.NewString(queryStars.Repository.Stargazers.PageInfo.EndCursor)
		i++
		if updateChannel != nil {
			updateChannel <- i
		}
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalStars = result[i-1].TotalStars + day.Stars
		} else {
			result[i].TotalStars = day.Stars
		}
	}

	if updateChannel != nil {
		close(updateChannel)
	}

	return result, nil
}

func (c *ClientGQL) getStarsHistory(ctx context.Context, owner, name string, totalStars int) (stats.StarsHistory, error) {
	result := stats.StarsHistory{}

	ctx, span := tracer.Start(ctx, "fetch-all-stars")
	defer span.End()

	if totalStars == 0 {
		return result, nil
	}

	currentTime := time.Now()

	variablesStars := map[string]any{
		"owner":       githubv4.String(owner),
		"name":        githubv4.String(name),
		"starsCursor": (*githubv4.String)(nil),
	}

	type starred struct {
		StarredAt time.Time
	}

	var queryStars struct {
		Repository struct {
			Stargazers struct {
				TotalCount int
				Edges      []starred
				PageInfo   struct {
					StartCursor     githubv4.String
					HasPreviousPage bool
				}
			} `graphql:"stargazers(last: 100, before: $starsCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for i := 1; i < 31; i++ {
		result.StarsTimeline = append(result.StarsTimeline, stats.StarsPerDay{Day: stats.JSONDay(currentTime.AddDate(0, 0, -(30 - i)).Truncate(24 * time.Hour))})
	}

	for {
		err := c.query(ctx, &queryStars, variablesStars)
		if err != nil {
			return result, err
		}

		// fmt.Println("Desc:", len(queryStars.Repository.Stargazers.Edges))

		res := queryStars.Repository.Stargazers.Edges
		slices.Reverse(res) // order from most recent to least

		if len(res) > 0 && result.LastStarDate.IsZero() {
			result.LastStarDate = res[0].StarredAt
		}

		moreThan30daysFlag := false

		for _, star := range res {
			days := currentTime.Sub(star.StarredAt).Hours() / 24

			if days > 30 {
				moreThan30daysFlag = true
				break
			}

			if days <= 1 {
				result.AddedLast24H += 1
			}

			if days <= 7 {
				result.AddedLast7d += 1
			}

			if days <= 14 {
				result.AddedLast14d += 1
			}

			if days <= 30 {
				result.AddedLast30d += 1
			}

			result.StarsTimeline[29-int(days)].Stars += 1
		}

		if !queryStars.Repository.Stargazers.PageInfo.HasPreviousPage || moreThan30daysFlag {
			break
		}

		variablesStars["starsCursor"] = githubv4.NewString(queryStars.Repository.Stargazers.PageInfo.StartCursor)
	}

	if totalStars > 0 {
		result.AddedPerMille30d = 1000 * (float32(result.AddedLast30d) / float32(totalStars))
	}

	for i := len(result.StarsTimeline) - 1; i >= 0; i-- {
		if i == len(result.StarsTimeline)-1 {
			result.StarsTimeline[i].TotalStars = totalStars
		} else {
			result.StarsTimeline[i].TotalStars = result.StarsTimeline[i+1].TotalStars - result.StarsTimeline[i+1].Stars
		}
	}

	return result, nil
}

func (c *ClientGQL) getCommitsShortHistory(ctx context.Context, owner, name string, totalCommits int) (stats.CommitsHistory, error) {
	result := stats.CommitsHistory{}

	ctx, span := tracer.Start(ctx, "fetch-short-commits")
	defer span.End()

	if totalCommits == 0 {
		return result, nil
	}

	currentTime := time.Now()

	uniqueAuthors := make(map[string]struct{})

	variablesCommits := map[string]any{
		"owner":         githubv4.String(owner),
		"name":          githubv4.String(name),
		"commitsCursor": (*githubv4.String)(nil),
	}

	type commit struct {
		Node struct {
			Author struct {
				User struct {
					Id string
				}
			}
			CommittedDate time.Time
			Additions     int
		}
	}

	var queryCommits struct {
		Repository struct {
			DefaultBranchRef struct {
				Name   string
				Target struct {
					Commit struct {
						History struct {
							Edges    []commit
							PageInfo struct {
								EndCursor   githubv4.String
								HasNextPage bool
							}
						} `graphql:"history(first: 100, after: $commitsCursor)"`
					} `graphql:"... on Commit"`
				}
			}
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for i := 1; i < 31; i++ {
		result.CommitsTimeline = append(result.CommitsTimeline, stats.CommitsPerDay{Day: stats.JSONDay(currentTime.AddDate(0, 0, -(30 - i)).Truncate(24 * time.Hour))})
	}

	for {
		err := c.query(ctx, &queryCommits, variablesCommits)
		if err != nil {
			return result, err
		}

		// fmt.Println("Desc:", len(queryStars.Repository.Stargazers.Edges))

		res := queryCommits.Repository.DefaultBranchRef.Target.Commit.History.Edges

		if len(res) > 0 && result.LastCommitDate.IsZero() {
			result.LastCommitDate = res[0].Node.CommittedDate
		}

		moreThan30daysFlag := false

		for _, star := range res {
			days := currentTime.Sub(star.Node.CommittedDate).Hours() / 24

			if days > 30 {
				moreThan30daysFlag = true
				break
			}

			if days <= 1 {
				result.AddedLast24H += 1
			}

			if days <= 7 {
				result.AddedLast7d += 1
			}

			if days <= 14 {
				result.AddedLast14d += 1
			}

			if days <= 30 {
				result.AddedLast30d += 1
			}

			result.CommitsTimeline[29-int(days)].Commits += 1
			uniqueAuthors[star.Node.Author.User.Id] = struct{}{}
		}

		if !queryCommits.Repository.DefaultBranchRef.Target.Commit.History.PageInfo.HasNextPage || moreThan30daysFlag {
			break
		}

		variablesCommits["commitsCursor"] = githubv4.NewString(queryCommits.Repository.DefaultBranchRef.Target.Commit.History.PageInfo.EndCursor)
	}

	if totalCommits > 0 {
		result.AddedPerMille30d = 1000 * (float32(result.AddedLast30d) / float32(totalCommits))
	}

	result.DifferentAuthors = len(uniqueAuthors)

	for i := len(result.CommitsTimeline) - 1; i >= 0; i-- {
		if i == len(result.CommitsTimeline)-1 {
			result.CommitsTimeline[i].TotalCommits = totalCommits
		} else {
			result.CommitsTimeline[i].TotalCommits = result.CommitsTimeline[i+1].TotalCommits - result.CommitsTimeline[i+1].Commits
		}
	}

	return result, nil
}

func (c *ClientGQL) GetAllStarsHistoryTwoWays(ctx context.Context, ghRepo string, updateChannel chan<- int) ([]stats.StarsPerDay, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	defer func() {
		if updateChannel != nil {
			close(updateChannel)
		}
	}()

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.StarsPerDay{}
	counter := &Counter{}

	var resultMutex sync.Mutex

	totalStars, repoCreationDate, err := c.GetTotalStars(ctx, ghRepo)
	if err != nil {
		log.Printf("%v\n", err)
		return result, err
	}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.StarsPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	type starred struct {
		StarredAt time.Time
		Cursor    string
	}

	processedStars := make(map[string]struct{})

	eg, ctx := errgroup.WithContext(ctx)

	fmt.Printf("Total Stars: %d %d %d %d \n", totalStars, int(math.Ceil(float64(totalStars/2)/100))+1, int(math.Floor(float64(totalStars/2)/100))-1, int(math.Ceil(float64(totalStars)/100)))

	forwardLimit := int(math.Ceil(float64(totalStars/2)/100)) + 1
	backwardLimit := int(math.Floor(float64(totalStars/2) / 100))

	if totalStars < 300 {
		forwardLimit = int(math.Floor(float64(totalStars)/100)) + 1
		backwardLimit = int(math.Ceil(float64(totalStars/2)/100)) + 1
	}

	eg.Go(func() error {
		variablesStars := map[string]any{
			"owner":       githubv4.String(owner),
			"name":        githubv4.String(name),
			"starsCursor": (*githubv4.String)(nil),
		}

		var queryStars struct {
			Repository struct {
				Stargazers struct {
					Edges    []starred
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
				} `graphql:"stargazers(first: 100, after: $starsCursor)"`
			} `graphql:"repository(owner: $owner, name: $name)"`
		}

		for i := 0; i < forwardLimit; i++ {
			err := c.query(ctx, &queryStars, variablesStars)
			if err != nil {
				fmt.Printf("F %d %v\n", i, err)
				return err
			}

			res := queryStars.Repository.Stargazers.Edges

			if len(res) == 0 {
				break
			}

			resultMutex.Lock()
			for _, star := range res {
				starID := star.Cursor
				if _, ok := processedStars[starID]; !ok {
					processedStars[starID] = struct{}{}
					days := star.StarredAt.Sub(repoCreationDate).Hours() / 24
					result[int(days)].Stars++
				}
			}
			resultMutex.Unlock()

			if !queryStars.Repository.Stargazers.PageInfo.HasNextPage {
				break
			}

			variablesStars["starsCursor"] = githubv4.NewString(queryStars.Repository.Stargazers.PageInfo.EndCursor)

			counter.Increment()

			if updateChannel != nil {
				updateChannel <- counter.Value()
			}
		}
		return nil
	})

	eg.Go(func() error {
		variablesStars := map[string]any{
			"owner":       githubv4.String(owner),
			"name":        githubv4.String(name),
			"starsCursor": (*githubv4.String)(nil),
		}

		var queryStars struct {
			Repository struct {
				Stargazers struct {
					Edges    []starred
					PageInfo struct {
						StartCursor     githubv4.String
						HasPreviousPage bool
					}
				} `graphql:"stargazers(last: 100, before: $starsCursor)"`
			} `graphql:"repository(owner: $owner, name: $name)"`
		}

		for i := 0; i < backwardLimit; i++ {
			err := c.query(ctx, &queryStars, variablesStars)
			if err != nil {
				fmt.Printf("B %d %v\n", i, err)
				starsCursor := ""
				if v, ok := variablesStars["starsCursor"].(*githubv4.String); ok && v != nil {
					starsCursor = string(*v)
				} else {
					log.Println("Warning: starsCursor is nil or not a githubv4.String")
				}

				fmt.Println(starsCursor)
				return err
			}

			res := queryStars.Repository.Stargazers.Edges

			if len(res) == 0 {
				break
			}

			resultMutex.Lock()
			for _, star := range res {
				starID := star.Cursor
				if _, ok := processedStars[starID]; !ok {
					processedStars[starID] = struct{}{}
					days := star.StarredAt.Sub(repoCreationDate).Hours() / 24
					result[int(days)].Stars++
				}
			}
			resultMutex.Unlock()

			if !queryStars.Repository.Stargazers.PageInfo.HasPreviousPage {
				break
			}

			variablesStars["starsCursor"] = githubv4.NewString(queryStars.Repository.Stargazers.PageInfo.StartCursor)

			counter.Increment()

			if updateChannel != nil {
				updateChannel <- counter.Value()
			}
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		// Handle the first error that occurred.
		log.Printf("An error occurred: %v", err)
		return result, err
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalStars = result[i-1].TotalStars + day.Stars
		} else {
			result[i].TotalStars = day.Stars
		}
	}

	return result, nil
}

type RateLimit struct {
	Limit     int
	Cost      int
	Remaining int
	ResetAt   time.Time
}

func (c *ClientGQL) GetCurrentLimits(ctx context.Context) (*RateLimit, error) {
	result := RateLimit{}

	var query struct {
		RateLimit struct {
			Limit     int
			Cost      int
			Remaining int
			ResetAt   time.Time
		}
	}

	err := c.query(ctx, &query, nil)
	if err != nil {
		log.Printf("%v\n", err)
		return &RateLimit{}, err
	}

	result.Limit = query.RateLimit.Limit
	result.Cost = query.RateLimit.Cost
	result.Remaining = query.RateLimit.Remaining
	result.ResetAt = query.RateLimit.ResetAt

	return &result, nil
}

func (c *ClientGQL) GetAllStats(ctx context.Context, ghRepo string) (*stats.RepoStats, error) {
	result := stats.RepoStats{}

	currentTime := time.Now()

	ctx, span := tracer.Start(ctx, "fetch-repo-stats")
	defer span.End()

	span.SetAttributes(attribute.String("github.repo", ghRepo))

	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	variables := map[string]any{
		"owner": githubv4.String(repoSplit[0]),
		"name":  githubv4.String(repoSplit[1]),
	}

	type commit struct {
		Node struct {
			CommittedDate time.Time
		}
	}

	type release struct {
		Node struct {
			CreatedAt   time.Time
			PublishedAt time.Time
			Name        string
		}
	}

	type starred struct {
		StarredAt time.Time
	}

	var query struct {
		Repository struct {
			Description     string
			StargazerCount  int
			CreatedAt       time.Time
			PrimaryLanguage struct {
				Name string
			}
			ForkCount        int
			IsArchived       bool
			DiskUsage        int
			MentionableUsers struct {
				TotalCount int
			}
			OpenIssues struct {
				TotalCount int
			} `graphql:"issues(states: OPEN)"`
			DefaultBranchRef struct {
				Name   string
				Target struct {
					Commit struct {
						History struct {
							TotalCount int
							Edges      []commit
						} `graphql:"history(first: 1)"`
					} `graphql:"... on Commit"`
				}
			}
			Stargazers struct {
				Edges []starred
			} `graphql:"stargazers(last: 1)"`
			Releases struct {
				TotalCount int
				Edges      []release
			} `graphql:"releases(first: 1)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	err := c.query(ctx, &query, variables)
	if err != nil {
		log.Printf("%v\n", err)
		return &result, err
	}
	fmt.Println("Desc:", query.Repository.Description)
	fmt.Println("Total Commit:", query.Repository.DefaultBranchRef.Target.Commit.History.TotalCount)

	result.GHPath = ghRepo
	result.CreatedAt = query.Repository.CreatedAt
	result.Stars = query.Repository.StargazerCount
	result.Commits = query.Repository.DefaultBranchRef.Target.Commit.History.TotalCount
	result.DefaultBranch = query.Repository.DefaultBranchRef.Name
	result.Archived = query.Repository.IsArchived
	result.Forks = query.Repository.ForkCount
	result.OpenIssues = query.Repository.OpenIssues.TotalCount
	result.Language = query.Repository.PrimaryLanguage.Name
	result.Size = query.Repository.DiskUsage
	result.MentionableUsers = query.Repository.MentionableUsers.TotalCount

	commits := query.Repository.DefaultBranchRef.Target.Commit.History.Edges
	if len(commits) > 0 {
		result.LastCommitDate = commits[0].Node.CommittedDate
	}
	releases := query.Repository.Releases.Edges
	if len(releases) > 0 {
		result.LastReleaseDate = releases[0].Node.CreatedAt
	}

	stars := query.Repository.Stargazers.Edges

	if len(stars) > 0 && result.LastStarDate.IsZero() {
		result.LastStarDate = stars[0].StarredAt
	}

	// 30d stars history

	days := currentTime.Sub(result.LastStarDate).Hours() / 24

	if days < 30 {
		result.StarsHistory, err = c.getStarsHistory(ctx, repoSplit[0], repoSplit[1], result.Stars)
		if err != nil {
			log.Printf("%v\n", err)
			return &result, err
		}
	} else {
		for i := 1; i < 31; i++ {
			result.StarsHistory.StarsTimeline = append(result.StarsTimeline, stats.StarsPerDay{Day: stats.JSONDay(currentTime.AddDate(0, 0, -(30 - i)).Truncate(24 * time.Hour))})
		}
	}

	// 30d commits history
	days = currentTime.Sub(result.LastCommitDate).Hours() / 24

	if days < 30 {
		result.CommitsHistory, err = c.getCommitsShortHistory(ctx, repoSplit[0], repoSplit[1], result.Commits)
		if err != nil {
			log.Printf("%v\n", err)
			return &result, err
		}
	} else {
		for i := 1; i < 31; i++ {
			result.CommitsHistory.CommitsTimeline = append(result.CommitsTimeline, stats.CommitsPerDay{Day: stats.JSONDay(currentTime.AddDate(0, 0, -(30 - i)).Truncate(24 * time.Hour))})
		}
	}

	if depFetcher := deps.CreateFetcher(result.Language); depFetcher != nil {
		depFetcher.GetDepsList(ctx, c.restyClient, ghRepo, &result)
	}

	getLivenessScore(ctx, c.restyClient, ghRepo, &result)

	return &result, nil
}

func getLivenessScore(ctx context.Context, restyClient *resty.Client, ghRepo string, result *stats.RepoStats) {
	score := float32(0.0)

	// calculate days since last commit
	if !result.LastCommitDate.IsZero() {
		days := time.Now().Sub(result.LastCommitDate).Hours() / 24

		switch {
		case days <= 1:
			score += 50
			break
		case days <= 3:
			score += 40
			break
		case days < 7:
			score += 30
			break
		case days < 14:
			score += 20
			break
		case days < 30:
			score += 10
			break
		case days < 60:
			score += 6
			break
		}
	}

	// calculate days since last star
	if !result.LastStarDate.IsZero() {
		days := time.Now().Sub(result.LastStarDate).Hours() / 24
		switch {
		case days <= 1:
			score += 20
			break
		case days < 7:
			score += 10
			break
		case days < 14:
			score += 5
			break
		case days < 30:
			score += 2
			break
		}
	}

	switch {
	case result.StarsHistory.AddedLast30d > 20:
		score += 10
		break
	case result.StarsHistory.AddedLast30d > 10:
		score += 6
		break
	case result.StarsHistory.AddedLast30d > 1:
		score += 2
		break
	}

	switch {
	case result.StarsHistory.AddedLast14d > 50:
		score += 30
		break
	case result.StarsHistory.AddedLast14d > 30:
		score += 20
		break
	case result.StarsHistory.AddedLast14d > 20:
		score += 10
		break
	case result.StarsHistory.AddedLast14d > 5:
		score += 5
		break
	}

	switch {
	case result.StarsHistory.AddedLast24H > 30:
		score += 10
		break
	case result.StarsHistory.AddedLast24H > 20:
		score += 5
		break
	case result.StarsHistory.AddedLast24H > 5:
		score += 2
		break
	}

	switch {
	case result.CommitsHistory.AddedLast30d > 20:
		score += 10
		break
	case result.CommitsHistory.AddedLast30d > 10:
		score += 6
		break
	case result.CommitsHistory.AddedLast30d > 1:
		score += 2
		break
	}

	if result.Archived {
		score -= 30
	}

	// score should be between 0 and 100
	score = float32(math.Max(0, math.Min(100, float64(score))))

	result.LivenessScore = score
}

func (c *ClientGQL) GetTotalStars(ctx context.Context, ghRepo string) (int, time.Time, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return -1, time.Time{}, fmt.Errorf("Repo should be provided as owner/name")
	}

	variables := map[string]any{
		"owner": githubv4.String(repoSplit[0]),
		"name":  githubv4.String(repoSplit[1]),
	}

	var query struct {
		Repository struct {
			StargazerCount int
			CreatedAt      time.Time
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	err := c.query(ctx, &query, variables)
	if err != nil {
		log.Printf("%v\n", err)
		return 0, time.Time{}, err
	}

	return query.Repository.StargazerCount, query.Repository.CreatedAt, nil
}

func (c *ClientGQL) GetAllIssuesHistory(ctx context.Context, ghRepo string, updateChannel chan<- int) ([]stats.IssuesPerDay, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	defer func() {
		if updateChannel != nil {
			close(updateChannel)
		}
	}()

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.IssuesPerDay{}
	counter := &Counter{}

	_, repoCreationDate, err := c.GetTotalStars(ctx, ghRepo)
	if err != nil {
		log.Printf("%v\n", err)
		return result, err
	}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.IssuesPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	variablesStars := map[string]any{
		"owner":        githubv4.String(owner),
		"name":         githubv4.String(name),
		"issuesCursor": (*githubv4.String)(nil),
	}

	type issues struct {
		State     string
		ClosedAt  time.Time
		CreatedAt time.Time
	}

	var queryStars struct {
		Repository struct {
			Issues struct {
				Nodes    []issues
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"issues(first: 100, orderBy: {field: CREATED_AT, direction: ASC}, after: $issuesCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		err := c.query(ctx, &queryStars, variablesStars)
		if err != nil {
			fmt.Printf("%v\n", err)
			return nil, err
		}

		res := queryStars.Repository.Issues.Nodes

		if len(res) == 0 {
			break
		}

		for _, issue := range res {
			daysOpened := issue.CreatedAt.Sub(repoCreationDate).Hours() / 24

			if daysOpened < 0 {
				continue
			}

			result[int(daysOpened)].Opened++

			if issue.State == "CLOSED" {
				if !issue.ClosedAt.IsZero() {
					daysClosed := issue.ClosedAt.Sub(repoCreationDate).Hours() / 24
					result[int(daysClosed)].Closed++
				}
			}

			if issue.State == "OPEN" {
				result[int(daysOpened)].CurrentlyOpen++
			}
		}

		if !queryStars.Repository.Issues.PageInfo.HasNextPage {
			break
		}

		variablesStars["issuesCursor"] = githubv4.NewString(queryStars.Repository.Issues.PageInfo.EndCursor)

		counter.Increment()

		if updateChannel != nil {
			updateChannel <- counter.Value()
		}
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalOpened = result[i-1].TotalOpened + day.Opened
			result[i].TotalClosed = result[i-1].TotalClosed + day.Closed
			result[i].TotalCurrentlyOpen = result[i-1].TotalCurrentlyOpen + day.CurrentlyOpen
		} else {
			result[i].TotalOpened = day.Opened
			result[i].TotalClosed = day.Closed
			result[i].TotalCurrentlyOpen = day.CurrentlyOpen
		}
	}

	return result, nil
}

func (c *ClientGQL) GetAllForksHistory(ctx context.Context, ghRepo string, updateChannel chan<- int) ([]stats.ForksPerDay, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	defer func() {
		if updateChannel != nil {
			close(updateChannel)
		}
	}()

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.ForksPerDay{}
	counter := &Counter{}

	_, repoCreationDate, err := c.GetTotalStars(ctx, ghRepo)
	if err != nil {
		log.Printf("%v\n", err)
		return result, err
	}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.ForksPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	type starred struct {
		StarredAt time.Time
		Cursor    string
	}

	variablesStars := map[string]any{
		"owner":       githubv4.String(owner),
		"name":        githubv4.String(name),
		"forksCursor": (*githubv4.String)(nil),
	}

	type forks struct {
		CreatedAt time.Time
	}

	var queryStars struct {
		Repository struct {
			Forks struct {
				Nodes    []forks
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"forks(first: 100, orderBy: {field: CREATED_AT, direction: ASC}, after: $forksCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		err := c.query(ctx, &queryStars, variablesStars)
		if err != nil {
			fmt.Printf("%v\n", err)
			return nil, err
		}

		res := queryStars.Repository.Forks.Nodes

		if len(res) == 0 {
			break
		}

		for _, fork := range res {
			daysForkCreated := fork.CreatedAt.Sub(repoCreationDate).Hours() / 24

			if daysForkCreated < 0 {
				continue
			}

			result[int(daysForkCreated)].Forks++
		}

		if !queryStars.Repository.Forks.PageInfo.HasNextPage {
			break
		}

		variablesStars["forksCursor"] = githubv4.NewString(queryStars.Repository.Forks.PageInfo.EndCursor)

		counter.Increment()

		if updateChannel != nil {
			updateChannel <- counter.Value()
		}
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalForks = result[i-1].TotalForks + day.Forks
		} else {
			result[i].TotalForks = day.Forks
		}
	}
	return result, nil
}

func (c *ClientGQL) GetAllPRsHistory(ctx context.Context, ghRepo string, updateChannel chan<- int) ([]stats.PRsPerDay, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, fmt.Errorf("Repo should be provided as owner/name")
	}

	defer func() {
		if updateChannel != nil {
			close(updateChannel)
		}
	}()

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.PRsPerDay{}
	counter := &Counter{}

	_, repoCreationDate, err := c.GetTotalStars(ctx, ghRepo)
	if err != nil {
		log.Printf("%v\n", err)
		return result, err
	}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.PRsPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	variablesStars := map[string]any{
		"owner":     githubv4.String(owner),
		"name":      githubv4.String(name),
		"prsCursor": (*githubv4.String)(nil),
	}

	type prs struct {
		State     string
		MergedAt  time.Time
		ClosedAt  time.Time
		CreatedAt time.Time
	}

	var queryPRs struct {
		Repository struct {
			PullRequests struct {
				Nodes    []prs
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: 100, orderBy: {field: CREATED_AT, direction: ASC}, after: $prsCursor)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		err := c.query(ctx, &queryPRs, variablesStars)
		if err != nil {
			fmt.Printf("%v\n", err)
			return nil, err
		}

		res := queryPRs.Repository.PullRequests.Nodes

		if len(res) == 0 {
			break
		}

		for _, pr := range res {
			daysOpened := pr.CreatedAt.Sub(repoCreationDate).Hours() / 24

			if daysOpened < 0 {
				continue
			}

			result[int(daysOpened)].Opened++

			if pr.State == "MERGED" {
				if !pr.ClosedAt.IsZero() {
					daysClosed := pr.MergedAt.Sub(repoCreationDate).Hours() / 24
					result[int(daysClosed)].Merged++
				}
			}

			if pr.State == "CLOSED" {
				if !pr.ClosedAt.IsZero() {
					daysClosed := pr.ClosedAt.Sub(repoCreationDate).Hours() / 24
					result[int(daysClosed)].Closed++
				}
			}

			if pr.State == "OPEN" {
				result[int(daysOpened)].CurrentlyOpen++
			}
		}

		if !queryPRs.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}

		variablesStars["prsCursor"] = githubv4.NewString(queryPRs.Repository.PullRequests.PageInfo.EndCursor)

		counter.Increment()

		if updateChannel != nil {
			updateChannel <- counter.Value()
		}
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalOpened = result[i-1].TotalOpened + day.Opened
			result[i].TotalMerged = result[i-1].TotalMerged + day.Merged
			result[i].TotalClosed = result[i-1].TotalClosed + day.Closed
			result[i].TotalCurrentlyOpen = result[i-1].TotalCurrentlyOpen + day.CurrentlyOpen
		} else {
			result[i].TotalOpened = day.Opened
			result[i].TotalMerged = day.Merged
			result[i].TotalClosed = day.Closed
			result[i].TotalCurrentlyOpen = day.CurrentlyOpen
		}
	}

	return result, nil
}

func (c *ClientGQL) GetAllCommitsHistory(ctx context.Context, ghRepo string, updateChannel chan<- int) ([]stats.CommitsPerDay, string, error) {
	repoSplit := strings.Split(ghRepo, "/")

	if len(repoSplit) != 2 || !strings.Contains(ghRepo, "/") {
		return nil, "", fmt.Errorf("Repo should be provided as owner/name")
	}

	defer func() {
		if updateChannel != nil {
			close(updateChannel)
		}
	}()

	owner := repoSplit[0]
	name := repoSplit[1]

	result := []stats.CommitsPerDay{}
	counter := &Counter{}

	_, repoCreationDate, err := c.GetTotalStars(ctx, ghRepo)
	if err != nil {
		log.Printf("%v\n", err)
		return result, "", err
	}

	currentTime := time.Now().UTC().Truncate(24 * time.Hour)
	repoCreationDate = repoCreationDate.Truncate(24 * time.Hour)
	diff := currentTime.Sub(repoCreationDate)
	days := int(diff.Hours()/24 + 1)

	for i := 0; i < days; i++ {
		result = append(result, stats.CommitsPerDay{Day: stats.JSONDay(repoCreationDate.AddDate(0, 0, i).Truncate(24 * time.Hour))})
	}

	type defaultBranch struct {
		Repository struct {
			DefaultBranchRef struct {
				Name githubv4.String
			}
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variablesDefaultBranch := map[string]any{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}

	var queryDefaultBranch defaultBranch
	err = c.query(ctx, &queryDefaultBranch, variablesDefaultBranch)
	if err != nil {
		log.Printf("Error getting default branch: %v\n", err)
		// You can choose to return the error or handle it differently
	}

	defaultBranchName := queryDefaultBranch.Repository.DefaultBranchRef.Name

	log.Printf("Default branch: %s\n", defaultBranchName)

	type commit struct {
		CommittedDate time.Time
	}

	variablesCommits := map[string]any{
		"owner":             githubv4.String(owner),
		"name":              githubv4.String(name),
		"defaultBranchName": githubv4.String(defaultBranchName),
		"commitsCursor":     (*githubv4.String)(nil),
	}

	var queryCommits struct {
		Repository struct {
			Ref struct {
				Target struct {
					Commit struct {
						History struct {
							Nodes    []commit
							PageInfo struct {
								EndCursor   githubv4.String
								HasNextPage bool
							}
						} `graphql:"history(first: 100, after: $commitsCursor)"`
					} `graphql:"... on Commit"`
				} `graphql:"target"`
			} `graphql:"ref(qualifiedName: $defaultBranchName)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	for {
		err := c.query(ctx, &queryCommits, variablesCommits)
		if err != nil {
			fmt.Printf("%v\n", err)
			return nil, "", err
		}

		res := queryCommits.Repository.Ref.Target.Commit.History.Nodes

		if len(res) == 0 {
			break
		}

		for _, commit := range res {
			daysCommitMade := commit.CommittedDate.Sub(repoCreationDate).Hours() / 24

			if daysCommitMade < 0 {
				continue
			}

			result[int(daysCommitMade)].Commits++
		}

		if !queryCommits.Repository.Ref.Target.Commit.History.PageInfo.HasNextPage {
			break
		}

		variablesCommits["commitsCursor"] = githubv4.NewString(queryCommits.Repository.Ref.Target.Commit.History.PageInfo.EndCursor)

		counter.Increment()

		if updateChannel != nil {
			updateChannel <- counter.Value()
		}
	}

	for i, day := range result {
		if i > 0 {
			result[i].TotalCommits = result[i-1].TotalCommits + day.Commits
		} else {
			result[i].TotalCommits = day.Commits
		}
	}
	return result, string(defaultBranchName), nil
}

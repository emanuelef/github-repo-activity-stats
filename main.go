package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/google/go-github/v54/github"
)

// https://api.github.com/repos/jasonrudolph/keyboard

// https://docs.github.com/en/rest/activity/starring?apiVersion=2022-11-28#alternative-response-with-star-creation-timestamps
// https://docs.github.com/en/rest/metrics/statistics?apiVersion=2022-11-28

func main() {
	client := github.NewClient(nil)

	// list all organizations for user "willnorris"
	_, _, err := client.Organizations.List(context.Background(), "willnorris", nil)
	if err != nil {
		log.Fatal("Error Getting List")
	}

	// log.Printf("%v", orgs)

	c := resty.New()

	res := make(map[string]any)

	restyReq := c.R().SetResult(&res)

	_, _ = restyReq.Get("https://api.github.com/repos/jasonrudolph/keyboard")

	fmt.Println("  Stars:", res["stargazers_count"])
	fmt.Println("  Open Issues:", res["open_issues_count"])
	fmt.Println("  Forks:", res["forks_count"])
	fmt.Println("  Archived:", res["archived"])
}

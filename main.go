package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/google/go-github/v54/github"
	"golang.org/x/mod/modfile"
)

// https://api.github.com/repos/jasonrudolph/keyboard

// https://docs.github.com/en/rest/activity/starring?apiVersion=2022-11-28#alternative-response-with-star-creation-timestamps
// https://docs.github.com/en/rest/metrics/statistics?apiVersion=2022-11-28
// https://api.github.com/repos/kubernetes/kubernetes/releases

// https://pkg.go.dev/golang.org/x/mod@v0.5.1/modfile#Require

var file_bytes = []byte(`module module_name

go 1.16

require (
	foo/bar v1.2.3
)
`)

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

	fmt.Println("Stars:", res["stargazers_count"])
	fmt.Println("Open Issues:", res["open_issues_count"])
	fmt.Println("Forks:", res["forks_count"])
	fmt.Println("Archived:", res["archived"])

	f, err := modfile.Parse("go.mod", file_bytes, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(f.Go.Version)

	for _, req := range f.Require {
		fmt.Printf("%s %t\n", req.Mod.Path, req.Indirect)
	}

	resp, err := c.R().Get("https://raw.githubusercontent.com/kubernetes/kubernetes/master/go.mod")

	if err == nil {
		f, err := modfile.Parse("go.mod", resp.Body(), nil)
		if err != nil {
			panic(err)
		}
		fmt.Println(f.Go.Version)

		for _, req := range f.Require {
			// only direct dependencies
			if !req.Indirect {
				fmt.Printf("%s\n", req.Mod.Path)
			}
		}
	}

}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

type myPR struct {
	*github.PullRequest
	Review *github.PullRequestReview `json:"review"`
}

func getGitHubClient() (*github.Client, error) {
	ctx := context.Background()
	token := "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"

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client, nil
}

func filterPullRequests(prs []*github.PullRequest) []*github.PullRequest {
	var filteredPRs []*github.PullRequest
	for _, pr := range prs {
		if pr.GetState() == "open" && pr.GetUser().GetLogin() != "dependabot[bot]" {
			filteredPRs = append(filteredPRs, pr)
		}
	}
	return filteredPRs
}

func lmao() []byte {
	client, err := getGitHubClient()
	if err != nil {
		fmt.Printf("Error creating GitHub client: %v\n", err)
	}

	myPRs := []*myPR{}
	// List of repository full names (e.g., "owner/repo")
	repositories := []string{"RedHatInsights/patchman-ui", "RedHatInsights/vulnerability-ui", "RedHatInsights/insights-dashboard", "RedHatInsights/insights-inventory-frontend", "RedHatInsights/compliance-frontend", "RedHatInsights/insights-advisor-frontend", "RedHatInsights/vuln4shift-frontend", "RedHatInsights/insights-remediations-frontend", "RedHatInsights/frontend-components", "RedHatInsights/ocp-advisor-frontend", "RedHatInsights/drift-frontend", "RedHatInsights/malware-detection-frontend", "RedHatInsights/tasks-frontend"}

	for _, repo := range repositories {
		owner, repoName := parseRepositoryFullName(repo)

		// Fetch pull requests for the repository
		pullRequests, _, err := client.PullRequests.List(context.Background(), owner, repoName, nil)
		if err != nil {
			fmt.Printf("Error fetching pull requests for %s: %v\n", repo, err)
			continue
		}

		pullRequests = filterPullRequests(pullRequests)

		for _, pr := range pullRequests {
			// Fetch reviews for the pull request
			reviews, _, err := client.PullRequests.ListReviews(context.Background(), owner, repoName, *pr.Number, nil)
			if err != nil {
				fmt.Printf("Error fetching reviews for PR %d: %v\n", *pr.Number, err)
				continue
			}
			// allPrs = append(allPrs, pr)

			// create an empty github.PullRequestReview

			fmt.Printf("Reviews for PR #%d in %s:\n", *pr.Number, repo)

			myReview := &github.PullRequestReview{}

			if len(reviews) != 0 {
				myReview = reviews[0]
				for _, review := range reviews {
					if review.GetState() == "APPROVED" || review.GetState() == "CHANGES_REQUESTED" {
						myReview = review
					}
				}

			}
			// myReview print json
			// create myPR struct
			myPR := &myPR{PullRequest: pr}
			myPR.Review = myReview
			myPRs = append(myPRs, myPR)
		}
	}

	sort.Slice(myPRs, func(i, j int) bool {
		return myPRs[i].GetUpdatedAt().After(myPRs[j].GetUpdatedAt())
	})

	jsonData, err := json.Marshal(myPRs)
	if err != nil {
		log.Fatal(err)
	}

	return jsonData
}

var jsonDataa = []byte{}

func runCronJobs() {
	// 3
	s := gocron.NewScheduler(time.UTC)

	// 4
	s.Every(40).Seconds().Do(func() {
		fmt.Println("Running cron job")
		jsonDataa = lmao()
	})
	s.StartAsync()

}

func main() {

	runCronJobs()

	// return jsonData as a server on /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Serving request")
		w.Write(jsonDataa)
	})
	http.ListenAndServe(":8080", nil)
}

func parseRepositoryFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	}
	return
}

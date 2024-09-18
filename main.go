package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

type PullRequestWithReview struct {
	*github.PullRequest
	LatestReview *github.PullRequestReview `json:"review"`
}

func createGitHubClient() (*github.Client, error) {
	ctx := context.Background()
	accessToken := os.Getenv("TOKEN")
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	httpClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(httpClient)
	return client, nil
}

func filterOpenNonDependabotPRs(prs []*github.PullRequest) []*github.PullRequest {
	var filteredPRs []*github.PullRequest
	for _, pr := range prs {
		if pr.GetState() == "open" && pr.GetUser().GetLogin() != "dependabot[bot]" {
			filteredPRs = append(filteredPRs, pr)
		}
	}
	return filteredPRs
}

func fetchLatestReviewForPR(client *github.Client, owner, repoName string, prNumber int) (*github.PullRequestReview, error) {
	reviews, _, err := client.PullRequests.ListReviews(context.Background(), owner, repoName, prNumber, nil)
	if err != nil {
		return nil, err
	}

	var latestReview *github.PullRequestReview

	for _, review := range reviews {
		if review.GetState() == "APPROVED" || review.GetState() == "CHANGES_REQUESTED" {
			latestReview = review
			break
		}
	}

	return latestReview, nil
}

func fetchPullRequestsWithReviews(client *github.Client, repositories []string) []*PullRequestWithReview {
	var pullRequestsWithReviews []*PullRequestWithReview

	for _, repo := range repositories {
		owner, repoName := parseRepositoryFullName(repo)
		pullRequests, _, err := client.PullRequests.List(context.Background(), owner, repoName, nil)
		if err != nil {
			fmt.Printf("Error fetching pull requests for %s: %v\n", repo, err)
			continue
		}

		openNonDependabotPRs := filterOpenNonDependabotPRs(pullRequests)

		for _, pr := range openNonDependabotPRs {
			latestReview, err := fetchLatestReviewForPR(client, owner, repoName, *pr.Number)
			if err != nil {
				fmt.Printf("Error fetching reviews for PR %d: %v\n", *pr.Number, err)
				continue
			}

			pullRequestWithReview := &PullRequestWithReview{
				PullRequest:  pr,
				LatestReview: latestReview,
			}
			pullRequestsWithReviews = append(pullRequestsWithReviews, pullRequestWithReview)
		}
	}

	return pullRequestsWithReviews
}

func runCronJobs() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(10).Minutes().Do(func() {
		fmt.Println("Running cron job")
		data = generateJSONData()
	})
	scheduler.StartAsync()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"

	}
	runCronJobs()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(data)
	})

	fmt.Println("Server listening on :" + port)
	http.ListenAndServe(":"+port, nil)
}

func parseRepositoryFullName(fullName string) (owner, repo string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		owner = parts[0]
		repo = parts[1]
	}
	return
}

var data, _ = json.Marshal([]string{})

func generateJSONData() []byte {
	client, err := createGitHubClient()
	if err != nil {
		log.Printf("Error creating GitHub client: %v\n", err)
		return nil
	}

	repositories := []string{"RedHatInsights/patchman-ui", "RedHatInsights/vulnerability-ui", "RedHatInsights/insights-dashboard", "RedHatInsights/insights-inventory-frontend", "RedHatInsights/compliance-frontend", "RedHatInsights/insights-advisor-frontend", "RedHatInsights/vuln4shift-frontend", "RedHatInsights/insights-remediations-frontend", "RedHatInsights/frontend-components", "RedHatInsights/ocp-advisor-frontend", "RedHatInsights/drift-frontend", "RedHatInsights/malware-detection-frontend", "RedHatInsights/tasks-frontend"}

	pullRequestsWithReviews := fetchPullRequestsWithReviews(client, repositories)

	sort.Slice(pullRequestsWithReviews, func(i, j int) bool {
		return pullRequestsWithReviews[i].GetUpdatedAt().After(pullRequestsWithReviews[j].GetUpdatedAt())
	})

	jsonData, err := json.Marshal(pullRequestsWithReviews)
	if err != nil {
		log.Fatal(err)
	}

	return jsonData
}

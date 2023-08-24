package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/joho/godotenv"
)

const baseURL = "https://api.github.com"

type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
}

type BaseRepo struct {
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
}

type HeadRepo struct {
	Repo struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repo"`
}

type Label struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type Review struct {
	State reviewState `json:"event"`
}

type PullRequest struct {
	ID             int      `json:"id"`
	HTMLURL        string   `json:"html_url"`
	Title          string   `json:"title"`
	User           User     `json:"user"`
	Base           BaseRepo `json:"base"`
	Number         int      `json:"number"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	Labels         []Label  `json:"labels"`
	Draft          bool     `json:"draft"`
	Head           HeadRepo `json:"head"`
	Reviews        []Review `json:"reviews"`
	SelectedReview Review   // New field to store selected review
}

type reviewState string

const (
	Approved         reviewState = "APPROVE"
	ChangesRequested reviewState = "REQUEST_CHANGES"
	Commented        reviewState = "COMMENT"
	EmptyState       reviewState = "EMPTY"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		fmt.Println("GitHub token not found in .env")
		return
	}

	repositories := []string{
		"RedHatInsights/patchman-ui", "RedHatInsights/vulnerability-ui", "RedHatInsights/insights-dashboard", "RedHatInsights/insights-inventory-frontend", "RedHatInsights/compliance-frontend", "RedHatInsights/insights-advisor-frontend", "RedHatInsights/vuln4shift-frontend", "RedHatInsights/insights-remediations-frontend", "RedHatInsights/frontend-components", "RedHatInsights/ocp-advisor-frontend", "RedHatInsights/drift-frontend", "RedHatInsights/malware-detection-frontend", "RedHatInsights/tasks-frontend",
	}

	allPulls := []PullRequest{}
	for _, repoLink := range repositories {
		pulls, err := getPullRequests(repoLink, token)
		if err != nil {
			fmt.Printf("Error fetching pull requests for %s: %s\n", repoLink, err)
			continue
		}

		allPulls = append(allPulls, pulls...)
	}

	// Sort pull requests from all repositories by UpdatedAt timestamp
	sort.SliceStable(allPulls, func(i, j int) bool {
		return allPulls[i].UpdatedAt > allPulls[j].UpdatedAt
	})

	fmt.Printf("Pull requests from all repositories:\n")
	for _, pull := range allPulls {
		fmt.Printf("#%d: %s\n", pull.Number, pull.Title)
		selectedReview, err := getReviewsForPullRequest(pull, token)
		if err != nil {
			fmt.Printf("Error fetching reviews for pull request #%d: %s\n", pull.Number, err)
			continue
		}
		pull.SelectedReview = selectedReview
		if pull.SelectedReview.State != EmptyState {
			fmt.Printf("Selected Review State: %s\n", pull.SelectedReview.State)
		} else {
			// fmt.Println("No significant reviews found.")
		}
	}
}

func getReviewsForPullRequest(pull PullRequest, token string) (Review, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/pulls/%d/reviews", baseURL, pull.Head.Repo.FullName, pull.Number)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return Review{}, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Review{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Review{}, nil
	}

	var reviews []Review
	err = json.NewDecoder(resp.Body).Decode(&reviews)
	if err != nil {
		return Review{}, err
	}

	var selectedReview Review = Review{State: EmptyState}
	for _, review := range reviews {
		if review.State == Approved || review.State == ChangesRequested {
			selectedReview = review
			break
		} else if review.State == Commented && selectedReview.State != Approved && selectedReview.State != ChangesRequested {
			selectedReview = review
		}
	}

	return selectedReview, nil
}

func getPullRequests(repoLink, token string) ([]PullRequest, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/pulls?state=open", baseURL, repoLink)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pulls []PullRequest
	err = json.NewDecoder(resp.Body).Decode(&pulls)
	if err != nil {
		return nil, err
	}

	return pulls, nil
}

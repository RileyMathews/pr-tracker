package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	baseURL = "https://api.github.com"
	perPage = 100
)

type PullRequest struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Draft     bool   `json:"draft"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type IssueComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type ReviewComment struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Path      string `json:"path"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type PullRequestDetails struct {
	PullRequest
	IssueCommentCount  int             `json:"comments"`
	ReviewCommentCount int             `json:"review_comments"`
	IssueComments      []IssueComment  `json:"-"`
	ReviewComments     []ReviewComment `json:"-"`
}

type CommitStatusContext struct {
	Context     string `json:"context"`
	State       string `json:"state"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CheckRun struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	HTMLURL     string `json:"html_url"`
	DetailsURL  string `json:"details_url"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	App         struct {
		Name string `json:"name"`
	} `json:"app"`
}

type PullRequestCIStatuses struct {
	PullRequestNumber int                   `json:"pull_request_number"`
	HeadSHA           string                `json:"head_sha"`
	CombinedState     string                `json:"combined_state"`
	Statuses          []CommitStatusContext `json:"statuses"`
	CheckRuns         []CheckRun            `json:"check_runs"`
}

func FetchOpenPullRequests(repoName, authToken string) ([]PullRequest, error) {
	if strings.TrimSpace(repoName) == "" {
		return nil, errors.New("repo name is required")
	}
	if strings.TrimSpace(authToken) == "" {
		return nil, errors.New("auth token is required")
	}

	httpClient := &http.Client{}
	var allPRs []PullRequest

	nextURL := fmt.Sprintf("%s/repos/%s/pulls?state=open&per_page=%d&page=1", baseURL, repoName, perPage)
	for nextURL != "" {
		var pagePRs []PullRequest
		log.Printf("fetching open prs from: %s", nextURL)
		resp, err := getJSON(httpClient, nextURL, authToken, &pagePRs)
		if err != nil {
			return nil, err
		}

		allPRs = append(allPRs, pagePRs...)
		nextURL = parseNextURL(resp.Header.Get("Link"))
	}

	return allPRs, nil
}

func FetchPullRequestDetails(repoName string, prID int, authToken string) (*PullRequestDetails, error) {
	if strings.TrimSpace(repoName) == "" {
		return nil, errors.New("repo name is required")
	}
	if prID <= 0 {
		return nil, errors.New("pr id must be greater than zero")
	}
	if strings.TrimSpace(authToken) == "" {
		return nil, errors.New("auth token is required")
	}

	httpClient := &http.Client{}
	prDetails := &PullRequestDetails{}

	prURL := fmt.Sprintf("%s/repos/%s/pulls/%d", baseURL, repoName, prID)
	if _, err := getJSON(httpClient, prURL, authToken, prDetails); err != nil {
		return nil, err
	}

	issueCommentsURL := fmt.Sprintf("%s/repos/%s/issues/%d/comments?per_page=%d", baseURL, repoName, prID, perPage)
	issueComments, err := fetchAllIssueComments(httpClient, issueCommentsURL, authToken)
	if err != nil {
		return nil, err
	}
	prDetails.IssueComments = issueComments

	reviewCommentsURL := fmt.Sprintf("%s/repos/%s/pulls/%d/comments?per_page=%d", baseURL, repoName, prID, perPage)
	reviewComments, err := fetchAllReviewComments(httpClient, reviewCommentsURL, authToken)
	if err != nil {
		return nil, err
	}
	prDetails.ReviewComments = reviewComments

	return prDetails, nil
}

func FetchPullRequestCIStatuses(repoName string, prID int, authToken string) (*PullRequestCIStatuses, error) {
	if strings.TrimSpace(repoName) == "" {
		return nil, errors.New("repo name is required")
	}
	if prID <= 0 {
		return nil, errors.New("pr id must be greater than zero")
	}
	if strings.TrimSpace(authToken) == "" {
		return nil, errors.New("auth token is required")
	}

	httpClient := &http.Client{}

	var prResponse struct {
		Number int `json:"number"`
		Head   struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}

	prURL := fmt.Sprintf("%s/repos/%s/pulls/%d", baseURL, repoName, prID)
	if _, err := getJSON(httpClient, prURL, authToken, &prResponse); err != nil {
		return nil, err
	}
	if strings.TrimSpace(prResponse.Head.SHA) == "" {
		return nil, errors.New("pull request head sha is missing")
	}

	var combinedStatus struct {
		State    string                `json:"state"`
		Statuses []CommitStatusContext `json:"statuses"`
	}

	statusURL := fmt.Sprintf("%s/repos/%s/commits/%s/status", baseURL, repoName, prResponse.Head.SHA)
	if _, err := getJSON(httpClient, statusURL, authToken, &combinedStatus); err != nil {
		return nil, err
	}

	checkRunsURL := fmt.Sprintf("%s/repos/%s/commits/%s/check-runs?per_page=%d&page=1", baseURL, repoName, prResponse.Head.SHA, perPage)
	checkRuns, err := fetchAllCheckRuns(httpClient, checkRunsURL, authToken)
	if err != nil {
		return nil, err
	}

	return &PullRequestCIStatuses{
		PullRequestNumber: prResponse.Number,
		HeadSHA:           prResponse.Head.SHA,
		CombinedState:     combinedStatus.State,
		Statuses:          combinedStatus.Statuses,
		CheckRuns:         checkRuns,
	}, nil
}

func fetchAllIssueComments(httpClient *http.Client, firstURL, authToken string) ([]IssueComment, error) {
	nextURL := firstURL
	var allComments []IssueComment

	for nextURL != "" {
		var pageComments []IssueComment
		resp, err := getJSON(httpClient, nextURL, authToken, &pageComments)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, pageComments...)
		nextURL = parseNextURL(resp.Header.Get("Link"))
	}

	return allComments, nil
}

func fetchAllReviewComments(httpClient *http.Client, firstURL, authToken string) ([]ReviewComment, error) {
	nextURL := firstURL
	var allComments []ReviewComment

	for nextURL != "" {
		var pageComments []ReviewComment
		resp, err := getJSON(httpClient, nextURL, authToken, &pageComments)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, pageComments...)
		nextURL = parseNextURL(resp.Header.Get("Link"))
	}

	return allComments, nil
}

func fetchAllCheckRuns(httpClient *http.Client, firstURL, authToken string) ([]CheckRun, error) {
	nextURL := firstURL
	var allCheckRuns []CheckRun

	for nextURL != "" {
		var page struct {
			CheckRuns []CheckRun `json:"check_runs"`
		}

		resp, err := getJSON(httpClient, nextURL, authToken, &page)
		if err != nil {
			return nil, err
		}

		allCheckRuns = append(allCheckRuns, page.CheckRuns...)
		nextURL = parseNextURL(resp.Header.Get("Link"))
	}

	return allCheckRuns, nil
}

func getJSON(httpClient *http.Client, reqURL, authToken string, out any) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "pr-tracker-debug-client")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return nil, fmt.Errorf("github API request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return resp, nil
}

func parseNextURL(linkHeader string) string {
	if strings.TrimSpace(linkHeader) == "" {
		return ""
	}

	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if !strings.Contains(segment, `rel="next"`) {
			continue
		}

		start := strings.Index(segment, "<")
		end := strings.Index(segment, ">")
		if start == -1 || end == -1 || end <= start+1 {
			continue
		}

		return segment[start+1 : end]
	}

	return ""
}

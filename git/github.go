package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

const (
	apiURL   = "https://api.github.com"
	tokenVar = "GITHUB_TOKEN"
)

func CreatePRURL(repo, branch string) string {
	return fmt.Sprintf("https://github.com/%s/pull/new/%s", repo, branch)
}

func CreatePRDescription(results []string) string {
	desc := "Changes applied by commit-cannon:\n"

	for _, result := range results {
		desc += fmt.Sprintf("  * %s\n", result)
	}

	return desc
}

func CreatePR(repo, base, branch, desc string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls", apiURL, repo)
	client := &http.Client{}

	reqBody := map[string]string{
		"title": branch,
		"head":  branch,
		"base":  base,
		"body":  desc,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create JSON body for request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create POST request to GitHub API")
	}

	token := fmt.Sprintf("token %s", os.Getenv(tokenVar))
	req.Header.Add("Authorization", token)
	// Use v3 API
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to create PR for repo %s", repo)
	}

	defer res.Body.Close()

	if res.StatusCode != 201 {
		return "", errors.Errorf("Got %d response from GitHub API", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "Unable to read response body")
	}

	var jsonDict map[string]interface{}
	err = json.Unmarshal(body, &jsonDict)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse JSON from reponse body")
	}

	return jsonDict["html_url"].(string), nil
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
	"github.com/theinventor/oopsie_exceptions-cli/internal/exitcode"
)

type apiProject struct {
	ID               json.Number `json:"id"`
	Name             string      `json:"name"`
	ErrorGroupsCount int         `json:"error_groups_count"`
	UnresolvedCount  int         `json:"unresolved_count"`
	CreatedAt        string      `json:"created_at"`
}

func effectiveProjectInput(cli *client.Client) string {
	if rootProject != "" {
		return rootProject
	}
	if envProject := os.Getenv(client.EnvProject); envProject != "" {
		return envProject
	}
	if cli.ProjectID != "" {
		return cli.ProjectID
	}
	return cli.ProjectName
}

func projectHeaders(cmd *cobra.Command, cli *client.Client) (map[string]string, string, error) {
	projectInput := effectiveProjectInput(cli)
	if projectInput == "" {
		return nil, "", nil
	}
	projectID, err := resolveProjectID(cmd, cli, projectInput)
	if err != nil {
		return nil, "", err
	}
	return map[string]string{"X-Project-Id": projectID}, projectID, nil
}

func resolveProjectID(cmd *cobra.Command, cli *client.Client, input string) (string, error) {
	if isNumericID(input) {
		return input, nil
	}
	projects, _, err := fetchProjects(cmd, cli)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.Name == input {
			return p.ID.String(), nil
		}
	}
	return "", exitcode.Wrap(exitcode.NotFound, fmt.Errorf("project %q is not accessible; run `oopsie project list`", input))
}

func fetchProjects(cmd *cobra.Command, cli *client.Client) ([]apiProject, bool, error) {
	resp, err := cli.Do(http.MethodGet, "/api/v1/project", nil, nil)
	body, readErr := readAPIResponse(cmd, resp, err, http.MethodGet, "/api/v1/project")
	if readErr != nil {
		return nil, false, readErr
	}

	var multi struct {
		Projects []apiProject `json:"projects"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&multi); err != nil {
		return nil, false, exitcode.Wrap(exitcode.Validation, fmt.Errorf("parse /api/v1/project response: %w", err))
	}
	if multi.Projects != nil {
		return multi.Projects, true, nil
	}

	var single apiProject
	dec = json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&single); err != nil {
		return nil, false, exitcode.Wrap(exitcode.Validation, fmt.Errorf("parse /api/v1/project response: %w", err))
	}
	return []apiProject{single}, false, nil
}

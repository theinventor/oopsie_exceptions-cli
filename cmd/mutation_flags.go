package cmd

import "github.com/spf13/cobra"

type mutationFlags struct {
	IdempotencyKey string
	DryRun         bool
}

func bindMutationFlags(c *cobra.Command, mf *mutationFlags) {
	c.Flags().StringVar(&mf.IdempotencyKey, "idempotency-key", "", "caller-supplied token for safe mutation retries")
	c.Flags().BoolVar(&mf.DryRun, "dry-run", false, "print the request envelope without making an HTTP call")
}

func (mf mutationFlags) Headers() map[string]string {
	if mf.IdempotencyKey == "" {
		return nil
	}
	return map[string]string{"Idempotency-Key": mf.IdempotencyKey}
}

type dryRunEnvelope struct {
	DryRun         bool              `json:"dry_run"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Body           any               `json:"body,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	ProjectID      string            `json:"project_id,omitempty"`
}

func newDryRunEnvelope(method, path string, body any, mf mutationFlags, projectID string) dryRunEnvelope {
	return dryRunEnvelope{
		DryRun:         true,
		Method:         method,
		Path:           path,
		Body:           body,
		Headers:        mf.Headers(),
		IdempotencyKey: mf.IdempotencyKey,
		ProjectID:      projectID,
	}
}

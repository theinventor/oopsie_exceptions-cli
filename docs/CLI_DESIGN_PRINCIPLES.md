# CLI Design Principles

## JSON First

Commands that return data emit JSON to stdout by default. Stderr is reserved for diagnostics, hints, and the final command error rendered by `main`.

## Self-Hosted By Default

Oopsie has no canonical hosted API. The CLI requires `--api-url`, `OOPSIE_API_URL`, or a saved profile URL instead of guessing a production endpoint.

## Secret Safe

API keys are masked everywhere. Webhook destinations are masked in server responses and in CLI dry-run output. Saved secrets prefer the OS keychain and fall back to mode-0600 file storage.

## Agent Native

`oopsie agent-context` describes command names, flags, enums, exit codes, profiles, and bundled skill resources in one versioned JSON document.

## Mutation Boundaries

Mutating commands support `--dry-run` and `--idempotency-key`. Dry-run prints the request envelope and exits before making an HTTP call.

## CI Portability

`bin/ci` is the canonical CI contract. The GitHub Actions templates in `docs/github-workflows/` run the same checks once a workflow-scoped publisher moves them into `.github/workflows/`.

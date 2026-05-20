---
name: oopsie
description: Use the Oopsie CLI to inspect and manage self-hosted Oopsie exception trackers.
---

# Oopsie CLI Workflow

Start with `oopsie agent-context` when you need a machine-readable command map. The CLI writes JSON to stdout and diagnostics to stderr.

## Auth

```sh
oopsie auth save --profile prod --api-url https://oopsie.example.com --api-key "$OOPSIE_USER_KEY"
oopsie auth status
oopsie auth use prod
```

Environment overrides:

```sh
export OOPSIE_API_URL=https://oopsie.example.com
export OOPSIE_API_KEY=...
export OOPSIE_PROJECT=my-app
```

Never paste raw API keys or webhook URLs into issue comments, logs, or final answers.

## Workflows

```sh
oopsie whoami
oopsie project list
oopsie error list --project my-app --status unresolved --limit 25
oopsie error get 42 --project my-app
oopsie error resolve 42 --project my-app --dry-run
oopsie error resolve 42 --project my-app --idempotency-key "$(uuidgen)"
printf '%s' "$WEBHOOK_URL" | oopsie notification create --project my-app --channel webhook --url-stdin
oopsie exception create --project my-app --class RuntimeError --message "smoke" --environment production --dry-run
```

User keys need project context for scoped endpoints. Use `--project`, `OOPSIE_PROJECT`, or `oopsie project pin`.

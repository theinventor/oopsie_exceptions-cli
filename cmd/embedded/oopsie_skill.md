---
name: oopsie
description: Use the Oopsie CLI to inspect and manage self-hosted Oopsie exception trackers.
---

# Oopsie CLI Workflow

Use `oopsie agent-context` first when you need a machine-readable command map.
The CLI writes JSON to stdout and diagnostics to stderr.

## Auth

Configure a self-hosted Oopsie instance:

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

## Common Commands

```sh
oopsie whoami
oopsie project list
oopsie project get --project my-app
oopsie error list --project my-app --status unresolved --limit 25
oopsie error get 42 --project my-app
oopsie error resolve 42 --project my-app --dry-run
oopsie error resolve 42 --project my-app --idempotency-key "$(uuidgen)"
oopsie notification list --project my-app
printf '%s' "$WEBHOOK_URL" | oopsie notification create --project my-app --channel webhook --url-stdin
oopsie exception create --project my-app --class RuntimeError --message "smoke" --environment production --dry-run
```

## Notes

- User keys need project context for scoped endpoints. Use `--project`, `OOPSIE_PROJECT`, or `oopsie project pin`.
- Project keys are already scoped and do not need `--project`.
- Mutating commands support `--dry-run` and `--idempotency-key`.
- Dry-run notification creation redacts webhook destinations.

# `oopsie` — JSON-first CLI for Oopsie

`oopsie` is the standalone CLI for self-hosted [Oopsie](https://github.com/theinventor/Oopsie) exception trackers. It replaces the old bash client with a real Go/Cobra command surface built for agents and automation.

Every data command writes JSON to stdout. Diagnostics and command errors go to stderr. Raw API keys and webhook URLs are masked in status, list, and dry-run output.

## Install

### From source

```sh
git clone https://github.com/theinventor/oopsie_exceptions-cli
cd oopsie_exceptions-cli
go build -o ~/.local/bin/oopsie .
```

### Go install

```sh
go install github.com/theinventor/oopsie_exceptions-cli@latest
```

Release archives are built with GoReleaser as `oopsie_<version>_<os>_<arch>`.

## Auth

Oopsie is self-hosted, so there is no silent production default. Set an API URL explicitly by saving a profile or using environment variables.

```sh
oopsie auth save \
  --profile prod \
  --api-url https://oopsie.example.com \
  --api-key "$OOPSIE_USER_KEY" \
  --storage auto

oopsie auth status
oopsie auth use prod
```

Runtime resolution order:

| Order | Source | Notes |
| --- | --- | --- |
| 1 | `--profile <name>` | Explicit profile for one invocation. |
| 2 | `OOPSIE_API_KEY` + `OOPSIE_API_URL` | Environment wins over the saved default. |
| 3 | Saved default profile | Stored in `$XDG_CONFIG_HOME/oopsie/config.json`. |

Secrets are stored in the OS keychain by default when available. Headless systems fall back to a mode-0600 config file. Force a backend with `--storage keychain|file|auto` or `OOPSIE_STORAGE`.

## Project Scoping

Project API keys are already scoped. User API keys can access multiple projects and need a project context for error, notification, and exception endpoints.

Use one of:

```sh
oopsie --project 42 error list
oopsie --project my-app error list
export OOPSIE_PROJECT=my-app
oopsie project pin my-app --profile prod
```

Project names are resolved through `GET /api/v1/project` and then sent as `X-Project-Id`.

## Commands

```text
Global:
  oopsie --profile <name> --project <id-or-name> --api-url <url> <command>

Auth:
  oopsie auth save --profile <name> --api-key <key> --api-url <url> [--project <id-or-name>] [--storage auto|keychain|file]
  oopsie auth status
  oopsie auth list
  oopsie auth use <profile>
  oopsie auth logout [profile]
  oopsie auth migrate --profile <name> | --all

Projects:
  oopsie whoami
  oopsie project list
  oopsie project get [id-or-name]
  oopsie project pin <id-or-name> | --clear

Errors:
  oopsie error list [--status unresolved|resolved|ignored] [--limit N] [--offset N]
  oopsie error get <id> [--limit N]
  oopsie error resolve <id> [--dry-run] [--idempotency-key <key>]
  oopsie error ignore <id> [--dry-run] [--idempotency-key <key>]
  oopsie error reopen <id> [--dry-run] [--idempotency-key <key>]

Notifications:
  oopsie notification list [--channel email|webhook]
  oopsie notification create --channel webhook --url <url> [--events new_error,regression]
  printf '%s' "$WEBHOOK_URL" | oopsie notification create --channel webhook --url-stdin

Exceptions:
  oopsie exception create --class RuntimeError --message boom --environment production
  oopsie exception create --payload-file exception.json
  cat exception.json | oopsie exception create --stdin

Agent discovery:
  oopsie agent-context
  oopsie skill get oopsie
  oopsie version
```

Deprecated aliases from the bash CLI are preserved and hidden from `agent-context`: `oopsie projects`, `oopsie errors`, `oopsie show`, top-level `resolve`/`ignore`/`reopen`, `notifications`, `webhook`, `webhooks`, and `config`.

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | generic error |
| 2 | usage error |
| 3 | authentication or authorization failure |
| 4 | resource not found |
| 5 | validation failed |
| 6 | server error |
| 7 | network or transport failure |
| 8 | conflict |

The same taxonomy is emitted by `oopsie agent-context`.

## Development

```sh
mise trust .mise.toml
./bin/ci
mise x go@1.26.2 -- go test ./...
mise x go@1.26.2 -- go vet ./...
mise x go@1.26.2 -- go build -o ./dist/oopsie .
./dist/oopsie agent-context
```

`docs/github-workflows/` contains GitHub Actions workflow templates for CI and releases. They intentionally live outside `.github/workflows` until a publisher with GitHub's `workflow` scope installs them, because the current automation token cannot push active workflow files.

GoReleaser smoke test:

```sh
goreleaser release --snapshot --clean
```

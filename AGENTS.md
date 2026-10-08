# AGENTS.md

`awsc` is a single-binary Go CLI (Cobra + Bubble Tea) for AWS SSO auth, RDS/EC2/OpenSearch port forwarding via SSM, and Secrets Manager. Module: `github.com/blontic/awsc`. macOS/Linux only.

## Commands

- Build: `make build` (injects version via ldflags; plain `go build` works but omits version vars).
- Test: `make test` (`go test ./...`). Single package: `go test ./internal/aws/`. Single test: `go test ./cmd/ -run TestName`.
- Full dev loop: `make dev` (= mocks + deps + test + build).
- `make check-mod` (`go mod verify` + `go mod tidy -diff`) runs in CI and as GoReleaser's pre-release hook; run `go mod tidy` after any dependency change or both fail.
- After ANY code change, run `go build -o awsc main.go` then `go test ./...` before considering it done. CI (`.github/workflows/ci.yml`) runs `make check-mod`, `make test`, `make vuln` and `make build` on `ubuntu-latest`, with the Go version from `go.mod`.

## Mocks (easy to get wrong)

- Mocks live in `internal/aws/mocks/aws_mocks.go` and ARE committed to git (despite being generated).
- Regenerate with `make mocks` — NOT `go generate`. The target hardcodes the interface list: `RDSClient,EC2Client,SSMClient,SecretsManagerClient,OpenSearchClient`.
- Adding a new AWS service client interface? You must add it to the `mocks` target in the `Makefile` or it won't be mocked.

## Architecture (non-obvious)

- `cmd/` = Cobra commands only, no business logic. All implementation lives in `internal/`.
- `internal/aws/` = per-service "Manager" structs. `internal/config/` = awsc config + orgs (`orgs.go`, `setup.go`), `~/.aws/config` management (`awsconfig.go`, on top of the INI helpers in `ini.go`), terminal sessions (`session.go`), AWS config loading (`load.go`), and the one-time 0.5 migration (`migrate.go`). `internal/ui/` = Bubble Tea selectors. `internal/debug/` = verbose logging.
- Pure AWS SDK Go v2. NO AWS CLI dependency and NO AWS CLI fallback suggestions in errors. The only external binary is `session-manager-plugin`, shelled out via `os/exec` in `internal/aws/externalplugin.go` (SSM sessions/port forwarding).

## Patterns to follow (verify against existing managers, e.g. `internal/aws/rds.go`)

- Manager constructor is always `NewXManager(ctx context.Context, opts ...XManagerOptions)`. Tests pass mock clients via the variadic `opts`; production path (no opts) calls `config.LoadAWSConfigWithProfile(ctx)`. NEVER add a separate `NewXManagerWithClients` test constructor.
- Config loading: `LoadAWSConfig` for SSO/OIDC ops (org's SSO region, no credentials); `LoadAWSConfigWithProfile` for service ops (awsc profile + region).
- Auth errors are handled reactively, never pre-checked. In `cmd/`, create managers with `newManager(ctx, create, switchAccount)` (login if needed, `-s` handling) and report errors with `exitOnError`. In managers, wrap every AWS call in `withReauth(ctx, m.reloadClients, func() (...) { return m.client.Op(...) })` (`internal/aws/reauth.go`): on an auth error it prompts login, reloads clients and retries once. Long-running loops must re-check on each iteration. `IsAuthError` matches error substrings such as `"no active session"`, `"failed to get shared config profile"`, `"ExpiredToken"`, `"InvalidToken"`, `"failed to refresh cached credentials"` (full list in `internal/aws/credentials.go`).
- All list/describe ops MUST paginate (NextToken/Marker loop).
- Every resource command supports interactive selection AND `--name`/`--instance-id` direct access; a direct name that doesn't exist is an error (`namedNotFoundError`), never an interactive fallback, so commands stay scriptable. `-s`/`--switch-account` switches account first.
- Output: stderr for all interactive/status messages and errors (`exitOnError`); stdout only for a command's actual output (e.g. the secret value, `config list`/`show`, `version`). Never leak credentials.
- In `cmd/`, report errors with `exitOnError` (stderr, exit code 1 for scripts); packages return errors instead of exiting.
- Pickers (`internal/ui/selector.go`, Bubble Tea v2) render inline, not full screen: the list is windowed to the terminal size (wrapped lines counted) so the title and context header never scroll off, and a resize clears the screen to avoid stale lines. The header shows the terminal's current context; login pickers pass their own via `RunSelectorWithContext`. Esc clears the filter or quits; `q` is a filter character.
- Browser login (`browserLogin`) shows the device code and waits for Enter before opening the browser; when stdin isn't a terminal it only prints the URL. `~/.aws/config` sync is silent (details only with `-v`).

## State & config (runtime, not in repo)

- Config: `~/.awsc/config.yaml` = `default_org` + `orgs.<name>.{sso.start_url,sso.region,default_region}`. There is no `--config` flag. Before every command except help/version/completion, `config.ActivateOrg` runs (`cmd/root.go`): migrate a 0.5 install, sync `~/.aws/config`, then pick the org (`--org` > `AWSC_ORG` > PPID session's org > `default_org` > only org) and store it in `config.Settings` (`Org`, `StartURL`, `SSORegion`, `DefaultRegion`; `--region` overrides `DefaultRegion`). Code reads `config.Active()`, never the file directly. Viper is not used; don't reintroduce it.
- `migrate.go` is the ONLY code that knows the 0.5 format. Do not add legacy handling elsewhere.
- `~/.aws/config`: one `[sso-session awsc-{org}]` per org plus SSO profiles `awsc-{accountName}/{roleName}` (`awsc-{org}-{accountName}/{roleName}` on cross-org name clash; owned profiles without a `/` use the old naming and are removed by `syncSessions`) — NEVER write static credentials. It is kept in exact sync with the orgs (`syncSessions`): only awsc-owned sections (awsc-* names containing only keys awsc writes) are modified. Writes go through `modifyAWSConfig` (lock, backup, atomic write).
- Per-terminal sessions tracked by PPID in `~/.awsc/sessions/session-{ppid}.json` (includes org). A missing profile is restored from the session; a profile whose account/role no longer matches the session counts as "no active session". SSO token cache: `~/.aws/sso/cache/<sha1("awsc-{org}")>.json` (0600, AWS CLI format with refresh token). Profile selection: `AWSC_PROFILE` env > PPID session > "no active session" error (which auto-triggers login).
- Dirs 0700, sensitive files 0600.

## Testing

- Tests must never touch the real `~/.aws` or `~/.awsc`: use `t.Setenv("HOME", t.TempDir())` (see `setupHome` in `internal/config`). The AWS SDK caches its shared config path at startup, so tests that load a profile through the SDK must also set `sdkconfig.DefaultSharedConfigFiles`.
- Login is stubbed through variables: `promptForReauth` (`internal/aws`) and `reauthFn`/`switchAccountFn` (`cmd`). Pickers are tested via `render()` and `Update`, or `tea.NewProgram` with `WithInput`/`WithWindowSize`.
- Guard tests enforce conventions: no stdout writes in `internal/aws` except the secret value, and no Viper in `go.mod`.

## New command checklist

Interactive + `--name`/`--id` direct mode (error on miss); `NewXManager(ctx, opts...)` constructor; `newManager` in `cmd/` and `withReauth` around every AWS call; paginate all list ops; empty results return `notFoundError(...)` (says which account/role/region was searched); `--switch-account`/`-s` flag; `exitOnError` for errors in `cmd/`; add new client interfaces to the Makefile `mocks` target; update `README.md` with both usage modes.

## Security

- File perms: dirs `0700`, sensitive files `0600`. Validate file paths (traversal). Always nil-check AWS SDK response pointers before deref. Never log/print credentials.

## Conventions source of truth

Verify patterns against existing code before changing them (`internal/aws/rds.go` is the canonical manager; `cmd/root.go` for global flags/wiring). Keep `README.md` in sync when adding/altering commands or flags. (Historical detailed rules lived in `.amazonq/rules/`; their essentials are now folded into this file.)

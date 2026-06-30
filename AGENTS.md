# AGENTS.md

`awsc` is a single-binary Go CLI (Cobra + Bubble Tea) for AWS SSO auth, RDS/EC2/OpenSearch port forwarding via SSM, and Secrets Manager. Module: `github.com/blontic/awsc`. macOS/Linux only.

## Commands

- Build: `make build` (injects version via ldflags; plain `go build` works but omits version vars).
- Test: `make test` (`go test ./...`). Single package: `go test ./internal/aws/`. Single test: `go test ./cmd/ -run TestName`.
- Full dev loop: `make dev` (= mocks + deps + test + build).
- After ANY code change, run `go build -o awsc main.go` then `go test ./...` before considering it done. CI (`.github/workflows/ci.yml`) only runs `make test` + `make build` on `ubuntu-latest`, Go 1.25.

## Mocks (easy to get wrong)

- Mocks live in `internal/aws/mocks/aws_mocks.go` and ARE committed to git (despite being generated).
- Regenerate with `make mocks` — NOT `go generate`. The target hardcodes the interface list: `RDSClient,EC2Client,SSMClient,SecretsManagerClient,OpenSearchClient`.
- Adding a new AWS service client interface? You must add it to the `mocks` target in the `Makefile` or it won't be mocked.

## Architecture (non-obvious)

- `cmd/` = Cobra commands only, no business logic. All implementation lives in `internal/`.
- `internal/aws/` = per-service "Manager" structs. `internal/config/` = config, profiles, sessions, AWS config loading. `internal/ui/` = Bubble Tea selectors. `internal/debug/` = verbose logging.
- Pure AWS SDK Go v2. NO AWS CLI dependency and NO AWS CLI fallback suggestions in errors. The only external binary is `session-manager-plugin`, shelled out via `os/exec` in `internal/aws/externalplugin.go` (SSM sessions/port forwarding).

## Patterns to follow (verify against existing managers, e.g. `internal/aws/rds.go`)

- Manager constructor is always `NewXManager(ctx context.Context, opts ...XManagerOptions)`. Tests pass mock clients via the variadic `opts`; production path (no opts) calls `config.LoadAWSConfigWithProfile(ctx)`. NEVER add a separate `NewXManagerWithClients` test constructor.
- Config loading: `LoadAWSConfig` for SSO ops (region override only); `LoadAWSConfigWithProfile` for service ops (awsc profile + region).
- Auth errors are handled reactively, never pre-checked: call the op, then `if IsAuthError(err)` -> `PromptForReauth(ctx)` -> reload clients (mandatory, with fresh creds) -> retry. Handle this both at manager creation and during every operation. Long-running loops must re-check on each iteration. `IsAuthError` matches: `"no active session"`, `"failed to get shared config profile"`, `"ExpiredToken"`, `"InvalidToken"`, `"failed to refresh cached credentials"`.
- All list/describe ops MUST paginate (NextToken/Marker loop).
- Every resource command supports interactive selection AND `--name`/`--instance-id` direct access, with fallback to interactive on miss. `-s`/`--switch-account` switches account first.
- Output: stderr for all interactive/status messages; stdout ONLY for export commands meant for `eval $(...)`. Never leak credentials.
- In `cmd/`, exit with `os.Exit(1)` on any error (needed for script exit codes); packages return errors instead of exiting.

## State & config (runtime, not in repo)

- Config: `~/.awsc/config.yaml` (viper). Required keys: `sso.start_url`, `sso.region`, `default_region`.
- Profiles written to `~/.aws/config` as `awsc-{accountName}`. Per-terminal sessions tracked by PPID in `~/.awsc/sessions/session-{ppid}.json`. SSO token cache in `~/.aws/sso/cache/` (0600). Profile selection: `AWSC_PROFILE` env > PPID session > "no active session" error (which auto-triggers login).
- Dirs 0700, sensitive files 0600.

## New command checklist

Interactive + `--name`/`--id` direct mode with interactive fallback; `NewXManager(ctx, opts...)` constructor; `IsAuthError`/`PromptForReauth` + mandatory client reload; paginate all list ops; "No [resources] found" on empty; `--switch-account`/`-s` flag; `os.Exit(1)` on errors in `cmd/`; add new client interfaces to the Makefile `mocks` target; update `README.md` with both usage modes.

## Security

- File perms: dirs `0700`, sensitive files `0600`. Validate file paths (traversal). Always nil-check AWS SDK response pointers before deref. Never log/print credentials.

## Conventions source of truth

Verify patterns against existing code before changing them (`internal/aws/rds.go` is the canonical manager; `cmd/root.go` for global flags/wiring). Keep `README.md` in sync when adding/altering commands or flags. (Historical detailed rules lived in `.amazonq/rules/`; their essentials are now folded into this file.)

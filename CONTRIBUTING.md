# Contributing

Read the [architecture](docs/ARCHITECTURE.md) and [PoC scope](docs/POC.md). Keep changes focused, documentation and code in English, and dependency additions tied to a concrete need.

## Build and checks

Use Go 1.27.3+ and Node 24.22.2 with npm 11.19.1 (validated toolchain).

```powershell
./scripts/build.ps1
```

This installs locked frontend dependencies, type-checks/builds the UI, runs Go tests and vet, then builds the executable. It accepts `-GoExe`, `-NodeDir`, and `-SkipInstall` for local toolchains. It does not install Go or Node.

Format frontend changes with `npm run format` inside `web`; CI-style formatting validation is `npm run format:check`.

For individual checks:

```powershell
cd web
npm ci
npm run build
cd ..
go test ./...
go vet ./...
```

Build the UI before `go test ./...` because the executable embeds its output.

## Interface development with hot reload

Install locked frontend dependencies once (`npm ci` in `web`), then run:

```powershell
./scripts/dev.ps1
```

The script accepts `-GoExe`, `-NodeDir` and `-DataDir`. It compiles a separate backend into ignored `work/dev-bin`, starts Vite on an available loopback port, and opens the authenticated browser session. Vue and CSS edits update through Vite HMR. Stop with Ctrl+C and restart for Go changes. If no embedded frontend build exists, the launcher creates one for the Go embed requirement.

By default, development uses isolated `work/dev-data`, so test ideas and keys do not modify the normal app's data. To work with your normal library, stop the normal server first and explicitly pass `-DataDir "$env:APPDATA/CATsoup Trends"`. The normal executable remains a single embedded app and has no Node/Vite runtime dependency.

On other platforms, run `npm run dev` in `web` with Go on PATH. Optional environment settings are `TRENDS_GO_EXE`, `TRENDS_DEV_DATA_DIR` and `TRENDS_DEV_NO_BROWSER=1`. Provider keys retain the same backend-only handling. The development launcher does not print private session URLs. Do not expose Vite on a public network.

The hot-reload test starts a synthetic backend and an isolated Vite instance, temporarily changes one CSS rule, checks that input survives, restores the file, and verifies API authentication and cross-origin rejection.

## Browser tests

The test host uses the real API and SQLite with a synthetic YouTube adapter. It does not require a provider key or publish content.

```powershell
go build -o bin/trends-fixture.exe ./support/testserver
cd web
npx playwright install chromium
npm run test:ui
```

On Windows, an existing Chrome installation can be used instead of downloading Chromium:

```powershell
$env:TRENDS_TEST_BROWSER = "chrome"
npm run test:ui
```

Tests cover search, pagination, persistence across process restart, analysis, export, deletion, filters, provider errors, cancellation, and mobile layout. Artifacts go in ignored `work/`. Browser tests currently target the Windows fixture binary.

## Review expectations

Explain the concrete problem, resulting behavior, and checks performed. Maintain API schemas with endpoint changes. Use meaningful synthetic failure cases for persistence, credentials, provider boundaries, and concurrency. Check current provider policies before adding data collection or analysis.

Do not submit credentials, private research, personal account information, downloaded source media, agent instructions, generated binaries, or local development settings. Use synthetic or explicitly redistributable fixtures. Never print launch tokens or provider credentials in test failures.

Contributions intentionally submitted for inclusion are under [Apache-2.0](LICENSE), unless explicitly agreed otherwise. Submit only material you have the right to contribute.

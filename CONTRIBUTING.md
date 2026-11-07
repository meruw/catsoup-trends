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

# Proof of concept

## Scope delivered

The local workflow is implemented: YouTube search, pagination, optional filters, current source metrics, reference saving, editable creator analysis, and Markdown export for Claude. The application is independent of Publisher and has no model/API subscription dependency.

Explore and Saved are the primary views. Details and analysis open in a side panel. Saved research survives process restart. Notes are manually saved, with visible confirmation and protection against discarding unsaved edits.

The app does not generate a factual video analysis by itself. The user watches the reference, records observations, and exports selected sheets. Claude can then help compare storytelling approaches and draft original ideas; missing observations must remain explicit.

## Validation

Validated on Windows with Go 1.27.3, Node 24.22.2, npm 11.19.1, and Chrome on 2026-11-07.

- Frontend type-check and production build.
- Go unit/integration tests and go vet.
- YouTube response joining, nullable metrics, provider error redaction, and cancellation.
- SQLite persistence, deduplication, note preservation, schema migration, metadata expiry, deletion, and local-workspace query scoping.
- Windows DPAPI storage, replacement, and rejection of corrupt encrypted data.
- API session authentication, Host/Origin rejection, cross-site rejection, request bounds, cached search saving, and export boundaries.
- Two browser scenarios using the real API and SQLite with a synthetic provider: search/pagination/save/analysis/export/restart/delete; filters/quota error/cancel/empty results/mobile layout.
- Visual inspection of Explore, Saved with analysis, and mobile details.
- Runtime frontend dependency audit: no known vulnerabilities reported at the time of the check.

Browser tests intercept thumbnail requests with original synthetic artwork. No real videos, account data, or API keys are included in fixtures.

**Live YouTube search is not yet verified with a user API key.** The remaining acceptance check is to configure a key in Settings, search a real topic, save a reference, watch it, complete a sheet, and export it. Provider quota/access may differ from the synthetic test responses.

## Resource observation

One local run of the production executable, empty database, no provider key; startup includes process launch and the first authenticated status request. Numbers are observations, not performance guarantees.

| Measurement | Result |
| --- | --- |
| Executable | 12,120,064 bytes |
| Startup to status response | 885 ms |
| Backend working set | 50.3 MiB |
| Backend private memory | 47.6 MiB |
| Empty saved-list response, 20 sequential requests | 1.7 ms average |
| Empty SQLite database | 16,384 bytes |
| Frontend JavaScript | about 96 kB; 36.3 kB gzip |
| Frontend CSS | about 11.3 kB; 3.2 kB gzip |
| Shutdown through API | exit code 0 |

Browser memory and real YouTube latency are not included. No large-library performance claim is made. Search caches and provider concurrency are bounded; stored creator notes grow with the reference library. The first working slice establishes a baseline; load benchmarks should accompany larger workloads.

## Limits and next work

- YouTube is the only connector.
- Short-duration search is not a reliable Shorts classifier.
- Current popularity is shown as source metrics, with no virality score or trend forecast.
- Keywords are creator-authored; trending hashtags and audio discovery are not implemented.
- No scraping, media downloads, transcription, local model, or Python worker.
- One local user/workspace. No remote hosting, teams, Publisher integration, or scheduler.
- Metadata cleanup runs only while the process is active and on next launch; user-made backups are outside that cleanup.
- Launch once per session. No installer, single-instance coordinator, or automatic updater.
- No stable release has been created.

The next product decision should follow the live research session: whether saving and annotating references produces a useful script brief. Add quantitative analysis only when the input evidence, provider permissions, and analytical method support it.

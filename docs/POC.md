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
- Windows DPAPI storage, verified encryption before replacement, compatibility with existing encrypted keys, rejection of corrupt/oversized data, and preservation of the previous key plus temporary-file cleanup when replacement fails.
- API session authentication, Host/Origin rejection, cross-site rejection, request bounds, cached search saving, and export boundaries.
- Three browser scenarios using the real API and SQLite with a synthetic provider: search/pagination/save/analysis/individual and batch export/restart/delete; filters/quota error/cancel/empty results/mobile layout; recent/all-time presets, pending filter changes and pagination tied to the applied search. Provider tests verify publication windows and ordering are forwarded correctly.
- Visual inspection of Explore, Saved with analysis, and mobile details.
- Runtime frontend dependency audit: no known vulnerabilities reported at the time of the check.

Browser tests intercept thumbnail requests with original synthetic artwork. No real videos, account data, or API keys are included in fixtures.

**Live YouTube search was verified on November 7, 2026:** the operator searched a real topic and received 20 results with source metrics. The remaining creative acceptance check is to watch a real reference, save it, write observations and assess the exported brief with Claude. Storage, analysis editing, individual/batch export and persistence across restart are covered by isolated synthetic browser tests.

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

Explore defaults to the last 30 days ordered by total views. All-time references retains access to older examples. These are publication-window searches, not measurements of recent view growth or predictions of virality. Changing a preset does not request YouTube data until Search is pressed.

The next product decision should follow the live research session: whether saving and annotating references produces a useful script brief. Add quantitative analysis only when the input evidence, provider permissions, and analytical method support it.

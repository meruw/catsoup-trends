# Architecture

## Implemented local application

Trends is a standalone research product. Publisher, animation tools, and paid models are not runtime dependencies.

| Layer | Implementation |
| --- | --- |
| Interface | Vue 3, TypeScript, Vite build; system fonts and Lucide icons |
| API | Go net/http under /api/v1; same origin as the interface |
| Research | Google Trends RSS, saved format queries, YouTube adapter, reference storage, editable analysis, Markdown export |
| Storage | SQLite, one local workspace, latest metadata per saved reference |
| Packaging | Frontend embedded in one Go executable; Node needed only for builds |

Code lives in `cmd/trends`, `internal/research`, `internal/server`, `internal/secret`, and `web/src`. Synthetic browser-test hosting is isolated in `support/testserver` and is not compiled into the app.

## API and data

The [OpenAPI contract](../api/openapi.json) describes implemented routes. The API returns controlled error codes/messages and never returns the provider key. Reference IDs are validated YouTube IDs, not arbitrary URLs.

A saved reference contains a provider ID, save time, optional current video metadata, a note, and creator-authored analysis fields. Missing metrics are nullable. SQLite queries scope reference access to the fixed local workspace. This is not multi-tenant authentication.

Search joins search.list results with videos.list metadata, preserving provider order. One page is at most 20 results. Search results cache in memory for ten minutes, bounded to 20 pages; save candidates are bounded to 400. New provider lookups have two concurrency slots and a local limit of 20 lookups/minute. Provider quotas remain authoritative. No automatic retry or continuous polling is performed.

Saved lists use pages of 20. References deduplicate by workspace and video ID. Updating metadata preserves notes and analysis. The schema initialization migrates the analysis column without discarding saved notes.

## Analysis and exports

Analysis is written by the user after watching a reference: hook, structure, twist, original adaptation, keywords, and notes. No model, transcript extraction, footage processing, or derived performance score runs inside this PoC.

Claude export includes up to ten references, their source URLs, and user-authored fields, followed by no hidden network operation. It omits provider metadata. The prompt requests original adaptations, ES/EN hook variants, animation effort, and explicit unknowns. Reference notes are marked as quoted research material, not instructions.

Python is reserved for a specific future analytical requirement. If introduced, Go will start a bounded, cancellable worker with structured input/output and no provider secrets. Simple filtering remains in Go/SQL.

## Provider constraints

The under-four-minute API filter includes videos outside Shorts. Language is a relevance preference, not a strict filter. Search ordering and current view counts do not establish current growth or likelihood of future success.

YouTube metadata is kept as one current snapshot, cleared after 29 days unless refreshed. Cleanup runs at startup, on reference reads, and hourly while running. A closed app cannot clean files; reopening performs cleanup before serving saved data. Unavailable refreshed videos lose their metadata while retaining bookmarks and creator notes.

Do not add derived YouTube metrics, historical metric archives, scraping, automated model inputs from API content, or redistribution without checking current provider policies and obtaining any required authorization.

Sources:
- [Search API](https://developers.google.com/youtube/data/reference/rest/v3/search/list)
- [Videos API](https://developers.google.com/youtube/data/reference/rest/v3/videos/list)
- [YouTube developer policies](https://developers.google.com/youtube/terms/developer-policies)

## Local security

The process binds only to 127.0.0.1 on an OS-assigned port. The browser receives a random 256-bit session credential in the launch URL fragment, removes it from the address bar, and keeps it in sessionStorage. API calls use Bearer authentication. The credential expires with the process.

Host must match the listener. Origin, when supplied, must match the local origin; cross-site browser requests are rejected. No wildcard CORS is enabled. CSP limits scripts/styles/connections to the app and thumbnails to YouTube. Responses use no-store and no-referrer.

Windows key storage uses DPAPI scoped to the current user; other operating systems require an environment key. An attacker already running under your Windows user is outside this boundary. No plaintext key fallback is provided.

Provider requests use a fixed endpoint, reject redirects, time out after 20 seconds, and limit response bodies to 2 MiB. API bodies are bounded to 32 KiB, validated, and reject unknown fields. User text is rendered using Vue text bindings, never raw HTML. No generic filesystem or shell endpoint exists.

## Interface

Discover, Explore, Saved and Ideas are the primary destinations. Settings is secondary. Results use compact thumbnail rows; details and analysis open on demand. Fields and controls remain keyboard-accessible. The visual direction follows [unslop](https://github.com/heydamir/unslop): restrained warm surfaces, orange actions, consistent spacing, and no decorative dashboard cards.

## Future boundaries

Publisher integration will use an explicit export or separately scoped API credential; the apps will not share databases or provider credentials. No Publisher client is implemented.

Public hosting needs authentication, authorization, tenant isolation, HTTPS, quotas, backups, secret management, and operational monitoring. The local credential scheme must not be reused as hosted authentication. PostgreSQL and separate workers are possible scaling choices, not PoC dependencies.

The useful local engine, API, UI, and documentation are Apache-2.0. Future operated hosting, collaboration, scheduled workloads, and support may be commercial offerings maintained separately. There is no billing or proprietary service in this repository.

## Discovery and idea workspace

`GET /discover` uses `videos.list` with `chart=mostPopular`, a two-letter country and a supported category. It shares search cache limits, rate limiting, concurrency and save candidates. Opening Discover performs one lookup; changing country/category performs another. There is no scheduler. Provider order and source attribution remain intact. Tags are stored only inside the expiring video metadata snapshot.

The interface displays literal terms per video: provider tags, hashtags and words from title/description. ES/EN stop words, links and duplicates within that source field are excluded; outputs are capped. It does not aggregate terms across channels, calculate frequencies, process footage or claim global trend coverage.

SQLite schema version 2 adds an independent `ideas` table and index; reference rows and existing notes remain intact. Ideas contain creator-written text and at most ten source IDs, with no provider metadata snapshot. Lists use 20-item pages. Search treats the title query literally. Removing a reference preserves its link in an idea but removes its observations from subsequent exports. Source observations are loaded from their current saved reference, not copied into idea storage.

Idea editing has explicit save feedback and guards for unsaved navigation/browser closure. Creator observations can be copied into concept fields with an overwrite confirmation. Export uses saved data and excludes API metadata. The concept may then be developed manually with Claude; no model key or additional runtime dependency is introduced.

## Development runtime

`web/dev.mjs` starts an independently compiled Go backend and Vite, both on OS-assigned loopback ports. Vite serves the source UI with HMR and proxies only `/api/` to Go. A development middleware restricts Host, Origin and Fetch Metadata before forwarding; the proxy rewrites the accepted request's Origin to the backend origin. Bearer authentication remains enforced by Go. Filesystem serving is restricted to the web project and Vite CORS is disabled.

The launch token remains in memory and the initial browser fragment; the runner suppresses backend stdout and never prints it. Vite and the child backend stop together. Development uses an isolated data directory by default. Production neither imports the runner nor requires Node/Vite. Backend source changes require restarting the development runner.


## Topics and saved formats

`GET /topics?region=MX` retrieves the public Google Trends RSS feed through a fixed HTTPS endpoint. The provider adapter uses Go's XML decoder, a 15-second timeout, a 2 MiB response bound, no redirects and a two-letter country validator. External URLs in the feed are ignored; source links are built from the validated region and escaped topic. Rendering uses Vue text bindings, never feed HTML. No new runtime dependency or paid service is introduced.

A separate bounded in-memory cache holds up to 16 countries. Entries are fresh for 10 minutes; provider failures may return the last successful entry for up to 24 hours with `stale: true` and the original observation timestamp. Empty valid feeds remain empty; malformed responses are errors. The same local auth, origin checks, request limiter and provider concurrency limit apply. The endpoint requires no YouTube credential.

Schema version 3 adds a separate `formats` table in the local workspace. Existing reference and idea data are preserved. Each format stores a creator-authored name, YouTube query, language preference, publication window, order, duration, update time and last successful lookup time. Create/update validation reuses the search contract. The workspace is limited to 100 formats; deletion affects only the format. Changing a query or its options clears its previous check timestamp.

`GET /formats/{id}/results` loads the saved query server-side and uses the existing authenticated YouTube search path, cache, rate limiter and reference candidate store. Only the page token comes from the request. No poller, historical metrics archive, derived score or cross-channel keyword ranking is added. Results are current source totals, not evidence of growth. Each view aborts obsolete network lookups; unsaved format edits use the workspace navigation and tab-close guards.

Google Trends source: https://support.google.com/trends/answer/3076011?hl=en. TikTok Creative Center and YouTube Studio are optional outbound links, not imported data sources.

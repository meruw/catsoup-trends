# Architecture

## Implemented local application

Trends is a standalone research product. Publisher, animation tools, and paid models are not runtime dependencies.

| Layer | Implementation |
| --- | --- |
| Interface | Vue 3, TypeScript, Vite build; system fonts and Lucide icons |
| API | Go net/http under /api/v1; same origin as the interface |
| Research | YouTube adapter, reference storage, editable analysis, Markdown export |
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

Explore and Saved are the primary destinations. Settings is secondary. Results use compact thumbnail rows; details and analysis open on demand. Fields and controls remain keyboard-accessible. The visual direction follows [unslop](https://github.com/heydamir/unslop): restrained warm surfaces, orange actions, consistent spacing, and no decorative dashboard cards.

## Future boundaries

Publisher integration will use an explicit export or separately scoped API credential; the apps will not share databases or provider credentials. No Publisher client is implemented.

Public hosting needs authentication, authorization, tenant isolation, HTTPS, quotas, backups, secret management, and operational monitoring. The local credential scheme must not be reused as hosted authentication. PostgreSQL and separate workers are possible scaling choices, not PoC dependencies.

The useful local engine, API, UI, and documentation are Apache-2.0. Future operated hosting, collaboration, scheduled workloads, and support may be commercial offerings maintained separately. There is no billing or proprietary service in this repository.

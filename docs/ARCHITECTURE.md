# Architecture

## Product and boundaries

Trends helps creators find references, examine observable performance, save evidence, and develop original ideas. It does not require Publisher, animation tools, or a CATsoup account.

Publisher may later import selected results through the API or an export. It retains a snapshot so that closing Trends does not affect existing drafts or publishing. The applications never share a database or credentials.

Start with a modular backend, not microservices. Keep business logic independent of HTTP handlers and provider response formats.

| Component | Choice and responsibility |
| --- | --- |
| Web interface | Vue 3 + TypeScript; search, comparisons, saved references, briefs |
| Backend | Go; application services, credentials, connectors, persistence, job coordination |
| API | REST under /api/v1; OpenAPI contract alongside implemented endpoints |
| Local database | SQLite; migrations, transactions, timestamped observations |
| Analysis | Python, introduced when a concrete analytical feature needs it |

The browser calls the API. Application services coordinate persistence and jobs. Jobs call provider adapters or the analysis worker. Publisher is an optional API client.

## Python boundary

Go starts a bounded, cancellable Python process only when an analysis requires it. Use a fixed executable and arguments without a shell. Exchange versioned structured input and output, with diagnostics on a separate channel.

Python receives only necessary analysis inputs, not provider secrets or database ownership. Go validates and persists results. Pin dependencies in a managed environment; decide worker packaging when it is introduced. Simple filtering and sorting remain in Go/SQL. Do not load models at startup.

A separately deployed analysis worker is a future scaling option, not a PoC dependency.

## Data and evidence

- Workspace: ownership boundary for research, configuration, and job results.
- Reference: provider ID, canonical URL, creator, publication time, and available metadata.
- Observation: metric values, observation time, source, and collection method.
- Saved reference: workspace-specific notes and categorization.
- Analysis: algorithm version, inputs, parameters, reasons, and limitations.
- Brief: original concept, adaptation notes, references, and optional language variants.
- Job: progress, cancellation, state, failure reason, and retry metadata.

Missing values remain missing rather than becoming zero. Distinguish historical popularity from current growth: growth requires multiple observations. Compare similar age windows and creator baselines when available. A hashtag associated with successful videos is not evidence of causation.

Verify each provider's access, quotas, storage rules, and redistribution permissions before implementing its connector. Do not promise unrestricted discovery across social platforms. Support manual URL import where useful. Do not download or redistribute reference videos by default.

## Local deployment

A Go executable serves compiled frontend assets and the API from the same origin. Node is needed for building, not as a production server. Store runtime data outside the repository.

Bind only to loopback and let the OS assign an available port. Define a user-protected mechanism for discovering the actual endpoint. Provide an explicit shutdown action; closing a browser tab does not stop a server. No always-on service is required for the PoC.

Bound job concurrency, retries, request sizes, timeouts, caches, and log retention. Persist job state and define restart recovery when jobs are implemented. Avoid continuous polling when the user is not researching.

## Hosted evolution

Reuse the core and frontend, but treat public hosting as a separate delivery milestone. It requires authentication, authorization, tenant isolation tests, HTTPS, quotas, backups, protected secrets, and operational monitoring.

Workspace identifiers establish ownership in the model; they are not sufficient security by themselves. Enforce ownership in every relevant service and query.

PostgreSQL is a candidate for a hosted multi-user edition. Persistence boundaries help migration, but moving from SQLite still requires schema/data migrations and integration tests. Do not maintain two database implementations before there is a concrete need.

## Security

- Authenticate local sessions and optional API clients; loopback is not authentication.
- Validate Host and Origin, protect state-changing requests against CSRF, and avoid wildcard CORS.
- Keep provider credentials in backend-controlled protected storage, never in frontend assets or logs.
- Use a scoped, revocable credential for optional Publisher integration.
- Treat imported URLs as untrusted: restrict schemes/providers and redirects, prevent private-network fetching, and bound response sizes.
- Do not expose arbitrary filesystem operations or command execution.
- Keep public fixtures synthetic and exports free of credentials.
- Define source-specific retention and deletion behavior during connector implementation.

## Interface

Use a compact research workspace with search, results, and on-demand details. Keep saved references and briefs easy to reach. Avoid dense dashboards, decorative cards, and a mandatory numbered wizard. Show provenance and uncertainty where users make comparisons.

Reuse CATsoup design principles without depending on Publisher runtime code.

## Public and commercial scope

The public component should support a useful individual workflow, including a local web interface. Possible paid value includes operated hosting, scheduled workloads, collaboration, storage, and support. Billing and team management are outside the PoC.

The engine, API, local web interface, and repository documentation are licensed under Apache-2.0. Commercial use and proprietary derivatives are permitted subject to the license terms, including attribution and redistribution obligations. These permissions also apply to third parties.

The public project must remain useful on its own. Any future proprietary product would be maintained separately, with its own licensing and dependency review. No proprietary service or paid feature is implemented in this PoC. Third-party content and platform data are not relicensed by this repository.

## References

- [Go HTTP](https://pkg.go.dev/net/http)
- [Vue application approaches](https://vuejs.org/guide/extras/ways-of-using-vue.html)
- [YouTube search API](https://developers.google.com/youtube/v3/docs/search/list)
- [Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)

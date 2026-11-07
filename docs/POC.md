# Proof of concept

## Current state

Documentation only. No application code, dependencies, credentials, remote repository, or release has been created by this scaffold.

The intended engine is public from its first publication. The owner's licensing questions reopened the earlier AGPL preference. Apache-2.0 is now recommended, but no final license has been applied.

## First useful workflow

Search one supported source, compare available evidence, save references, and export an original creative brief. This must work independently of Publisher and without a paid AI API.

YouTube is the initial connector candidate, subject to verifying current access, quotas, permitted storage, and discovery coverage. A short-duration search filter does not guarantee Shorts classification.

## Implementation sequence

1. Verify source constraints and define representative synthetic fixtures.
2. Build one local API/frontend slice: search, pagination, details, and loading/error states.
3. Add saved references and timestamped observations with persistence.
4. Add comparisons with visible evidence and missing-data handling. Introduce Python for a concrete analytical requirement.
5. Add editable briefs and Markdown/JSON exports preserving provenance.
6. Verify the end-to-end workflow, resource usage, failure recovery, and packaging.

## Acceptance criteria

- Search, inspect, save, restart, and recover a reference.
- Show source, available metrics, publication time when provided, and observation time.
- Pagination and cancellation do not trigger unbounded requests or background polling.
- Explain provider rate limits and unavailable data.
- Explain comparison results rather than presenting an unsupported virality score.
- Preserve reference links and distinguish original adaptation from source descriptions.
- Require neither Publisher nor a paid model API for the core workflow.
- Keep credentials and sensitive provider data out of logs, exports, fixtures, and Git history.
- Test persistence, ownership checks, cancellation, provider failures, and URL validation as implemented.
- Measure startup time, idle/active memory, latency, and database/cache growth on a documented workload. Establish budgets from the first measured slice.

## Outside this PoC

Publishing, animation, guaranteed viral predictions, unrestricted scraping, reference-video copying, payments, teams, always-on collection, public hosting, and stable releases.

## Next action

Confirm the public license and publish the repository. Then review the architecture before implementing the first source-backed vertical slice. The current task authorizes repository scaffolding and documentation, not feature implementation.

# CATsoup Trends

A standalone research product for discovering successful short-form content, comparing evidence, and developing original creative briefs.

**Status: architecture and planning.** There is no runnable application, implemented API, or stable release yet.

Trends is independent of CATsoup Publisher. The open-source scope includes the research engine, its API, and a usable local web interface. A possible hosted commercial product is a future decision.

## Planned architecture

| Component | Technology |
| --- | --- |
| Web interface | Vue 3 + TypeScript |
| Backend and API | Go with a versioned REST API |
| Local storage | SQLite |
| Data analysis | On-demand Python workers when analytical features require them |

The first deployment runs locally. Public hosting and multi-user access require additional implementation and security validation.

## Documentation

- [Architecture](docs/ARCHITECTURE.md): technology, boundaries, data, security, and deployment.
- [PoC scope](docs/POC.md): the first useful workflow and acceptance criteria.
- [Contributing](CONTRIBUTING.md): how to propose changes.

## License

Licensed under the [Apache License 2.0](LICENSE). The public engine and local interface may be used in commercial products under its terms. See [NOTICE](NOTICE) for attribution.

The project license does not grant rights to third-party videos, platform data, or trademarks. Provider access and data usage remain subject to the applicable platform terms.

## Data boundaries

Credentials, private research, personal provider data, and production media must remain outside version control. Examples and fixtures must be synthetic or cleared for redistribution.

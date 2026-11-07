# CATsoup Trends

A local research workspace for finding YouTube references and developing original short-animation ideas. Search, inspect available metrics, save videos, write observations, and export a creative brief for Claude.

**Status: working proof of concept.** Windows is the validated platform. No stable release or hosted service is available.

Trends runs independently of CATsoup Publisher. It requires neither Publisher nor a paid AI API.

## Run locally

Requirements for building: Go **1.27.3+**, Node **24.22.2** (tested), and npm **11.19.1** (tested). Node is not needed to run the compiled app.

From the repository root in PowerShell:

```powershell
./scripts/build.ps1
./scripts/launch.ps1
```

The launcher opens your browser. A single Go process serves the interface and API on an available loopback port. Use the power button at the bottom of the sidebar to stop it; closing the browser tab leaves the server running. Launch once per session.

You can also run `bin/CATsoupTrends.exe` directly. Its `-no-browser` flag prints a private launch URL. Do not share that URL: its fragment contains the local session credential.

## Set up YouTube

1. Enable **YouTube Data API v3** in your Google Cloud project.
2. Create an **API key**, rather than OAuth desktop credentials.
3. Restrict the key to YouTube Data API v3. HTTP-referrer restrictions do not work for requests made by this backend; an IP restriction must match your public egress IP.
4. Open **Settings** in Trends, save the key, and perform a search.

On Windows, the saved key is encrypted with DPAPI for your Windows user. Each save verifies decryption before replacing the previous file; temporary files contain ciphertext only and a failed replacement preserves the previous key. Existing DPAPI keys remain readable. Alternatively, set `TRENDS_YOUTUBE_API_KEY` before launch. Environment keys take precedence and cannot be replaced from the interface.

Searches use your Google project quota. Trends makes no paid AI calls. Check your project's current quotas in Google Cloud; no universal quota or request-price assumption is built into the app.

## Research workflow

- **Explore:** search by topic or format, with relevance/views/date sorting, publication windows, language preference, and an optional under-four-minute filter. That duration filter does **not** identify Shorts reliably.
- **Details:** inspect source-provided views, likes, comments, publication date, and observation date. Missing metrics stay missing. Watch the actual video on YouTube.
- **Saved:** keep references and write your own hook, structure, twist, adaptation, keywords, and notes. Fields have explicit save feedback and unsaved-change protection.
- **Export for Claude:** select up to ten saved references and download a Markdown brief. It contains source links and your observations, with a prompt for original ideas and ES/EN hook variants. It does not upload anything to Claude.

Trends does not watch footage, infer retention, predict virality, or rank trending hashtags. Keywords are your labels; generated keyword suggestions in Claude are not verified trends.

## Data and privacy

The default Windows data directory is `%APPDATA%/CATsoup Trends`. It contains `trends.db` and, when configured, `youtube.key`. Override it with `-data-dir`.

Search terms and your API key go to Google. Thumbnail requests go from your browser to YouTube. Videos open on YouTube only when you choose to watch them. Notes and creative analysis remain local until you export or share them yourself.

Only the latest provider metadata is stored. It is cleared after 29 days by startup/read/hourly cleanup while the app runs; if the app is closed, cleanup resumes on its next launch. Your bookmarks and written observations remain. Refresh retrieves current details. Exports omit API titles, descriptions, thumbnails, and metrics. Back up your own research responsibly; backups are not cleaned by the app.

The local server validates Host and Origin and requires a random session credential for API access. It is **not** designed for public hosting or remote access.

## Documentation

- [Architecture and security boundaries](docs/ARCHITECTURE.md)
- [PoC scope, validation, and limitations](docs/POC.md)
- [API contract](api/openapi.json)
- [Contributing and tests](CONTRIBUTING.md)

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) and [third-party notices](THIRD_PARTY_NOTICES.txt). The license does not grant rights to third-party videos, platform data, or trademarks.

Provider use is subject to the [YouTube API policies](https://developers.google.com/youtube/terms/developer-policies), [YouTube terms](https://www.youtube.com/t/terms), and [Google privacy policy](https://policies.google.com/privacy).

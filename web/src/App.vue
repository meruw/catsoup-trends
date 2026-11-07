<script setup lang="ts">
import {
  Search,
  Bookmark,
  Settings,
  SlidersHorizontal,
  X,
  ArrowUpRight,
  Check,
  RefreshCw,
  Power,
  Play,
  Download,
} from '@lucide/vue'
import { count, date, duration } from './api'
import { useWorkspace } from './useWorkspace'
const {
  tab,
  status,
  query,
  savedQuery,
  results,
  references,
  savedIDs,
  exportIDs,
  total,
  nextPage,
  hasMore,
  order,
  period,
  language,
  short,
  showFilters,
  searched,
  cached,
  loading,
  exportBusy,
  error,
  feedback,
  stopped,
  settingsBusy,
  key,
  selected,
  note,
  analysis,
  detailTab,
  detailLoading,
  busyIDs,
  noteBusy,
  filtersCount,
  selectedSaved,
  dirty,
  rows,
  cancel,
  switchTab,
  searchYouTube,
  searchSaved,
  choose,
  save,
  remove,
  saveNote,
  refreshVideo,
  toggleExport,
  exportBrief,
  saveKey,
  removeKey,
  shutdown,
  closeDetails,
} = useWorkspace()
</script>
<template>
  <div v-if="stopped" class="stopped">
    <h1>Trends is closed</h1>
    <p>You can close this tab. Reopen the launcher to continue.</p>
  </div>
  <div v-else class="shell">
    <aside class="sidebar">
      <a class="wordmark" href="/" @click.prevent="switchTab('explore')"
        >CATsoup<span>Trends</span></a
      >
      <nav aria-label="Main navigation">
        <button :class="{ active: tab === 'explore' }" @click="switchTab('explore')">
          <Search /><span>Explore</span>
        </button>
        <button :class="{ active: tab === 'saved' }" @click="switchTab('saved')">
          <Bookmark /><span>Saved</span><span class="nav-count">{{ total }}</span>
        </button>
      </nav>
      <div class="sidebar-bottom">
        <button :class="{ active: tab === 'settings' }" @click="switchTab('settings')">
          <Settings /><span>Settings</span>
        </button>
        <div class="local">
          <span>Running locally</span
          ><button
            class="icon-button"
            title="Stop the local server"
            aria-label="Stop the local server"
            @click="shutdown"
          >
            <Power />
          </button>
        </div>
      </div>
    </aside>
    <main>
      <div v-if="error" role="alert" class="error">
        <span>{{ error }}</span
        ><button class="icon-button" aria-label="Dismiss error" @click="error = ''"><X /></button>
      </div>
      <div class="toast" role="status" aria-live="polite">{{ feedback }}</div>
      <template v-if="tab !== 'settings'">
        <header class="page-header">
          <h1>{{ tab === 'explore' ? 'Explore YouTube' : 'Saved references' }}</h1>
          <span v-if="tab === 'explore'" class="source-label">YouTube Data API</span
          ><button
            v-else
            :disabled="!exportIDs.size || exportBusy"
            title="Export up to 10 selected references with an analysis prompt"
            @click="exportBrief"
          >
            <Download />Export for Claude<span v-if="exportIDs.size">({{ exportIDs.size }})</span>
          </button>
        </header>
        <form
          class="search-bar"
          @submit.prevent="tab === 'explore' ? searchYouTube() : searchSaved()"
        >
          <div class="search-input">
            <Search /><input
              v-if="tab === 'explore'"
              v-model="query"
              aria-label="Search YouTube"
              placeholder="Search a topic, format or idea"
              maxlength="200"
            /><input
              v-else
              v-model="savedQuery"
              aria-label="Search saved references"
              placeholder="Find a saved reference or note"
              maxlength="200"
            />
          </div>
          <button type="submit" class="primary" :disabled="loading">
            {{ tab === 'explore' ? 'Search' : 'Find' }}
          </button>
          <button
            v-if="tab === 'explore'"
            type="button"
            class="filter-toggle"
            :aria-expanded="showFilters"
            aria-controls="search-filters"
            @click="showFilters = !showFilters"
          >
            <SlidersHorizontal />Filters<span v-if="filtersCount" class="filter-count">{{
              filtersCount
            }}</span>
          </button>
          <button v-if="loading" type="button" class="quiet" @click="cancel">Cancel</button>
        </form>
        <form
          v-if="showFilters && tab === 'explore'"
          id="search-filters"
          class="filters"
          @submit.prevent="searchYouTube()"
        >
          <label
            >Sort by<select v-model="order">
              <option value="relevance">Relevance</option>
              <option value="viewCount">View count</option>
              <option value="date">Newest</option>
            </select></label
          >
          <label
            >Published<select v-model="period">
              <option value="any">Any time</option>
              <option value="7">Last 7 days</option>
              <option value="30">Last 30 days</option>
              <option value="90">Last 90 days</option>
            </select></label
          >
          <label
            >Preferred language<select
              v-model="language"
              title="A relevance preference, not a strict language filter"
            >
              <option value="">Any language</option>
              <option value="es">Spanish</option>
              <option value="en">English</option>
            </select></label
          >
          <label class="checkbox"><input v-model="short" type="checkbox" />Under 4 minutes</label
          ><button type="submit">Apply</button>
        </form>
        <p class="search-hint" v-if="tab === 'explore'">
          {{
            short
              ? 'Short videos · under 4 minutes. Includes videos that are not Shorts.'
              : 'All video lengths'
          }}<span v-if="cached && searched"> · Cached results</span>
        </p>
        <p class="search-hint" v-else>Select references to prepare a brief for Claude</p>
        <div v-if="!status?.keyConfigured && tab === 'explore' && status" class="setup-line">
          <span>Connect YouTube search with an API key</span
          ><button @click="switchTab('settings')">Set up YouTube<ArrowUpRight /></button>
        </div>
        <div class="workspace" :class="{ 'with-detail': selected }">
          <section class="results" :aria-busy="loading" aria-label="Video results">
            <div v-if="rows.length" class="list-heading">
              <span>{{
                tab === 'explore'
                  ? results.length + ' results loaded'
                  : references.length + ' shown'
              }}</span
              ><span>Views</span>
            </div>
            <div
              v-for="row in rows"
              :key="row.id"
              class="result-row"
              :class="{ selected: selected?.id === row.id }"
            >
              <input
                v-if="tab === 'saved'"
                class="export-check"
                type="checkbox"
                :checked="exportIDs.has(row.id)"
                :aria-label="'Select for export: ' + (row.video?.title || row.id)"
                @change="toggleExport(row.id)"
              />
              <button
                class="result-content"
                :aria-label="'View details: ' + (row.video?.title || row.id)"
                @click="choose(row)"
              >
                <span class="thumbnail"
                  ><img
                    v-if="row.video?.thumbnail"
                    :src="row.video.thumbnail"
                    alt=""
                    loading="lazy"
                    referrerpolicy="no-referrer"
                  /><Play v-else /><span
                    v-if="row.video && duration(row.video.duration)"
                    class="duration"
                    >{{ duration(row.video.duration) }}</span
                  ></span
                >
                <span class="result-copy"
                  ><span class="video-title">{{
                    row.video?.title || 'Refresh YouTube details'
                  }}</span
                  ><span class="channel">{{
                    row.video?.channel || 'Metadata expired · your bookmark is saved'
                  }}</span
                  ><span class="meta">{{
                    row.video ? date(row.video.publishedAt) : row.id
                  }}</span></span
                >
                <span
                  class="views"
                  :title="row.video?.views?.toLocaleString() || 'View count not available'"
                  >{{ count(row.video?.views) }}</span
                >
              </button>
              <button
                class="save-button icon-button"
                :class="{ saved: savedIDs.has(row.id) }"
                :disabled="busyIDs.has(row.id)"
                :aria-label="savedIDs.has(row.id) ? 'Open saved reference' : 'Save reference'"
                :title="savedIDs.has(row.id) ? 'Saved · open details' : 'Save reference'"
                @click="savedIDs.has(row.id) ? choose(row) : save(row.id)"
              >
                <Bookmark :fill="savedIDs.has(row.id) ? 'currentColor' : 'none'" />
              </button>
            </div>
            <div v-if="!rows.length && !loading" class="empty">
              <template v-if="tab === 'explore'"
                ><Search />
                <h2>{{ searched ? 'No videos found' : 'Start with an idea' }}</h2>
                <p>
                  {{
                    searched
                      ? 'Try a broader topic or adjust your filters.'
                      : 'Search for a topic or story format. Save the videos you want to learn from.'
                  }}
                </p></template
              >
              <template v-else
                ><Bookmark />
                <h2>{{ savedQuery ? 'No matching references' : 'Your reference shelf' }}</h2>
                <p>
                  {{
                    savedQuery
                      ? 'Try a different title, channel or note.'
                      : 'Save a video from Explore to keep its source and add your analysis.'
                  }}
                </p>
                <button v-if="!savedQuery" @click="switchTab('explore')">
                  Explore YouTube<ArrowUpRight /></button
              ></template>
            </div>
            <p v-if="loading" role="status" class="loading">
              Loading {{ tab === 'explore' ? 'YouTube results' : 'references' }}…
            </p>
            <button
              v-if="(tab === 'explore' ? nextPage : hasMore) && !loading"
              class="load-more"
              @click="tab === 'explore' ? searchYouTube(true) : searchSaved(true)"
            >
              Load more
            </button>
          </section>
          <aside v-if="selected" class="detail" aria-label="Reference details">
            <div class="detail-top">
              <span>YouTube reference</span
              ><button class="icon-button" aria-label="Close details" @click="closeDetails">
                <X />
              </button>
            </div>
            <img
              v-if="selected.video?.thumbnail"
              class="detail-image"
              :src="selected.video.thumbnail"
              alt=""
              referrerpolicy="no-referrer"
            />
            <h2>{{ selected.video?.title || 'YouTube details expired' }}</h2>
            <p class="channel">
              {{ selected.video?.channel || 'Refresh to retrieve current metadata' }}
            </p>
            <div class="detail-actions">
              <a
                class="button"
                :href="'https://www.youtube.com/watch?v=' + selected.id"
                target="_blank"
                rel="noopener noreferrer"
                >Watch on YouTube<ArrowUpRight /></a
              ><button
                v-if="!selectedSaved"
                :disabled="busyIDs.has(selected.id)"
                @click="save(selected.id)"
              >
                <Bookmark />Save</button
              ><span v-else class="saved-label"><Check />Saved</span>
            </div>
            <div class="detail-tabs" aria-label="Reference sections">
              <button :aria-pressed="detailTab === 'details'" @click="detailTab = 'details'">
                Details</button
              ><button :aria-pressed="detailTab === 'analysis'" @click="detailTab = 'analysis'">
                Analysis
              </button>
            </div>
            <template v-if="detailTab === 'details'">
              <dl v-if="selected.video" class="facts">
                <div>
                  <dt>Views</dt>
                  <dd>{{ selected.video.views?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Likes</dt>
                  <dd>{{ selected.video.likes?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Comments</dt>
                  <dd>{{ selected.video.comments?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Published</dt>
                  <dd>{{ date(selected.video.publishedAt) }}</dd>
                </div>
                <div>
                  <dt>Observed</dt>
                  <dd>{{ date(selected.video.observedAt) }}</dd>
                </div>
              </dl>
              <details v-if="selected.video?.description" class="description">
                <summary>Description</summary>
                <p>{{ selected.video.description }}</p>
              </details>
            </template>
            <template v-else>
              <p class="analysis-help">
                Your observations after watching the video. Trends does not watch or analyze the
                footage.
              </p>
              <p v-if="!selectedSaved">Save this reference to add your analysis</p>
              <p v-else-if="detailLoading">Loading your analysis…</p>
              <form v-else class="analysis-form" @submit.prevent="saveNote">
                <label
                  >Hook<textarea
                    v-model="analysis.hook"
                    rows="2"
                    maxlength="1000"
                    placeholder="What makes the first seconds worth watching?"
                  ></textarea>
                </label>
                <label
                  >Structure<textarea
                    v-model="analysis.structure"
                    rows="2"
                    maxlength="1000"
                    placeholder="Setup → escalation → payoff"
                  ></textarea>
                </label>
                <label
                  >Twist<textarea
                    v-model="analysis.twist"
                    rows="2"
                    maxlength="1000"
                    placeholder="What changes the viewer's expectation?"
                  ></textarea>
                </label>
                <label
                  >Your adaptation<textarea
                    v-model="analysis.adaptation"
                    rows="3"
                    maxlength="1000"
                    placeholder="An original version for your animation"
                  ></textarea>
                </label>
                <label
                  >Topics and keywords<input
                    v-model="analysis.keywords"
                    maxlength="500"
                    placeholder="Your labels, not a popularity ranking"
                /></label>
                <label
                  >Notes<textarea
                    v-model="note"
                    rows="3"
                    maxlength="1000"
                    placeholder="Other observations"
                  ></textarea>
                </label>
                <button type="submit" :disabled="noteBusy || !dirty">
                  {{ noteBusy ? 'Saving…' : 'Save analysis' }}
                </button>
              </form>
            </template>
            <div v-if="selectedSaved" class="detail-footer">
              <button class="quiet" :disabled="busyIDs.has(selected.id)" @click="refreshVideo">
                <RefreshCw />Refresh details</button
              ><button
                class="quiet danger"
                :disabled="busyIDs.has(selected.id)"
                @click="remove(selected.id)"
              >
                Remove
              </button>
            </div>
          </aside>
        </div>
      </template>
      <section v-else class="settings-page">
        <header class="page-header"><h1>Settings</h1></header>
        <div class="settings-heading">
          <h2>YouTube API key</h2>
          <span :class="{ verified: status?.keyVerified }">{{
            status?.keyVerified
              ? 'Verified this session'
              : status?.keyConfigured
                ? 'Saved · not verified this session'
                : 'Not configured'
          }}</span>
        </div>
        <p>
          Search public videos using your own Google Cloud project. No YouTube sign-in is needed.
        </p>
        <form v-if="status?.canStoreKey" class="key-form" @submit.prevent="saveKey">
          <label for="api-key">{{ status.keyConfigured ? 'Replace API key' : 'API key' }}</label
          ><input
            id="api-key"
            v-model="key"
            type="password"
            autocomplete="off"
            spellcheck="false"
            maxlength="128"
            placeholder="Paste your Google API key"
          />
          <div class="key-actions">
            <button class="primary" :disabled="settingsBusy || !key.trim()" type="submit">
              {{ settingsBusy ? 'Saving…' : 'Save key' }}</button
            ><button
              v-if="status.keyConfigured"
              class="quiet"
              type="button"
              :disabled="settingsBusy"
              @click="removeKey"
            >
              Remove saved key
            </button>
          </div>
          <p class="meta">Encrypted for your Windows account. Never sent to Publisher.</p>
        </form>
        <p v-else-if="status?.keySource === 'environment'">
          The key comes from <code>TRENDS_YOUTUBE_API_KEY</code>. Change that environment variable
          and restart Trends to replace it.
        </p>
        <p v-else-if="status">
          Set <code>TRENDS_YOUTUBE_API_KEY</code> before launching Trends on this operating system.
        </p>
        <details class="setup-help" :open="!status?.keyConfigured">
          <summary>Get an API key</summary>
          <ol>
            <li>
              Enable
              <a
                href="https://console.cloud.google.com/apis/library/youtube.googleapis.com"
                target="_blank"
                rel="noopener noreferrer"
                >YouTube Data API v3</a
              >
              in your Google Cloud project.
            </li>
            <li>
              Open
              <a
                href="https://console.cloud.google.com/apis/credentials"
                target="_blank"
                rel="noopener noreferrer"
                >Credentials</a
              >
              and create an <strong>API key</strong>. OAuth desktop credentials are different.
            </li>
            <li>
              Restrict the key to YouTube Data API v3. Browser-referrer restrictions do not work
              with this backend. An IP restriction needs your current public egress IP.
            </li>
            <li>Save it here, then search to verify access.</li>
          </ol>
        </details>
        <div class="settings-section">
          <h2>Local data</h2>
          <p>
            References, notes and analysis stay on this PC. YouTube metadata expires after 29 days;
            refresh a saved reference to retrieve current details.
          </p>
          <p>
            Searches go to Google. Thumbnails load from YouTube; videos open on YouTube when you
            choose to watch them. Claude exports contain your selected notes and source links,
            without API metrics.
          </p>
          <div class="legal-links">
            <a href="https://www.youtube.com/t/terms" target="_blank" rel="noopener noreferrer"
              >YouTube terms</a
            ><a href="https://policies.google.com/privacy" target="_blank" rel="noopener noreferrer"
              >Google privacy policy</a
            >
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

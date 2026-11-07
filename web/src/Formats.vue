<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Plus, Pencil, RefreshCw, Bookmark, ArrowUpRight, X, ArrowLeft } from '@lucide/vue'
import { api, count, date, duration, type Video, type SearchPage } from './api'
interface Format {
  id: string
  name: string
  query: string
  language: string
  period: string
  order: string
  duration: string
  updatedAt: string
  lastChecked: string
}
const props = defineProps<{ configured: boolean; seed: string }>()
const emit = defineEmits<{ dirty: [value: boolean]; consumed: []; saved: []; settings: [] }>()
const items = ref<Format[]>([]),
  selected = ref<Format | null>(null),
  draft = ref<Format | null>(null),
  snapshot = ref(''),
  filter = ref('')
const videos = ref<Video[]>([]),
  details = ref<Video | null>(null),
  saved = ref(new Set<string>()),
  busyIDs = ref(new Set<string>()),
  next = ref(''),
  cached = ref(false)
const loading = ref(false),
  listing = ref(false),
  busy = ref(false),
  error = ref(''),
  feedback = ref(''),
  checked = ref('')
let controller: AbortController | undefined,
  alive = true
const dirty = computed(() => !!draft.value && JSON.stringify(draft.value) !== snapshot.value)
const visible = computed(() =>
  items.value.filter((f) => f.name.toLocaleLowerCase().includes(filter.value.toLocaleLowerCase())),
)
watch(dirty, (v) => emit('dirty', v))
function ok() {
  return !busy.value && (!dirty.value || confirm('Discard your unsaved format?'))
}
function resetResults() {
  controller?.abort()
  controller = undefined
  loading.value = false
  videos.value = []
  next.value = ''
  details.value = null
  checked.value = ''
}
function newFormat(term = '') {
  if (!ok()) return
  resetResults()
  selected.value = null
  error.value = ''
  feedback.value = ''
  draft.value = {
    id: '',
    name: term.slice(0, 100),
    query: term.slice(0, 200),
    language: '',
    period: '30',
    order: 'viewCount',
    duration: 'short',
    updatedAt: '',
    lastChecked: '',
  }
  snapshot.value = JSON.stringify(draft.value)
}
function edit() {
  if (!selected.value || !ok()) return
  draft.value = { ...selected.value }
  snapshot.value = JSON.stringify(draft.value)
  error.value = ''
  feedback.value = ''
}
function cancelEdit() {
  if (!ok()) return
  draft.value = null
}
async function list() {
  listing.value = true
  try {
    const result = await api<{ items: Format[] }>('/formats')
    if (alive) items.value = result.items
  } catch (e) {
    if (alive) error.value = (e as Error).message
  } finally {
    if (alive) listing.value = false
  }
}
async function choose(f: Format) {
  if (!ok()) return
  draft.value = null
  resetResults()
  selected.value = f
  feedback.value = ''
  await load()
}
async function load(more = false) {
  if (!selected.value || draft.value) return
  if (!props.configured) {
    return
  }
  controller?.abort()
  controller = new AbortController()
  const current = controller,
    id = selected.value.id
  loading.value = true
  error.value = ''
  details.value = null
  if (!more) {
    videos.value = []
    next.value = ''
  }
  try {
    const page = await api<SearchPage>(
      '/formats/' + id + '/results' + (more ? '?page=' + encodeURIComponent(next.value) : ''),
      { signal: current.signal },
    )
    if (current !== controller || !alive) return
    videos.value = [
      ...new Map(
        (more ? [...videos.value, ...page.items] : page.items).map((v) => [v.id, v]),
      ).values(),
    ]
    next.value = page.nextPage
    cached.value = page.cached
    checked.value = new Date().toISOString()
    selected.value!.lastChecked = checked.value
    const listed = items.value.find((f) => f.id === id)
    if (listed) listed.lastChecked = checked.value
    for (const v of page.items) saved.value.delete(v.id)
    for (const id of page.savedIds) saved.value.add(id)
  } catch (e) {
    if (current === controller && !current.signal.aborted) error.value = (e as Error).message
  } finally {
    if (current === controller) loading.value = false
  }
}
async function saveFormat() {
  if (!draft.value) return
  busy.value = true
  error.value = ''
  feedback.value = ''
  try {
    const f = await api<Format>('/formats' + (draft.value.id ? '/' + draft.value.id : ''), {
      method: draft.value.id ? 'PUT' : 'POST',
      body: JSON.stringify({
        name: draft.value.name,
        query: draft.value.query,
        language: draft.value.language,
        period: draft.value.period,
        order: draft.value.order,
        duration: draft.value.duration,
      }),
    })
    if (!alive) return
    draft.value = null
    selected.value = f
    resetResults()
    await list()
    feedback.value = 'Format saved'
    await load()
  } catch (e) {
    if (alive) error.value = (e as Error).message
  } finally {
    if (alive) busy.value = false
  }
}
async function remove() {
  if (
    !selected.value ||
    !ok() ||
    !confirm('Delete this format? Your saved references will remain.')
  )
    return
  busy.value = true
  error.value = ''
  try {
    await api('/formats/' + selected.value.id, { method: 'DELETE' })
    if (!alive) return
    resetResults()
    selected.value = null
    draft.value = null
    await list()
    feedback.value = 'Format deleted'
  } catch (e) {
    if (alive) error.value = (e as Error).message
  } finally {
    if (alive) busy.value = false
  }
}
async function saveVideo(id: string) {
  busyIDs.value.add(id)
  error.value = ''
  try {
    await api('/references', { method: 'POST', body: JSON.stringify({ id }) })
    if (alive) {
      saved.value.add(id)
      feedback.value = 'Reference saved · Open Saved to add your analysis'
      emit('saved')
    }
  } catch (e) {
    if (alive) error.value = (e as Error).message
  } finally {
    busyIDs.value.delete(id)
  }
}
function back() {
  if (!ok()) return
  draft.value = null
  selected.value = null
  resetResults()
}
const summary = computed(() =>
  selected.value
    ? [
        selected.value.period === 'any' ? 'Any time' : `Last ${selected.value.period} days`,
        { viewCount: 'Most viewed', date: 'Newest', relevance: 'Relevance' }[selected.value.order],
        selected.value.duration === 'short' ? 'Under 4 minutes' : 'All lengths',
        selected.value.language
          ? `${selected.value.language.toUpperCase()} preferred`
          : 'Any language',
      ].join(' · ')
    : '',
)
onMounted(async () => {
  await list()
  if (props.seed) {
    newFormat(props.seed)
    emit('consumed')
  }
})
onBeforeUnmount(() => {
  alive = false
  controller?.abort()
  emit('dirty', false)
})
</script>
<template>
  <section class="formats-page">
    <header class="page-header">
      <h1>Formats</h1>
      <button class="primary" :disabled="busy" @click="newFormat()"><Plus />New format</button>
    </header>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p class="format-feedback" role="status">{{ feedback }}</p>
    <div class="format-workspace" :class="{ 'format-open': selected || draft }">
      <aside class="format-list" aria-label="Saved formats">
        <label class="format-search"
          >Find a format<input v-model="filter" type="search" placeholder="Filter formats"
        /></label>
        <button
          v-for="f in visible"
          :key="f.id"
          class="format-item"
          :aria-pressed="selected?.id === f.id"
          :disabled="busy"
          @click="choose(f)"
        >
          <strong>{{ f.name }}</strong
          ><span
            >{{ f.language ? f.language.toUpperCase() : 'Any language' }} ·
            {{ f.lastChecked ? 'Checked ' + date(f.lastChecked) : 'Not checked yet' }}</span
          >
        </button>
        <p v-if="listing" role="status">Loading formats…</p>
        <p v-else-if="!visible.length" class="meta">
          {{ filter ? 'No matching formats' : 'No formats saved yet' }}
        </p>
      </aside>
      <div class="format-content">
        <button v-if="selected || draft" class="quiet formats-back" @click="back">
          <ArrowLeft />All formats
        </button>
        <form v-if="draft" class="format-editor" @submit.prevent="saveFormat">
          <h2>{{ draft.id ? 'Edit format' : 'New format' }}</h2>
          <label
            >Format name<input
              aria-label="Format name"
              v-model="draft.name"
              required
              maxlength="100"
              placeholder="A town with one strange rule"
              :disabled="busy"
          /></label>
          <label
            >YouTube query<input
              aria-label="YouTube query"
              v-model="draft.query"
              required
              maxlength="200"
              placeholder='"en este pueblo" roblox'
              :disabled="busy"
            /><span class="meta"
              >Use a phrase, topic or keywords. This query is saved with the format.</span
            ></label
          >
          <details class="format-options">
            <summary>Search options</summary>
            <div class="format-option-grid">
              <label
                >Preferred language<select
                  aria-label="Preferred language"
                  v-model="draft.language"
                  :disabled="busy"
                >
                  <option value="">Any language</option>
                  <option value="es">Spanish</option>
                  <option value="en">English</option>
                </select></label
              >
              <label
                >Published<select aria-label="Published" v-model="draft.period" :disabled="busy">
                  <option value="7">Last 7 days</option>
                  <option value="30">Last 30 days</option>
                  <option value="90">Last 90 days</option>
                  <option value="any">Any time</option>
                </select></label
              >
              <label
                >Sort by<select aria-label="Sort by" v-model="draft.order" :disabled="busy">
                  <option value="viewCount">Most viewed</option>
                  <option value="date">Newest</option>
                  <option value="relevance">Relevance</option>
                </select></label
              >
              <label
                >Length<select aria-label="Length" v-model="draft.duration" :disabled="busy">
                  <option value="short">Under 4 minutes</option>
                  <option value="any">All lengths</option>
                </select></label
              >
            </div>
          </details>
          <div class="format-editor-actions">
            <button
              type="submit"
              class="primary"
              :disabled="busy || !draft.name.trim() || !draft.query.trim()"
            >
              {{ busy ? 'Saving…' : 'Save format' }}</button
            ><button type="button" class="quiet" :disabled="busy" @click="cancelEdit">Cancel</button
            ><button
              v-if="draft.id"
              type="button"
              class="quiet danger"
              :disabled="busy"
              @click="remove"
            >
              Delete
            </button>
          </div>
        </form>
        <template v-else-if="selected">
          <header class="format-result-header">
            <div>
              <h2>{{ selected.name }}</h2>
              <p class="format-query">{{ selected.query }}</p>
            </div>
            <div>
              <button
                class="icon-button"
                aria-label="Edit format"
                title="Edit format"
                :disabled="busy"
                @click="edit"
              >
                <Pencil /></button
              ><button
                class="icon-button"
                aria-label="Refresh format"
                title="Check this saved query; recent results may come from the 10-minute cache"
                :disabled="loading || busy"
                @click="load()"
              >
                <RefreshCw />
              </button>
            </div>
          </header>
          <p
            class="search-hint"
            title="Language is a relevance preference. Under 4 minutes can include videos that are not Shorts."
          >
            {{ summary }}
          </p>
          <div v-if="!configured" class="setup-line">
            <span>YouTube search needs your API key</span
            ><button @click="emit('settings')">Settings</button>
          </div>
          <div v-if="checked" class="radar-meta">
            <span>{{ videos.length }} results loaded{{ cached ? ' · Cached' : '' }}</span
            ><span title="Total views at lookup time, not growth or a virality score"
              >YouTube · Current totals</span
            >
          </div>
          <div v-for="v in videos" :key="v.id" class="format-video-wrap">
            <div class="format-video-row">
              <button
                class="format-video"
                :aria-expanded="details?.id === v.id"
                :aria-label="'Video details: ' + v.title"
                @click="details = details?.id === v.id ? null : v"
              >
                <span class="thumbnail"
                  ><img
                    v-if="v.thumbnail"
                    :src="v.thumbnail"
                    alt=""
                    loading="lazy"
                    referrerpolicy="no-referrer"
                  /><span class="duration">{{ duration(v.duration) }}</span></span
                >
                <span class="result-copy"
                  ><span class="video-title">{{ v.title }}</span
                  ><span class="meta">{{ v.channel }} · {{ date(v.publishedAt) }}</span></span
                ><span class="views" :title="v.views?.toLocaleString() + ' views'"
                  >{{ count(v.views) }}<small>views</small></span
                ></button
              ><button
                class="icon-button"
                :disabled="saved.has(v.id) || busyIDs.has(v.id)"
                :aria-label="saved.has(v.id) ? 'Reference saved' : 'Save reference'"
                @click="saveVideo(v.id)"
              >
                <Bookmark :fill="saved.has(v.id) ? 'currentColor' : 'none'" />
              </button>
            </div>
            <div v-if="details?.id === v.id" class="format-video-detail">
              <div class="format-detail-top">
                <a
                  :href="'https://www.youtube.com/watch?v=' + v.id"
                  target="_blank"
                  rel="noopener noreferrer"
                  >Watch on YouTube<ArrowUpRight /></a
                ><button
                  class="icon-button"
                  aria-label="Close video details"
                  @click="details = null"
                >
                  <X />
                </button>
              </div>
              <dl class="format-facts">
                <div>
                  <dt>Views</dt>
                  <dd>{{ v.views?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Likes</dt>
                  <dd>{{ v.likes?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Comments</dt>
                  <dd>{{ v.comments?.toLocaleString() ?? 'Not available' }}</dd>
                </div>
                <div>
                  <dt>Observed</dt>
                  <dd>{{ date(v.observedAt) }}</dd>
                </div>
              </dl>
              <p v-if="v.description" class="format-description">{{ v.description }}</p>
            </div>
          </div>
          <p v-if="loading" class="loading" role="status">Checking YouTube…</p>
          <p v-else-if="checked && !videos.length" class="empty">
            No results. Edit the query or try a wider date range.
          </p>
          <button v-if="next && !loading" class="load-more" @click="load(true)">Load more</button>
        </template>
        <div v-else class="empty format-empty">
          <h2>{{ items.length ? 'Choose a format' : 'Keep an eye on your kind of videos' }}</h2>
          <p>
            {{
              items.length
                ? 'Open a saved format to check its latest YouTube results.'
                : 'Save a query once. Return to it for current videos, views and references.'
            }}
          </p>
          <button v-if="!items.length" @click="newFormat()">Create your first format</button>
        </div>
      </div>
    </div>
  </section>
</template>

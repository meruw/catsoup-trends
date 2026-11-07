<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { RefreshCw, ArrowUpRight, Plus } from '@lucide/vue'
import { api, date } from './api'
interface Topic {
  title: string
  traffic: string
  publishedAt: string
  url: string
}
interface Feed {
  items: Topic[]
  region: string
  observedAt: string
  cached: boolean
  stale: boolean
}
const emit = defineEmits<{ search: [term: string]; format: [term: string] }>()
const countries = [
  ['MX', 'Mexico'],
  ['US', 'United States'],
  ['ES', 'Spain'],
  ['AR', 'Argentina'],
  ['CO', 'Colombia'],
  ['GB', 'United Kingdom'],
  ['BR', 'Brazil'],
  ['CA', 'Canada'],
]
const stored = localStorage.getItem('trends.topics.region')
const region = ref(countries.some((c) => c[0] === stored) ? stored! : 'MX')
const feed = ref<Feed | null>(null),
  query = ref(''),
  limit = ref(20),
  loading = ref(false),
  error = ref('')
const filtered = computed(() =>
  (feed.value?.items || []).filter((t) =>
    t.title.toLocaleLowerCase().includes(query.value.trim().toLocaleLowerCase()),
  ),
)
let controller: AbortController | undefined
async function load() {
  controller?.abort()
  controller = new AbortController()
  const current = controller
  loading.value = true
  error.value = ''
  feed.value = null
  limit.value = 20
  localStorage.setItem('trends.topics.region', region.value)
  try {
    const result = await api<Feed>('/topics?region=' + region.value, { signal: current.signal })
    if (current === controller) feed.value = result
  } catch (e) {
    if (!current.signal.aborted) error.value = (e as Error).message
  } finally {
    if (current === controller) loading.value = false
  }
}
onMounted(load)
onBeforeUnmount(() => controller?.abort())
</script>
<template>
  <section class="topics-page">
    <header class="page-header">
      <h1>Popular topics</h1>
      <a
        class="source-label"
        :href="'https://trends.google.com/trending?geo=' + region"
        target="_blank"
        rel="noopener noreferrer"
        >Google Trends <ArrowUpRight
      /></a>
    </header>
    <div class="radar-toolbar">
      <label class="country-control"
        >Country<select aria-label="Country" v-model="region" :disabled="loading" @change="load">
          <option v-for="c in countries" :key="c[0]" :value="c[0]">{{ c[1] }}</option>
        </select></label
      >
      <label class="topic-filter"
        >Filter this list<input
          v-model="query"
          type="search"
          placeholder="Find a topic"
          @input="limit = 20"
      /></label>
      <button
        class="icon-button"
        :disabled="loading"
        aria-label="Refresh topics"
        title="Fetch the latest feed; results are cached for 10 minutes"
        @click="load"
      >
        <RefreshCw />
      </button>
    </div>
    <p class="search-hint">Trending Google searches, not video views or a Shorts ranking</p>
    <p v-if="error" class="error" role="alert">
      {{ error }} <button @click="load">Try again</button>
    </p>
    <p v-if="feed?.stale" class="error" role="status">
      Google is unavailable. Showing the last successful feed from
      {{ new Date(feed.observedAt).toLocaleString() }}.
    </p>
    <div v-if="feed" class="radar-meta">
      <span>{{ filtered.length }} topics · Source order</span
      ><span :title="new Date(feed.observedAt).toLocaleString()"
        >Checked
        {{ new Date(feed.observedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
        }}{{ feed.cached ? ' · Cached' : '' }}</span
      >
    </div>
    <div v-if="filtered.length" class="topic-head">
      <span>Topic</span
      ><span
        title="Approximate search traffic exactly as supplied by Google's RSS feed. Not a video metric."
        >Search traffic</span
      ><span></span>
    </div>
    <article v-for="topic in filtered.slice(0, limit)" :key="topic.title" class="topic-row">
      <div>
        <a :href="topic.url" target="_blank" rel="noopener noreferrer" class="topic-title"
          >{{ topic.title }}<ArrowUpRight /></a
        ><span v-if="topic.publishedAt" class="meta">Listed {{ date(topic.publishedAt) }}</span>
      </div>
      <span
        class="topic-traffic"
        title="Google's approximate search traffic; no growth percentage is provided by this feed"
        >{{ topic.traffic || 'Not supplied' }}</span
      >
      <div class="topic-actions">
        <button class="quiet" @click="emit('search', topic.title)">Find videos</button
        ><button
          class="icon-button"
          :aria-label="'Create format: ' + topic.title"
          title="Create an editable format from this topic"
          @click="emit('format', topic.title)"
        >
          <Plus />
        </button>
      </div>
    </article>
    <p v-if="loading" class="loading" role="status">Loading popular topics…</p>
    <p v-else-if="feed && !filtered.length" class="empty">
      {{ query ? 'No topics match this filter' : 'No topics in this feed. Try another country.' }}
    </p>
    <button v-if="filtered.length > limit" class="load-more" @click="limit += 20">
      Show more topics
    </button>
    <details class="radar-sources">
      <summary>More free trend sources</summary>
      <p>These open the provider's own tools. Their data is not imported automatically.</p>
      <a
        href="https://ads.tiktok.com/business/creativecenter/inspiration/popular/hashtag/pc/en"
        target="_blank"
        rel="noopener noreferrer"
        >TikTok Creative Center <ArrowUpRight /></a
      ><a href="https://studio.youtube.com" target="_blank" rel="noopener noreferrer"
        >YouTube Studio · Analytics → Trends <ArrowUpRight
      /></a>
    </details>
  </section>
</template>

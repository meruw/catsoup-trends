<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { Bookmark, ArrowUpRight, RefreshCw } from '@lucide/vue'
import { api, count, date, type SearchPage, type Video } from './api'
import { previewTerms } from './terms'
const props = defineProps<{ configured: boolean }>()
const emit = defineEmits<{ search: [term: string]; saved: []; settings: [] }>()
const region = ref('MX'),
  category = ref('20'),
  items = ref<Video[]>([]),
  next = ref(''),
  saved = ref(new Set<string>()),
  busy = ref(new Set<string>()),
  loading = ref(false),
  error = ref(''),
  feedback = ref(''),
  applied = ref(''),
  cached = ref(false)
let controller: AbortController | undefined
async function load(more = false) {
  controller?.abort()
  controller = new AbortController()
  const current = controller
  loading.value = true
  error.value = ''
  if (!more) {
    items.value = []
    next.value = ''
    applied.value = new URLSearchParams({
      region: region.value,
      category: category.value,
    }).toString()
  }
  try {
    const data = await api<SearchPage>(
      '/discover?' + applied.value + (more ? '&page=' + encodeURIComponent(next.value) : ''),
      { signal: current.signal },
    )
    if (current !== controller) return
    items.value = [
      ...new Map(
        (more ? [...items.value, ...data.items] : data.items).map((v) => [v.id, v]),
      ).values(),
    ]
    next.value = data.nextPage
    cached.value = data.cached
    for (const v of data.items) saved.value.delete(v.id)
    for (const id of data.savedIds) saved.value.add(id)
  } catch (e) {
    if (!current.signal.aborted)
      error.value = e instanceof Error ? e.message : 'Discovery unavailable'
  } finally {
    if (current === controller) loading.value = false
  }
}
async function save(id: string) {
  busy.value.add(id)
  error.value = ''
  try {
    await api('/references', { method: 'POST', body: JSON.stringify({ id }) })
    saved.value.add(id)
    feedback.value = 'Reference saved'
    emit('saved')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value.delete(id)
  }
}
watch(
  () => props.configured,
  (configured) => {
    if (configured) void load()
    else {
      controller?.abort()
      items.value = []
      next.value = ''
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => controller?.abort())
</script>
<template>
  <section class="discover-page">
    <header class="page-header">
      <h1>Discover</h1>
      <span class="source-label">YouTube popular chart</span>
    </header>
    <div v-if="!configured" class="setup-line">
      Add a YouTube API key to discover videos<button @click="emit('settings')">
        Set up YouTube
      </button>
    </div>
    <template v-else>
      <form class="discovery-controls" @submit.prevent="load()">
        <label
          >Country<select v-model="region" :disabled="loading" @change="load()">
            <option value="MX">Mexico</option>
            <option value="US">United States</option>
            <option value="ES">Spain</option>
            <option value="AR">Argentina</option>
            <option value="CO">Colombia</option>
            <option value="GB">United Kingdom</option>
            <option value="BR">Brazil</option>
            <option value="CA">Canada</option>
          </select></label
        >
        <label
          >Category<select v-model="category" :disabled="loading" @change="load()">
            <option value="20">Gaming</option>
            <option value="10">Music</option>
            <option value="24">Entertainment</option>
            <option value="0">All chart categories</option>
          </select></label
        >
        <button
          :disabled="loading"
          type="submit"
          title="Reload the chart; recent results may come from the 10-minute cache"
        >
          <RefreshCw />Reload
        </button>
      </form>
      <p class="search-hint">
        Terms from popular videos, grouped by source. Not search volume or a keyword ranking.
      </p>
      <details class="discovery-scope">
        <summary>Chart coverage{{ cached ? ' · Cached' : '' }}</summary>
        <p>
          YouTube supplies this chart from music, movies and gaming. It can include long videos and
          is not a Shorts chart. Some country/category combinations have no chart. Terms shown below
          are literal metadata; they are not combined into a cross-channel ranking.
        </p>
      </details>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p class="meta" role="status">{{ feedback }}</p>
      <article v-for="video in items" :key="video.id" class="discovery-row">
        <a
          :href="'https://www.youtube.com/watch?v=' + video.id"
          target="_blank"
          rel="noopener noreferrer"
          class="discovery-thumb"
          ><img
            v-if="video.thumbnail"
            :src="video.thumbnail"
            alt=""
            loading="lazy"
            referrerpolicy="no-referrer"
        /></a>
        <div class="discovery-copy">
          <a
            class="video-title"
            :href="'https://www.youtube.com/watch?v=' + video.id"
            target="_blank"
            rel="noopener noreferrer"
            >{{ video.title }}<ArrowUpRight
          /></a>
          <p class="meta">
            {{ video.channel }} · {{ count(video.views) }} views · {{ date(video.publishedAt) }}
          </p>
          <div class="term-list">
            <button
              v-for="term in previewTerms(video)"
              :key="term"
              :title="'Search YouTube for ' + term"
              @click="emit('search', term)"
            >
              {{ term }}
            </button>
          </div>
        </div>
        <button
          class="icon-button"
          :disabled="saved.has(video.id) || busy.has(video.id)"
          :aria-label="saved.has(video.id) ? 'Reference saved' : 'Save reference'"
          @click="save(video.id)"
        >
          <Bookmark :fill="saved.has(video.id) ? 'currentColor' : 'none'" />
        </button>
      </article>
      <p v-if="loading" role="status">Loading YouTube chart…</p>
      <p v-else-if="!items.length && !error" class="empty">
        No videos in this chart. Try another country or category.
      </p>
      <button v-if="next && !loading" class="load-more" @click="load(true)">Load more</button>
    </template>
  </section>
</template>

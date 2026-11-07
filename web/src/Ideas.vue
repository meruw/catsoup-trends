<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Plus, Download, X, ArrowUpRight } from '@lucide/vue'
import { api, APIError, date, downloadMarkdown, type Idea, type Reference } from './api'
const props = defineProps<{ seed: string[] }>()
const emit = defineEmits<{ dirty: [value: boolean]; consumed: []; savedReferences: [] }>()
const items = ref<Idea[]>([]),
  query = ref(''),
  applied = ref(''),
  more = ref(false),
  loading = ref(false),
  busy = ref(false),
  error = ref(''),
  feedback = ref(''),
  draft = ref<Idea | null>(null),
  snapshot = ref(''),
  sources = ref<Reference[]>([]),
  sourceLoading = ref(false)
let alive = true,
  sourceVersion = 0,
  listVersion = 0
const dirty = computed(() => !!draft.value && JSON.stringify(draft.value) !== snapshot.value)
watch(dirty, (v) => emit('dirty', v))
function ok() {
  if (busy.value) return false
  return !dirty.value || confirm('Discard this unsaved idea?')
}
async function list(append = false) {
  const version = ++listVersion
  loading.value = true
  error.value = ''
  if (!append) applied.value = query.value.trim()
  try {
    const p = await api<{ items: Idea[]; hasMore: boolean }>(
      '/ideas?' +
        new URLSearchParams({ q: applied.value, offset: String(append ? items.value.length : 0) }),
    )
    if (!alive || version !== listVersion) return
    items.value = append ? [...items.value, ...p.items] : p.items
    more.value = p.hasMore
  } catch (e) {
    if (version === listVersion) error.value = (e as Error).message
  } finally {
    if (version === listVersion) loading.value = false
  }
}
async function readSources(ids: string[]) {
  const version = ++sourceVersion
  sources.value = []
  sourceLoading.value = true
  try {
    const found = await Promise.all(
      ids.map(async (id) => {
        try {
          return await api<Reference>('/references/' + id)
        } catch (e) {
          if (!(e instanceof APIError) || e.code !== 'not_found') throw e
          return {
            id,
            video: null,
            note: '',
            analysis: { hook: '', structure: '', twist: '', adaptation: '', keywords: '' },
            savedAt: '',
          } as Reference
        }
      }),
    )
    if (alive && version === sourceVersion) sources.value = found
  } catch (e) {
    if (alive && version === sourceVersion) error.value = (e as Error).message
  } finally {
    if (version === sourceVersion) sourceLoading.value = false
  }
}
function create(ids: string[] = []) {
  if (!ok()) return
  const blank: Idea = {
    id: '',
    title: '',
    sources: [...ids],
    hook: '',
    structure: '',
    twist: '',
    world: '',
    notes: '',
    updatedAt: '',
  }
  draft.value = blank
  snapshot.value = JSON.stringify({ ...blank, sources: [] })
  void readSources(ids)
  feedback.value = ''
}
function select(i: Idea) {
  if (!ok()) return
  draft.value = JSON.parse(JSON.stringify(i)) as Idea
  snapshot.value = JSON.stringify(draft.value)
  void readSources(i.sources)
  feedback.value = ''
}
function close() {
  if (ok()) {
    draft.value = null
    sources.value = []
    sourceVersion++
  }
}
function use(field: 'hook' | 'structure' | 'twist', text: string) {
  if (!draft.value || !text) return
  if (draft.value[field] && !confirm('Replace the current ' + field + ' with this observation?'))
    return
  draft.value[field] = text
}
function detach(id: string) {
  if (!draft.value) return
  draft.value.sources = draft.value.sources.filter((v) => v !== id)
  sources.value = sources.value.filter((v) => v.id !== id)
}
async function save() {
  if (!draft.value || busy.value) return
  busy.value = true
  error.value = ''
  const input = JSON.parse(JSON.stringify(draft.value)) as Idea
  try {
    const saved = await api<Idea>('/ideas' + (input.id ? '/' + input.id : ''), {
      method: input.id ? 'PUT' : 'POST',
      body: JSON.stringify(input),
    })
    draft.value = saved
    snapshot.value = JSON.stringify(saved)
    feedback.value = 'Idea saved'
    await list()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
async function remove() {
  if (!draft.value?.id || !confirm('Delete this idea? Saved references will remain.')) return
  busy.value = true
  try {
    await api('/ideas/' + draft.value.id, { method: 'DELETE' })
    draft.value = null
    sources.value = []
    feedback.value = 'Idea deleted'
    await list()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
async function exportIdea() {
  if (!draft.value?.id || dirty.value) return
  busy.value = true
  error.value = ''
  try {
    const result = await api<{ markdown: string }>('/ideas/' + draft.value.id + '/export', {
      method: 'POST',
    })
    downloadMarkdown(result.markdown, 'catsoup-idea.md')
    feedback.value = 'Idea exported for Claude'
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
onMounted(() => {
  void list()
  if (props.seed.length) {
    create(props.seed)
    emit('consumed')
  }
})
onBeforeUnmount(() => {
  alive = false
  sourceVersion++
  emit('dirty', false)
})
</script>
<template>
  <section class="ideas-page">
    <header class="page-header">
      <h1>Ideas</h1>
      <button :disabled="busy" @click="create()"><Plus />New idea</button>
    </header>
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p class="meta" role="status">{{ feedback }}</p>
    <div class="idea-workspace" :class="{ 'has-editor': draft }">
      <section class="idea-list" aria-label="Saved ideas">
        <form class="search-bar" @submit.prevent="list()">
          <input
            v-model="query"
            aria-label="Find ideas"
            placeholder="Find an idea"
            maxlength="200"
          /><button :disabled="loading">Find</button>
        </form>
        <button
          v-for="idea in items"
          :key="idea.id"
          class="idea-row"
          :class="{ selected: draft?.id === idea.id }"
          @click="select(idea)"
        >
          <strong>{{ idea.title }}</strong
          ><span class="meta"
            >{{ idea.sources.length }} references · {{ date(idea.updatedAt) }}</span
          >
        </button>
        <p v-if="loading" role="status">Loading ideas…</p>
        <div v-else-if="!items.length" class="empty">
          <h2>{{ query ? 'No matching ideas' : 'Keep the next story here' }}</h2>
          <p>
            Select references in Saved and choose Combine into idea, or start with your own concept.
          </p>
          <button v-if="!query" @click="emit('savedReferences')">Choose references</button>
        </div>
        <button v-if="more && !loading" class="load-more" @click="list(true)">Load more</button>
      </section>
      <section v-if="draft" class="idea-editor" aria-label="Idea editor">
        <div class="detail-top">
          <span>{{ draft.id ? 'Edit idea' : 'New idea' }}<span v-if="dirty"> · Unsaved</span></span
          ><button class="icon-button" :disabled="busy" aria-label="Close idea" @click="close">
            <X />
          </button>
        </div>
        <form class="analysis-form" @submit.prevent="save">
          <fieldset :disabled="busy">
            <label
              >Idea title<input
                v-model="draft.title"
                maxlength="160"
                required
                placeholder="Give this story a name"
            /></label>
            <label
              >World and constraints<input
                v-model="draft.world"
                maxlength="3000"
                placeholder="Block animation, two characters, 20 seconds…"
            /></label>
            <details v-if="draft.sources.length" class="idea-sources" open>
              <summary>Reference observations ({{ draft.sources.length }})</summary>
              <p v-if="sourceLoading">Loading observations…</p>
              <article v-for="source in sources" :key="source.id" class="idea-source">
                <div class="source-heading">
                  <a
                    :href="'https://www.youtube.com/watch?v=' + source.id"
                    target="_blank"
                    rel="noopener noreferrer"
                    >{{ source.video?.title || source.id }}<ArrowUpRight /></a
                  ><button
                    type="button"
                    class="quiet"
                    :aria-label="'Detach source ' + source.id"
                    @click="detach(source.id)"
                  >
                    Remove
                  </button>
                </div>
                <dl>
                  <template v-for="field in ['hook', 'structure', 'twist'] as const" :key="field"
                    ><div v-if="source.analysis[field]">
                      <dt>{{ field }}</dt>
                      <dd>{{ source.analysis[field] }}</dd>
                      <button
                        type="button"
                        class="quiet"
                        @click="use(field, source.analysis[field])"
                      >
                        Use {{ field }}
                      </button>
                    </div></template
                  >
                </dl>
                <p v-if="source.analysis.adaptation" class="source-note">
                  {{ source.analysis.adaptation }}
                </p>
                <p
                  v-if="
                    !source.analysis.hook && !source.analysis.structure && !source.analysis.twist
                  "
                  class="meta"
                >
                  No saved hook, structure or twist. Add your observations in Saved.
                </p>
              </article>
            </details>
            <label
              >Opening hook<textarea
                v-model="draft.hook"
                rows="2"
                maxlength="3000"
                placeholder="What will make someone stop scrolling?"
              />
            </label>
            <label
              >Story outline<textarea
                v-model="draft.structure"
                rows="3"
                maxlength="3000"
                placeholder="Setup → escalation → payoff"
              />
            </label>
            <label
              >Original twist<textarea
                v-model="draft.twist"
                rows="2"
                maxlength="3000"
                placeholder="What makes your version different?"
              />
            </label>
            <label
              >Notes<textarea
                v-model="draft.notes"
                rows="3"
                maxlength="3000"
                placeholder="Dialogue, ES/EN variants, or a concept developed with Claude"
              />
            </label>
            <div class="idea-actions">
              <button
                class="primary"
                type="submit"
                :disabled="!draft.title.trim() || (!dirty && !!draft.id)"
              >
                {{ busy ? 'Working…' : 'Save idea' }}</button
              ><button
                type="button"
                :disabled="!draft.id || dirty"
                :title="
                  dirty
                    ? 'Save the idea before exporting'
                    : 'Export your concept, observations and source links'
                "
                @click="exportIdea"
              >
                <Download />Export for Claude
              </button>
            </div>
          </fieldset>
        </form>
        <button v-if="draft.id" class="quiet danger" :disabled="busy" @click="remove">
          Delete idea
        </button>
      </section>
    </div>
  </section>
</template>

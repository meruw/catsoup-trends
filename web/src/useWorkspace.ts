import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import {
  api,
  APIError,
  emptyAnalysis,
  type Video,
  type Reference,
  type Status,
  type SearchPage,
  type SavedPage,
  type Analysis,
} from './api'
export function useWorkspace() {
  const tab = ref<'explore' | 'saved' | 'settings' | 'discover' | 'ideas'>('explore'),
    status = ref<Status>()
  const ideaDirty = ref(false),
    ideaSeed = ref<string[]>([])
  const query = ref(''),
    savedQuery = ref(''),
    results = ref<Video[]>([]),
    references = ref<Reference[]>([])
  const savedIDs = ref(new Set<string>()),
    exportIDs = ref(new Set<string>()),
    total = ref(0),
    nextPage = ref(''),
    hasMore = ref(false)
  const order = ref('viewCount'),
    period = ref('30'),
    language = ref(''),
    short = ref(true)
  const showFilters = ref(false),
    searched = ref(false),
    cached = ref(false),
    loading = ref(false),
    exportBusy = ref(false)
  const error = ref(''),
    feedback = ref(''),
    stopped = ref(false),
    settingsBusy = ref(false),
    key = ref('')
  const selected = ref<Reference | null>(null),
    note = ref(''),
    analysis = ref<Analysis>(emptyAnalysis()),
    detailTab = ref<'details' | 'analysis'>('details')
  const busyIDs = ref(new Set<string>()),
    noteBusy = ref(false),
    detailLoading = ref(false)
  let controller: AbortController | null = null,
    sequence = 0,
    timer: ReturnType<typeof setTimeout> | undefined,
    appliedSaved = ''
  const appliedSearch = ref('')
  const searchParameters = computed(() =>
    new URLSearchParams({
      q: query.value.trim(),
      order: order.value,
      period: period.value,
      language: language.value,
      duration: short.value ? 'short' : 'any',
    }).toString(),
  )
  const searchPending = computed(
    () => searched.value && appliedSearch.value !== searchParameters.value,
  )
  const preset = computed(() =>
    order.value === 'viewCount' && short.value
      ? period.value === '30'
        ? 'recent'
        : period.value === 'any'
          ? 'all'
          : ''
      : '',
  )
  const filtersCount = computed(
    () =>
      Number(order.value !== 'viewCount') +
      Number(period.value !== '30') +
      Number(language.value !== '') +
      Number(!short.value),
  )
  const searchSummary = computed(() => {
    const params = new URLSearchParams(
      searched.value ? appliedSearch.value : searchParameters.value,
    )
    const days = params.get('period')
    const sort = { relevance: 'Relevance', viewCount: 'Most viewed', date: 'Newest' }[
      params.get('order') || ''
    ]
    const lang = { es: 'Spanish preferred', en: 'English preferred' }[params.get('language') || '']
    return [
      days === 'any' ? 'Any time' : `Last ${days} days`,
      sort,
      params.get('duration') === 'short' ? 'Under 4 minutes' : 'All lengths',
      lang,
    ]
      .filter(Boolean)
      .join(' · ')
  })
  function setPreset(value: 'recent' | 'all') {
    order.value = 'viewCount'
    period.value = value === 'recent' ? '30' : 'any'
    short.value = true
  }
  const selectedSaved = computed(() => !!selected.value && savedIDs.value.has(selected.value.id))
  const dirty = computed(
    () =>
      selectedSaved.value &&
      !!selected.value &&
      (note.value !== selected.value.note ||
        JSON.stringify(analysis.value) !== JSON.stringify(selected.value.analysis)),
  )
  const rows = computed(() =>
    tab.value === 'saved'
      ? references.value
      : results.value.map((video) => ({
          id: video.id,
          video,
          note: '',
          analysis: emptyAnalysis(),
          savedAt: '',
        })),
  )
  function tell(text: string) {
    feedback.value = text
    clearTimeout(timer)
    timer = setTimeout(() => (feedback.value = ''), 4500)
  }
  function problem(e: unknown) {
    if (e instanceof DOMException && e.name === 'AbortError') return
    error.value = e instanceof Error ? e.message : 'Something went wrong'
  }
  function cancel() {
    sequence++
    controller?.abort()
    controller = null
    loading.value = false
  }
  async function loadStatus() {
    status.value = await api<Status>('/status')
  }
  async function loadCount() {
    total.value = (await api<SavedPage>('/references')).total
  }
  function discardOK() {
    return !(dirty.value || ideaDirty.value) || confirm('Discard your unsaved changes?')
  }
  function closeDetails() {
    if (discardOK()) selected.value = null
  }
  function switchTab(value: typeof tab.value) {
    if (!discardOK()) return
    cancel()
    selected.value = null
    tab.value = value
    feedback.value = ''
    error.value = ''
    showFilters.value = false
    if (value === 'saved') void searchSaved()
  }
  async function searchYouTube(more = false) {
    if (!discardOK()) return
    cancel()
    error.value = ''
    selected.value = null
    if (!query.value.trim() && !more) return
    if (!status.value?.keyConfigured) {
      tab.value = 'settings'
      tell('Add your YouTube API key to search')
      return
    }
    if (!more) {
      appliedSearch.value = searchParameters.value
      results.value = []
      nextPage.value = ''
    }
    const request = ++sequence
    controller = new AbortController()
    loading.value = true
    searched.value = true
    showFilters.value = false
    try {
      const page = await api<SearchPage>(
        '/search?' +
          appliedSearch.value +
          (more ? '&page=' + encodeURIComponent(nextPage.value) : ''),
        { signal: controller.signal },
      )
      if (request !== sequence) return
      results.value = [
        ...new Map(
          (more ? [...results.value, ...page.items] : page.items).map((v) => [v.id, v]),
        ).values(),
      ]
      nextPage.value = page.nextPage
      cached.value = page.cached
      for (const v of page.items) savedIDs.value.delete(v.id)
      for (const id of page.savedIds) savedIDs.value.add(id)
      await loadStatus()
    } catch (e) {
      if (request === sequence) problem(e)
    } finally {
      if (request === sequence) loading.value = false
    }
  }
  async function searchSaved(more = false) {
    if (!discardOK()) return
    cancel()
    error.value = ''
    selected.value = null
    if (!more) {
      references.value = []
      appliedSaved = savedQuery.value.trim()
    }
    const request = ++sequence
    controller = new AbortController()
    loading.value = true
    try {
      const page = await api<SavedPage>(
        '/references?' +
          new URLSearchParams({
            q: appliedSaved,
            offset: String(more ? references.value.length : 0),
          }),
        { signal: controller.signal },
      )
      if (request !== sequence) return
      references.value = more ? [...references.value, ...page.items] : page.items
      hasMore.value = page.hasMore
      for (const r of page.items) savedIDs.value.add(r.id)
      if (!appliedSaved) total.value = page.total
    } catch (e) {
      if (request === sequence) problem(e)
    } finally {
      if (request === sequence) loading.value = false
    }
  }
  async function choose(row: Reference) {
    if (!discardOK()) return
    selected.value = { ...row, analysis: { ...row.analysis } }
    note.value = row.note
    analysis.value = { ...row.analysis }
    detailTab.value = 'details'
    detailLoading.value = false
    if (savedIDs.value.has(row.id) && tab.value === 'explore') {
      detailLoading.value = true
      try {
        const found = await api<Reference>('/references/' + row.id)
        if (selected.value?.id === row.id) {
          selected.value = found
          note.value = found.note
          analysis.value = { ...found.analysis }
        }
      } catch (e) {
        problem(e)
      } finally {
        if (selected.value?.id === row.id) detailLoading.value = false
      }
    }
  }
  async function save(id: string) {
    if (busyIDs.value.has(id)) return
    busyIDs.value.add(id)
    error.value = ''
    try {
      await api('/references', { method: 'POST', body: JSON.stringify({ id }) })
      savedIDs.value.add(id)
      await loadCount()
      tell('Reference saved')
    } catch (e) {
      problem(e)
    } finally {
      busyIDs.value.delete(id)
    }
  }
  async function remove(id: string) {
    if (!confirm('Remove this reference, notes and analysis? The YouTube video is not affected.'))
      return
    busyIDs.value.add(id)
    error.value = ''
    try {
      await api('/references/' + id, { method: 'DELETE' })
      savedIDs.value.delete(id)
      exportIDs.value.delete(id)
      references.value = references.value.filter((r) => r.id !== id)
      if (selected.value?.id === id) selected.value = null
      await loadCount()
      tell('Reference removed')
    } catch (e) {
      problem(e)
    } finally {
      busyIDs.value.delete(id)
    }
  }
  async function saveNote() {
    if (!selected.value || detailLoading.value) return
    const id = selected.value.id,
      text = note.value,
      fields = { ...analysis.value }
    noteBusy.value = true
    error.value = ''
    try {
      await api('/references/' + id, {
        method: 'PATCH',
        body: JSON.stringify({ note: text, analysis: fields }),
      })
      if (selected.value?.id === id) {
        selected.value.note = text
        selected.value.analysis = fields
      }
      const r = references.value.find((r) => r.id === id)
      if (r) {
        r.note = text
        r.analysis = fields
      }
      tell('Analysis and notes saved')
    } catch (e) {
      problem(e)
    } finally {
      noteBusy.value = false
    }
  }
  async function refreshVideo() {
    if (!selected.value) return
    const id = selected.value.id
    busyIDs.value.add(id)
    error.value = ''
    try {
      const video = await api<Video>('/references/' + id + '/refresh', { method: 'POST' })
      if (selected.value?.id === id) selected.value.video = video
      const r = references.value.find((r) => r.id === id)
      if (r) r.video = video
      const index = results.value.findIndex((v) => v.id === id)
      if (index >= 0) results.value[index] = video
      tell('YouTube details refreshed')
    } catch (e) {
      if (e instanceof APIError && e.code === 'unavailable') {
        if (selected.value?.id === id) selected.value.video = null
        const row = references.value.find((r) => r.id === id)
        if (row) row.video = null
      }
      problem(e)
    } finally {
      busyIDs.value.delete(id)
    }
  }
  function toggleExport(id: string) {
    if (exportIDs.value.has(id)) exportIDs.value.delete(id)
    else if (exportIDs.value.size < 10) exportIDs.value.add(id)
    else tell('Choose up to 10 references per brief')
  }
  async function exportBrief(ids: string[] = [...exportIDs.value]) {
    if (dirty.value) {
      error.value = 'Save your analysis before exporting'
      return
    }
    exportBusy.value = true
    error.value = ''
    try {
      const data = await api<{ markdown: string }>('/exports/claude', {
        method: 'POST',
        body: JSON.stringify({ ids }),
      })
      const url = URL.createObjectURL(
        new Blob([data.markdown], { type: 'text/markdown;charset=utf-8' }),
      )
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = 'catsoup-research-brief.md'
      anchor.click()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
      tell('Brief exported for Claude')
    } catch (e) {
      problem(e)
    } finally {
      exportBusy.value = false
    }
  }
  async function saveKey() {
    settingsBusy.value = true
    error.value = ''
    try {
      await api('/settings/youtube', { method: 'PUT', body: JSON.stringify({ key: key.value }) })
      key.value = ''
      await loadStatus()
      tell('Key saved. Search to verify access')
    } catch (e) {
      problem(e)
    } finally {
      settingsBusy.value = false
    }
  }
  async function removeKey() {
    if (!confirm('Remove the saved API key? Your references will stay.')) return
    settingsBusy.value = true
    try {
      await api('/settings/youtube', { method: 'DELETE' })
      await loadStatus()
      tell('API key removed')
    } catch (e) {
      problem(e)
    } finally {
      settingsBusy.value = false
    }
  }
  async function shutdown() {
    if (!discardOK() || !confirm('Stop the local Trends server? Reopen the launcher to continue.'))
      return
    try {
      await api('/shutdown', { method: 'POST' })
      stopped.value = true
      sessionStorage.removeItem('trends.session')
    } catch (e) {
      problem(e)
    }
  }
  function escape(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showFilters.value = false
      if (selected.value) closeDetails()
    }
  }
  function beforeLeave(e: BeforeUnloadEvent) {
    if (dirty.value || ideaDirty.value) e.preventDefault()
  }
  onMounted(async () => {
    window.addEventListener('keydown', escape)
    window.addEventListener('beforeunload', beforeLeave)
    try {
      await loadStatus()
      await loadCount()
    } catch (e) {
      problem(e)
    }
  })
  onBeforeUnmount(() => {
    cancel()
    clearTimeout(timer)
    window.removeEventListener('keydown', escape)
    window.removeEventListener('beforeunload', beforeLeave)
  })
  function combineIdeas() {
    if (!exportIDs.value.size || !discardOK()) return
    ideaSeed.value = [...exportIDs.value]
    selected.value = null
    switchTab('ideas')
  }
  async function searchTerm(term: string) {
    if (!discardOK()) return
    selected.value = null
    ideaDirty.value = false
    switchTab('explore')
    query.value = term.slice(0, 200)
    await searchYouTube()
  }
  return {
    ideaDirty,
    ideaSeed,
    combineIdeas,
    searchTerm,
    loadCount,
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
    preset,
    setPreset,
    searchSummary,
    searchPending,
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
  }
}

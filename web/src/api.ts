export interface Video {
  id: string
  title: string
  channel: string
  channelId: string
  tags?: string[]
  description: string
  thumbnail: string
  publishedAt: string
  observedAt: string
  duration: string
  views: number | null
  likes: number | null
  comments: number | null
}
export interface Analysis {
  hook: string
  structure: string
  twist: string
  adaptation: string
  keywords: string
}
export const emptyAnalysis = (): Analysis => ({
  hook: '',
  structure: '',
  twist: '',
  adaptation: '',
  keywords: '',
})
export interface Reference {
  id: string
  video: Video | null
  note: string
  analysis: Analysis
  savedAt: string
}
export interface Status {
  keyConfigured: boolean
  keyVerified: boolean
  keySource: string
  canStoreKey: boolean
  version: string
}
export interface SearchPage {
  items: Video[]
  nextPage: string
  savedIds: string[]
  cached: boolean
}
export interface SavedPage {
  items: Reference[]
  total: number
  hasMore: boolean
}
const token = new URLSearchParams(location.hash.slice(1)).get('token')
if (token) {
  sessionStorage.setItem('trends.session', token)
  history.replaceState(null, '', location.pathname)
}
export class APIError extends Error {
  constructor(
    message: string,
    public code: string,
  ) {
    super(message)
  }
}
export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch('/api/v1' + path, {
    ...options,
    headers: {
      Authorization: 'Bearer ' + (sessionStorage.getItem('trends.session') || ''),
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
  })
  const result = await response.json()
  if (!response.ok)
    throw new APIError(
      result.message || 'The request could not be completed',
      result.code || 'unknown',
    )
  return result as T
}
export function count(n: number | null | undefined) {
  return n == null
    ? '—'
    : new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(n)
}
export function date(v?: string) {
  return !v || v.startsWith('0001')
    ? 'Not available'
    : new Intl.DateTimeFormat('en', { dateStyle: 'medium' }).format(new Date(v))
}
export function duration(v: string) {
  const m = /^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$/.exec(v)
  if (!m) return ''
  const h = Number(m[1] || 0),
    min = Number(m[2] || 0),
    s = Number(m[3] || 0)
  return h
    ? h + ':' + String(min).padStart(2, '0') + ':' + String(s).padStart(2, '0')
    : min + ':' + String(s).padStart(2, '0')
}

export interface Idea {
  id: string
  title: string
  sources: string[]
  hook: string
  structure: string
  twist: string
  world: string
  notes: string
  updatedAt: string
}
export function downloadMarkdown(markdown: string, name: string) {
  const url = URL.createObjectURL(new Blob([markdown], { type: 'text/markdown;charset=utf-8' }))
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = name
  anchor.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

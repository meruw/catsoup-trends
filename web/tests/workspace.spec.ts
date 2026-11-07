import { test, expect } from '@playwright/test'
import { spawn, type ChildProcess } from 'node:child_process'
import { mkdtemp, readFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
let child: ChildProcess, launch: string, dataDir: string
async function start() {
  child = spawn(path.resolve('../bin/trends-fixture.exe'), [], {
    env: { ...process.env, TRENDS_TEST_DATA: dataDir },
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  })
  launch = await new Promise<string>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Test host did not start')), 15000)
    let buffer = ''
    child.stdout!.on('data', (chunk) => {
      buffer += chunk
      const match = buffer.match(/http:\/\/127\.0\.0\.1:\d+\/#token=[a-f0-9]+/)
      if (match) {
        clearTimeout(timer)
        resolve(match[0])
      }
    })
    child.once('error', reject)
    child.once('exit', (code) => {
      if (code) reject(new Error('Test host exited'))
    })
  })
}
test.beforeAll(async () => {
  dataDir = await mkdtemp(path.join(tmpdir(), 'catsoup-trends-test-'))
  await start()
})
test.afterAll(() => {
  child?.kill()
})
test.beforeEach(async ({ page }) => {
  await page.route('https://i.ytimg.com/**', (route) =>
    route.fulfill({
      contentType: 'image/svg+xml',
      body: `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180"><rect width="320" height="180" fill="#252d35"/><path d="M0 140L320 110V180H0" fill="#4d565e"/><rect x="200" y="32" width="8" height="100" fill="#c9a370"/><path d="M175 36H235L220 20H190Z" fill="#f5b362"/><rect x="115" y="80" width="24" height="44" fill="#e18549"/><rect x="118" y="61" width="18" height="18" fill="#e5cbb2"/><path d="M127 124L65 155H100L143 124" fill="#111a22"/><text x="12" y="22" fill="#fff" font-family="sans-serif" font-size="12">Synthetic test artwork</text></svg>`,
    }),
  )
  await page.goto(launch)
})
test('search, paginate, save, analyze, export, restart and remove', async ({ page }) => {
  await expect(page.getByRole('heading', { name: 'Explore YouTube' })).toBeVisible()
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('animation')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(2)
  await page.getByRole('button', { name: 'Load more' }).click()
  await expect(page.locator('.result-row')).toHaveCount(3)
  await page.getByRole('button', { name: 'Save reference', exact: true }).first().click()
  await expect(page.getByRole('status').filter({ hasText: 'Reference saved' })).toBeVisible()
  await page.getByRole('button', { name: /View details: The shadow/ }).click()
  await page.getByRole('button', { name: 'Analysis', exact: true }).click()
  await page.getByLabel('Hook', { exact: true }).fill('The shadow reacts before the character')
  await page
    .getByLabel('Your adaptation', { exact: true })
    .fill('A reflection warns the character about an unseen visitor')
  await page.getByLabel('Topics and keywords').fill('suspense, anticipation')
  await expect(page.getByRole('button', { name: 'Export reference', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'Save analysis', exact: true }).click()
  await expect(
    page.getByRole('status').filter({ hasText: 'Analysis and notes saved' }),
  ).toBeVisible()
  const singleDownloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Export reference', exact: true }).click()
  const singleDownload = await singleDownloadPromise
  expect(await readFile((await singleDownload.path())!, 'utf8')).toContain(
    'The shadow reacts before the character',
  )
  await page.getByRole('button', { name: /^Saved/ }).click()
  await expect(page.locator('.result-row')).toHaveCount(1)
  await page.getByRole('checkbox', { name: /Select for export/ }).check()
  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: /Export for Claude/ }).click()
  const download = await downloadPromise
  const text = await readFile((await download.path())!, 'utf8')
  expect(text).toContain('A reflection warns')
  expect(text).toContain('https://www.youtube.com/watch?v=fixture0001')
  expect(text).not.toContain('24000')
  await page.getByRole('button', { name: /View details: The shadow/ }).click()
  await page.getByRole('button', { name: 'Analysis', exact: true }).click()
  await expect(page.getByLabel('Hook', { exact: true })).toHaveValue(
    'The shadow reacts before the character',
  )
  await page.screenshot({ path: '../work/screenshots/saved-analysis.png', fullPage: true })
  const old = child
  const exited = new Promise<void>((resolve) => old.once('exit', () => resolve()))
  old.kill()
  await exited
  await start()
  await page.goto(launch)
  await page.getByRole('button', { name: /^Saved/ }).click()
  await page.getByRole('button', { name: /View details: The shadow/ }).click()
  await page.getByRole('button', { name: 'Analysis', exact: true }).click()
  await expect(page.getByLabel('Your adaptation', { exact: true })).toHaveValue(
    'A reflection warns the character about an unseen visitor',
  )
  page.once('dialog', (dialog) => dialog.accept())
  await page.getByRole('button', { name: 'Remove', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(0)
})
test('filters, errors, cancellation and responsive layout', async ({ page }) => {
  await page.getByRole('button', { name: 'Filters', exact: true }).click()
  await page.getByLabel('Sort by').selectOption('viewCount')
  await page.getByLabel('Preferred language').selectOption('es')
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('animation')
  await page.getByRole('button', { name: 'Apply', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(2)
  await page.screenshot({ path: '../work/screenshots/explore.png', fullPage: true })
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('quota')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('quota')
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('slow')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Search', exact: true })).toBeEnabled()
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('empty')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'No videos found' })).toBeVisible()
  await page.setViewportSize({ width: 390, height: 844 })
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('animation')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(2)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.getByRole('button', { name: /View details: The shadow/ }).click()
  await expect(page.getByRole('link', { name: 'Watch on YouTube' })).toBeVisible()
  await page.screenshot({ path: '../work/screenshots/mobile-detail.png', fullPage: true })
  await page.getByRole('button', { name: 'Close details' }).click()
  await expect(page.getByRole('heading', { name: 'Explore YouTube' })).toBeVisible()
})

test('presets keep displayed results and pagination tied to applied filters', async ({ page }) => {
  const searches: URL[] = []
  page.on('request', (request) => {
    if (request.url().includes('/api/v1/search?')) searches.push(new URL(request.url()))
  })
  await page.getByRole('textbox', { name: 'Search YouTube', exact: true }).fill('animation')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(2)
  expect(searches[0]!.searchParams.get('period')).toBe('30')
  expect(searches[0]!.searchParams.get('order')).toBe('viewCount')
  await expect(page.locator('.applied-summary')).toHaveText(
    'Last 30 days · Most viewed · Under 4 minutes',
  )
  await page.getByRole('button', { name: 'All-time references', exact: true }).click()
  await expect(page.getByText('Changes not applied · Search to update')).toBeVisible()
  await expect(page.locator('.applied-summary')).toContainText('Last 30 days')
  expect(searches.length).toBe(1)
  await page.getByRole('button', { name: 'Load more', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(3)
  expect(searches[1]!.searchParams.get('period')).toBe('30')
  expect(searches[1]!.searchParams.get('page')).toBe('page2')
  await page.getByRole('button', { name: 'Search', exact: true }).click()
  await expect(page.locator('.result-row')).toHaveCount(2)
  expect(searches[2]!.searchParams.get('period')).toBe('any')
  expect(searches[2]!.searchParams.has('page')).toBe(false)
  await expect(page.locator('.applied-summary')).toContainText('Any time')
  await expect(page.getByText('Changes not applied · Search to update')).toHaveCount(0)
  await page.getByRole('button', { name: 'Filters', exact: true }).click()
  await page.getByLabel('Published').selectOption('7')
  await page.getByLabel('Preferred language').selectOption('es')
  await page.getByRole('button', { name: 'Apply', exact: true }).click()
  await expect(page.locator('.applied-summary')).toHaveText(
    'Last 7 days · Most viewed · Under 4 minutes · Spanish preferred',
  )
  await expect(page.locator('.result-row')).toHaveCount(2)
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})

test('discover terms and combine references into a persistent idea', async ({ page }) => {
  await page.getByRole('button', { name: 'Discover', exact: true }).click()
  await expect(page.locator('.discovery-row')).toHaveCount(3)
  await expect(
    page.getByText('Terms from popular videos, grouped by source.', { exact: false }),
  ).toBeVisible()
  await page.screenshot({ path: '../work/screenshots/discover.png', fullPage: true })
  await page
    .locator('.discovery-row')
    .first()
    .getByRole('button', { name: 'animation', exact: true })
    .click()
  await expect(page.getByRole('textbox', { name: 'Search YouTube', exact: true })).toHaveValue(
    'animation',
  )
  await expect(page.locator('.result-row')).toHaveCount(2)
  for (let i = 0; i < 2; i++) {
    const row = page.locator('.result-row').nth(i)
    const save = row.getByRole('button', { name: 'Save reference', exact: true })
    if (await save.count()) await save.click()
  }
  await page.getByRole('button', { name: /View details: The shadow/ }).click()
  await page.getByText('Keywords and hashtags', { exact: true }).click()
  await expect(page.getByRole('button', { name: '#Historia', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Analysis', exact: true }).click()
  await page.getByLabel('Hook', { exact: true }).fill('A warning appears before the danger')
  await page
    .getByLabel('Structure', { exact: true })
    .fill('Warning, failed escape, unexpected helper')
  await page.getByRole('button', { name: 'Save analysis', exact: true }).click()
  await expect(page.getByText('Analysis and notes saved', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: /^Saved/ }).click()
  const checks = page.getByRole('checkbox', { name: /Select for export/ })
  await checks.nth(0).check()
  await checks.nth(1).check()
  await page.getByRole('button', { name: 'Combine into idea', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Ideas', exact: true })).toBeVisible()
  await page.getByLabel('Idea title', { exact: true }).fill('The warning reflection')
  await page.getByRole('button', { name: 'Use hook', exact: true }).click()
  await expect(page.getByLabel('Opening hook', { exact: true })).toHaveValue(
    'A warning appears before the danger',
  )
  await page
    .getByLabel('Original twist', { exact: true })
    .fill('The reflection is trying to escape us')
  await expect(page.getByRole('button', { name: 'Export for Claude', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'Save idea', exact: true }).click()
  await expect(page.getByText('Idea saved', { exact: true })).toBeVisible()
  await page.screenshot({ path: '../work/screenshots/ideas.png', fullPage: true })
  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Export for Claude', exact: true }).click()
  const download = await downloadPromise
  const text = await readFile((await download.path())!, 'utf8')
  expect(text).toContain('The reflection is trying to escape us')
  expect(text).toContain('fixture0001')
  expect(text).not.toContain('Synthetic reference')
  await page.reload()
  await page.getByRole('button', { name: 'Ideas', exact: true }).click()
  await page.getByRole('button', { name: /The warning reflection/ }).click()
  await expect(page.getByLabel('Original twist', { exact: true })).toHaveValue(
    'The reflection is trying to escape us',
  )
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  )
  await page.screenshot({ path: '../work/screenshots/ideas-mobile.png', fullPage: true })
  await page.getByLabel('Notes', { exact: true }).fill('Unsaved detail')
  page.once('dialog', (dialog) => dialog.dismiss())
  await page.getByRole('button', { name: 'Explore', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Ideas', exact: true })).toBeVisible()
  page.once('dialog', (dialog) => dialog.accept())
  await page.getByRole('button', { name: 'Delete idea', exact: true }).click()
  await expect(page.getByText('Idea deleted', { exact: true })).toBeVisible()
})

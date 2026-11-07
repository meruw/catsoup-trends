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
  await page.getByRole('button', { name: 'Save analysis', exact: true }).click()
  await expect(
    page.getByRole('status').filter({ hasText: 'Analysis and notes saved' }),
  ).toBeVisible()
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

import { test, expect } from '@playwright/test'
import { fork, type ChildProcess } from 'node:child_process'
import { mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
test('development proxies authenticated API and hot reloads CSS without losing input', async ({
  page,
}) => {
  test.setTimeout(60000)
  const data = await mkdtemp(path.join(tmpdir(), 'trends-dev-test-'))
  let child: ChildProcess | undefined
  const css = path.resolve('src/style.css'),
    original = await readFile(css, 'utf8')
  try {
    child = fork(path.resolve('dev.mjs'), [], {
      env: {
        ...process.env,
        TRENDS_DEV_FIXTURE: '1',
        TRENDS_DEV_NO_BROWSER: '1',
        TRENDS_DEV_DATA_DIR: data,
      },
      silent: true,
      windowsHide: true,
    })
    const launch = await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('Dev host did not start')), 20000)
      child!.once('message', (value: unknown) => {
        clearTimeout(timer)
        resolve((value as { launch: string }).launch)
      })
      child!.once('error', () => {
        clearTimeout(timer)
        reject(new Error('Dev process failed'))
      })
      child!.once('exit', () => {
        clearTimeout(timer)
        reject(new Error('Dev process exited early'))
      })
    })
    await page.goto(launch)
    await expect(page.getByRole('heading', { name: 'Explore YouTube' })).toBeVisible()
    await page
      .getByRole('textbox', { name: 'Search YouTube', exact: true })
      .fill('input survives HMR')
    await writeFile(css, original + '\nbody { --hmr-verification: ready; }\n', 'utf8')
    await expect
      .poll(() =>
        page.evaluate(() =>
          getComputedStyle(document.body).getPropertyValue('--hmr-verification').trim(),
        ),
      )
      .toBe('ready')
    await expect(page.getByRole('textbox', { name: 'Search YouTube', exact: true })).toHaveValue(
      'input survives HMR',
    )
    await page.getByRole('button', { name: 'Search', exact: true }).click()
    await expect(page.locator('.result-row')).toHaveCount(2)
    const url = new URL(launch)
    const forbidden = await page.request.get(url.origin + '/api/v1/status', {
      headers: { Origin: 'https://untrusted.example' },
    })
    expect(forbidden.status()).toBe(403)
    const unauthenticated = await page.request.get(url.origin + '/api/v1/status')
    expect(unauthenticated.status()).toBe(401)
  } finally {
    await writeFile(css, original, 'utf8')
    if (child?.connected) child.disconnect()
    if (child && child.exitCode === null)
      await new Promise<void>((resolve) => {
        child!.once('exit', () => resolve())
        setTimeout(() => {
          child?.kill()
          resolve()
        }, 5000).unref()
      })
  }
})

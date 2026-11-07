// Local development only. Not imported by the production application.
import { spawn } from 'node:child_process'
import { mkdir, access } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { createServer, build } from 'vite'
const webDir = path.dirname(fileURLToPath(import.meta.url))
const repo = path.dirname(webDir)
const fixture = process.env.TRENDS_DEV_FIXTURE === '1'
const dataDir = process.env.TRENDS_DEV_DATA_DIR || path.join(repo, 'work', 'dev-data')
let backend,
  vite,
  closing = false
async function close(code = 0) {
  if (closing) return
  closing = true
  await vite?.close()
  if (backend && backend.exitCode === null) {
    backend.kill()
    await new Promise((resolve) => {
      backend.once('exit', resolve)
      setTimeout(resolve, 3000).unref()
    })
  }
  process.exit(code)
}
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => void close())
process.on('disconnect', () => void close())
function run(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: repo, stdio: 'inherit', windowsHide: true })
    child.once('error', () =>
      reject(new Error('Cannot start build tool. Check GoExe and NodeDir.')),
    )
    child.once('exit', (code) =>
      code === 0 ? resolve() : reject(new Error('Development build failed')),
    )
  })
}
try {
  await mkdir(dataDir, { recursive: true })
  let executable = path.join(repo, 'bin', 'trends-fixture.exe')
  if (!fixture) {
    try {
      await access(path.join(webDir, 'dist', 'index.html'))
    } catch {
      await build({ root: webDir, configFile: path.join(webDir, 'vite.config.ts') })
    }
    await mkdir(path.join(repo, 'work', 'dev-bin'), { recursive: true })
    executable = path.join(
      repo,
      'work',
      'dev-bin',
      process.platform === 'win32' ? 'trends-dev.exe' : 'trends-dev',
    )
    await run(process.env.TRENDS_GO_EXE || 'go', ['build', '-o', executable, './cmd/trends'])
  }
  backend = spawn(executable, fixture ? [] : ['-no-browser', '-data-dir', dataDir], {
    cwd: repo,
    windowsHide: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: { ...process.env, ...(fixture ? { TRENDS_TEST_DATA: dataDir } : {}) },
  })
  const launch = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Local backend did not start in time')), 15000)
    let buffer = ''
    backend.stdout.on('data', (chunk) => {
      buffer = (buffer + chunk.toString()).slice(-4096)
      const match = buffer.match(/http:\/\/127\.0\.0\.1:\d+\/#token=[a-f0-9]+/)
      if (match) {
        clearTimeout(timer)
        resolve(new URL(match[0]))
      }
    })
    backend.once('error', () => {
      clearTimeout(timer)
      reject(new Error('Cannot start the local backend'))
    })
    backend.once('exit', () => {
      clearTimeout(timer)
      reject(new Error('Local backend stopped before startup'))
    })
  })
  backend.on('exit', () => {
    if (!closing) void close()
  })
  // Never mirror backend stdout: it contains the private session launch URL.
  backend.stderr.on('data', () => console.error('The local backend reported an error.'))
  let origin = ''
  vite = await createServer({
    root: webDir,
    configFile: path.join(webDir, 'vite.config.ts'),
    plugins: [
      {
        name: 'local-development-boundary',
        configureServer(server) {
          server.middlewares.use((req, res, next) => {
            if (
              !origin ||
              req.headers.host !== new URL(origin).host ||
              (req.headers.origin && req.headers.origin !== origin) ||
              req.headers['sec-fetch-site'] === 'cross-site'
            ) {
              res.statusCode = 403
              res.end('Local development access only')
              return
            }
            res.setHeader('Referrer-Policy', 'no-referrer')
            res.setHeader('Cache-Control', 'no-store')
            next()
          })
        },
      },
    ],
    server: {
      host: '127.0.0.1',
      port: 0,
      cors: false,
      allowedHosts: ['127.0.0.1'],
      fs: { strict: true, allow: [webDir] },
      proxy: {
        '/api/': {
          target: launch.origin,
          changeOrigin: true,
          configure(proxy) {
            proxy.on('proxyReq', (proxyReq) => proxyReq.setHeader('Origin', launch.origin))
          },
        },
      },
    },
  })
  await vite.listen()
  const address = vite.httpServer.address()
  origin = `http://127.0.0.1:${address.port}`
  const browserURL = origin + '/' + launch.hash
  console.log('Trends development is ready at ' + origin)
  console.log('UI changes reload automatically. Stop with Ctrl+C; restart for Go changes.')
  if (process.send) process.send({ launch: browserURL, backendPID: backend.pid })
  if (process.env.TRENDS_DEV_NO_BROWSER !== '1') {
    const command =
      process.platform === 'win32'
        ? 'rundll32'
        : process.platform === 'darwin'
          ? 'open'
          : 'xdg-open'
    const args =
      process.platform === 'win32' ? ['url.dll,FileProtocolHandler', browserURL] : [browserURL]
    const browser = spawn(command, args, { stdio: 'ignore', windowsHide: true })
    browser.on('error', () =>
      console.error('Could not open the browser. Restart development to try again.'),
    )
  }
} catch (error) {
  console.error(error instanceof Error ? error.message : 'Development startup failed')
  await close(1)
}

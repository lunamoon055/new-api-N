import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

const root = process.cwd()
const client = path.join(root, 'dist/client')
let html = await readFile(path.join(client, 'index.html'), 'utf8')
const logo = 'data:image/png;base64,' + (await readFile(path.join(root, 'public/brand-logo.png'))).toString('base64')
const scriptMatch = html.match(/<script[^>]+src="([^"]+)"[^>]*><\/script>/)
const styleMatch = html.match(/<link[^>]+href="([^"]+\.css)"[^>]*>/)
if (!scriptMatch || !styleMatch) throw new Error('Expected Vite script and stylesheet were not found')
let js = await readFile(path.join(client, scriptMatch[1].replace(/^\//, '')), 'utf8')
const css = await readFile(path.join(client, styleMatch[1].replace(/^\//, '')), 'utf8')
js = js.replaceAll('/brand-logo.png', logo).replaceAll('</script', '<\\/script')
// Callback replacements preserve literal $& / $` sequences in bundled code.
html = html.replace(scriptMatch[0], () => '<script type="module">' + js + '</script>')
html = html.replace(styleMatch[0], () => '<style>' + css + '</style>')
const repo = path.resolve(root, '../..')
const outputs = [
  path.join(root, 'model-integration.html'),
  path.join(repo, 'docs/model-integration.html'),
  path.join(repo, 'web/default/public/docs/model-integration.html'),
]
for (const output of outputs) {
  await mkdir(path.dirname(output), { recursive: true })
  await writeFile(output, html)
  console.log('Exported ' + output)
}

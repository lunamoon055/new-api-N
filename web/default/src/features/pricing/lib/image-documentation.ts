/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
export function buildNativeImageSample(
  lang: 'curl' | 'python' | 'typescript' | 'javascript',
  options: {
    baseUrl: string
    endpointPath: string
    modelName: string
    apiKeyEnv: string
  }
): string {
  const url = `${options.baseUrl}${options.endpointPath}`
  const edit = options.endpointPath === '/v1/images/edits'
  const body = {
    model: options.modelName,
    prompt: 'A red ceramic cup on a white background.',
    n: 1,
    resolution: '1K',
    ratio: '1:1',
    response_format: 'url',
  }
  if (lang === 'curl') {
    const lines = [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${options.apiKeyEnv}" \\`,
    ]
    if (!edit)
      return [
        ...lines,
        '  -H "Content-Type: application/json" \\',
        `  -d '${JSON.stringify(body)}'`,
      ].join('\n')
    return [
      ...lines,
      ...Object.entries(body).map(
        ([key, value]) => `  -F '${key}=${value}' \\`
      ),
      "  -F 'image[]=@reference.png'",
    ].join('\n')
  }
  if (lang === 'python') {
    const lines = [
      'import os',
      'import requests',
      '',
      `headers = {"Authorization": "Bearer " + os.environ["${options.apiKeyEnv}"]}`,
      `body = ${JSON.stringify(body)}`,
    ]
    if (edit)
      lines.push(
        'with open("reference.png", "rb") as image:',
        `    response = requests.post("${url}", headers=headers, data=body, files=[("image[]", ("reference.png", image, "image/png"))])`
      )
    else
      lines.push(
        `response = requests.post("${url}", headers=headers, json=body)`
      )
    return [
      ...lines,
      'response.raise_for_status()',
      'print(response.json()["data"][0]["url"])',
    ].join('\n')
  }
  const lines: string[] = []
  if (edit) {
    if (lang === 'typescript')
      lines.push("import { readFile } from 'node:fs/promises'", '')
    lines.push('const body = new FormData()')
    for (const [key, value] of Object.entries(body))
      lines.push(`body.append('${key}', '${value}')`)
    if (lang === 'typescript')
      lines.push(
        "body.append('image[]', new Blob([new Uint8Array(await readFile('reference.png'))], { type: 'image/png' }), 'reference.png')"
      )
    else
      lines.push(
        "body.append('image[]', document.querySelector('input[type=file]').files[0])"
      )
  } else
    lines.push(`const body = JSON.stringify(${JSON.stringify(body, null, 2)})`)
  const authorization =
    lang === 'javascript'
      ? "'Bearer <YOUR_API_KEY>'"
      : '`Bearer ${process.env.' + options.apiKeyEnv + '}`'
  return [
    ...lines,
    '',
    `const response = await fetch('${url}', {`,
    "  method: 'POST',",
    `  headers: { Authorization: ${authorization}${edit ? '' : ", 'Content-Type': 'application/json'"} },`,
    '  body,',
    '})',
    'if (!response.ok) throw new Error(await response.text())',
    'console.log((await response.json()).data[0].url)',
  ].join('\n')
}

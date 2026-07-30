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
import { Fragment, createElement, useMemo, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

const ALLOWED_TAGS = new Set([
  'a',
  'b',
  'blockquote',
  'br',
  'code',
  'dd',
  'del',
  'details',
  'div',
  'dl',
  'dt',
  'em',
  'figcaption',
  'figure',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'hr',
  'i',
  'img',
  'li',
  'ol',
  'p',
  'pre',
  's',
  'small',
  'span',
  'strong',
  'sub',
  'summary',
  'sup',
  'table',
  'tbody',
  'td',
  'tfoot',
  'th',
  'thead',
  'tr',
  'u',
  'ul',
])

const BLOCKED_TAGS = new Set([
  'base',
  'button',
  'canvas',
  'embed',
  'form',
  'frame',
  'frameset',
  'iframe',
  'input',
  'link',
  'math',
  'meta',
  'object',
  'option',
  'script',
  'select',
  'style',
  'svg',
  'template',
  'textarea',
])

function currentBaseUrl() {
  return typeof window === 'undefined'
    ? 'http://localhost/'
    : window.location.href
}

function normalizeLink(value: string | null): string | null {
  if (!value) return null
  const trimmed = value.trim()
  if (!trimmed) return null

  if (/^(?:mailto|tel):/i.test(trimmed)) {
    return trimmed
  }

  try {
    const url = new URL(trimmed, currentBaseUrl())
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return null
    if (url.username || url.password) return null
    return url.toString()
  } catch {
    return null
  }
}

function normalizeImage(value: string | null): string | null {
  if (!value) return null
  const trimmed = value.trim()
  if (!trimmed) return null

  if (
    trimmed.length <= 2 * 1024 * 1024 &&
    /^data:image\/(?:png|gif|jpe?g|webp);base64,[a-z0-9+/=\s]+$/i.test(trimmed)
  ) {
    return trimmed
  }

  try {
    const url = new URL(trimmed, currentBaseUrl())
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return null
    if (url.username || url.password) return null
    return url.toString()
  } catch {
    return null
  }
}

function boundedSpan(value: string | null): number | undefined {
  if (!value) return undefined
  const parsed = Number.parseInt(value, 10)
  return Number.isInteger(parsed) && parsed >= 1 && parsed <= 100
    ? parsed
    : undefined
}

function renderNode(node: Node, key: string): ReactNode {
  if (node.nodeType === 3) return node.textContent
  if (node.nodeType !== 1) return null

  const element = node as HTMLElement
  const tag = element.tagName.toLowerCase()

  if (BLOCKED_TAGS.has(tag)) return null

  const children = Array.from(element.childNodes).map((child, index) =>
    renderNode(child, `${key}-${index}`)
  )

  if (!ALLOWED_TAGS.has(tag)) {
    return <Fragment key={key}>{children}</Fragment>
  }

  const props: Record<string, unknown> = { key }
  const title = element.getAttribute('title')
  if (title) props.title = title

  if (tag === 'a') {
    const href = normalizeLink(element.getAttribute('href'))
    if (!href) return <Fragment key={key}>{children}</Fragment>
    props.href = href
    props.target = '_blank'
    props.rel = 'noopener noreferrer nofollow'
  } else if (tag === 'img') {
    const src = normalizeImage(element.getAttribute('src'))
    if (!src) return null
    props.src = src
    props.alt = element.getAttribute('alt') || ''
    props.loading = 'lazy'
    props.decoding = 'async'
    props.referrerPolicy = 'no-referrer'
  } else if (tag === 'td' || tag === 'th') {
    props.colSpan = boundedSpan(element.getAttribute('colspan'))
    props.rowSpan = boundedSpan(element.getAttribute('rowspan'))
  } else if (tag === 'ol') {
    const start = boundedSpan(element.getAttribute('start'))
    if (start) props.start = start
  }

  if (tag === 'br' || tag === 'hr' || tag === 'img') {
    return createElement(tag, props)
  }

  return createElement(tag, props, ...children)
}

function renderSafeHtml(html: string): ReactNode {
  if (typeof DOMParser === 'undefined') return html

  const document = new DOMParser().parseFromString(html, 'text/html')
  return Array.from(document.body.childNodes).map((node, index) =>
    renderNode(node, `safe-html-${index}`)
  )
}

interface SafeHtmlProps {
  html: string
  className?: string
}

export function SafeHtml({ html, className }: SafeHtmlProps) {
  const content = useMemo(() => renderSafeHtml(html), [html])
  return <div className={cn(className)}>{content}</div>
}

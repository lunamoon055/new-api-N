import { useCallback, useEffect, useRef, useState } from 'react'
import { normalizeCatalog, resolveSelectedModel } from './model-catalog.js'

export function useModelCatalog() {
  const [catalog, setCatalog] = useState({ models: [], endpoints: {} })
  const [selectedName, setSelectedName] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [updatedAt, setUpdatedAt] = useState(null)
  const request = useRef(null)

  const loadCatalog = useCallback(async () => {
    request.current?.abort()
    const controller = new AbortController()
    request.current = controller
    setLoading(true)
    setError('')
    const timeout = window.setTimeout(() => controller.abort(), 12000)
    try {
      if (window.location.protocol === 'file:') {
        throw new Error('请从网站的文档入口打开，以读取实时模型列表。')
      }
      const response = await fetch('/api/pricing', {
        credentials: 'same-origin', cache: 'no-store', signal: controller.signal,
      })
      if (!response.ok) throw new Error(response.status === 401 || response.status === 403 ? '模型列表需要登录，请登录网站后重试。' : '暂时无法读取网站模型列表，请重试。')
      const next = normalizeCatalog(await response.json())
      if (request.current !== controller) return
      setCatalog(next)
      setSelectedName((current) => resolveSelectedModel(next.models, current))
      setUpdatedAt(new Date())
    } catch (cause) {
      if (request.current !== controller) return
      // A failed refresh must not expose removed models from an old snapshot.
      setCatalog({ models: [], endpoints: {} })
      setSelectedName('')
      setUpdatedAt(null)
      setError(cause.name === 'AbortError' ? '读取模型列表超时，请重试。' : cause.message)
    } finally {
      window.clearTimeout(timeout)
      if (request.current === controller) setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadCatalog()
    function refreshVisible() {
      if (document.visibilityState === 'visible') loadCatalog()
    }
    const timer = window.setInterval(refreshVisible, 60000)
    window.addEventListener('focus', refreshVisible)
    document.addEventListener('visibilitychange', refreshVisible)
    return () => {
      window.clearInterval(timer)
      window.removeEventListener('focus', refreshVisible)
      document.removeEventListener('visibilitychange', refreshVisible)
      const active = request.current
      request.current = null
      active?.abort()
    }
  }, [loadCatalog])

  return { catalog, selectedName, setSelectedName, loading, error, updatedAt, loadCatalog }
}

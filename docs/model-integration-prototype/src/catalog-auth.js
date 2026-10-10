// The standalone docs page must send the same user ID header as the main site.
// Its same-origin iframe shares the site's session cookie and localStorage.
export function getPricingRequestHeaders(storage) {
  try {
    const uid = (storage ?? window.localStorage).getItem('uid')
    return uid && /^[1-9]\d*$/.test(uid) ? { 'New-Api-User': uid } : {}
  } catch {
    return {}
  }
}

export function getCatalogHttpError(status) {
  if (status === 401) return '模型列表鉴权失败，请刷新页面；仍失败请重新登录。'
  if (status === 403) return '模型目录已关闭或当前账号无权访问。'
  return '暂时无法读取网站模型列表，请重试。'
}

const BASE = '/portal/api'

function headers() {
  return {
    'Content-Type': 'application/json',
    'X-API-Key': localStorage.getItem('portal_api_key') || '',
  }
}

async function request(path) {
  const res = await fetch(BASE + path, { headers: headers() })
  const json = await res.json()
  if (!json.success) {
    if (res.status === 401) { localStorage.removeItem('portal_api_key'); window.location.reload() }
    throw new Error(json.error?.message || 'Request failed')
  }
  return json.data
}

export const api = {
  me: () => request('/me'),
  dashboard: () => request('/dashboard'),
  orders: () => request('/orders'),
  orderDetail: (id) => request('/orders/' + id),
  events: () => request('/events'),
  updateSettings: (body) => fetch(BASE + '/settings', { method: 'PUT', headers: headers(), body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
}

export function login(key) {
  localStorage.setItem('portal_api_key', key)
}
export function logout() {
  localStorage.removeItem('portal_api_key')
  window.location.href = '/portal'
}
export function isLoggedIn() {
  return !!localStorage.getItem('portal_api_key')
}

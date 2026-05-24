export const SERVER = import.meta.env.VITE_SERVER_BASE || 'http://localhost:8082'
const BASE = SERVER + '/api/admin'
const H = { 'X-Admin-Key': 'admin-dev-key', 'Content-Type': 'application/json' }

async function request(path) {
  const res = await fetch(BASE + path, { headers: H })
  const json = await res.json()
  if (!json.success) throw new Error(json.error?.message || 'Request failed')
  return json.data
}

export async function exportCSV(params) {
  const res = await fetch(BASE + '/orders/export?' + params, { headers: H })
  if (!res.ok) throw new Error('Export failed')
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const ts = new Date().toISOString().replace(/[-:]/g,'').replace('T','_').slice(0,15)
  a.href = url; a.download = `星河支付_${ts}.csv`; a.click()
  URL.revokeObjectURL(url)
}

export const api = {
  dashboard: () => request('/dashboard'),
  config: () => request('/config'),
  listApps: () => request('/apps'),
  createApp: (body) => fetch(BASE + '/apps', { method: 'POST', headers: H, body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
  updateApp: (id, body) => fetch(BASE + '/apps/' + id, { method: 'PUT', headers: H, body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
  listOrders: (params = '') => fetch(BASE + '/orders?' + params, { headers: H }).then(r => r.json()).then(j => j.data),
  getOrder: (id) => request('/orders/' + id),
  listEvents: (paymentId) => request('/events?payment_id=' + paymentId),
}

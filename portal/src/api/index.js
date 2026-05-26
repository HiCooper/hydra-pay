const SERVER = import.meta.env.VITE_SERVER_BASE || 'http://localhost:8082'
const BASE = SERVER + '/portal/api'

function headers() {
  return {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer ' + (localStorage.getItem('portal_token') || ''),
  }
}

function apiKeyHeaders() {
  return {
    'Content-Type': 'application/json',
    'X-API-Key': localStorage.getItem('portal_api_key') || '',
  }
}

async function request(path) {
  const res = await fetch(BASE + path, { headers: headers() })
  const json = await res.json()
  if (!json.success) {
    if (res.status === 401) { localStorage.removeItem('portal_token'); window.location.href = '/portal/login' }
    throw new Error(json.error?.message || 'Request failed')
  }
  return json.data
}

export const api = {
  // Auth
  login: async (email, password) => {
    const res = await fetch(BASE + '/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    const json = await res.json()
    if (!json.success) throw new Error(json.error?.message || 'Login failed')
    return json.data
  },

  // Me + Apps
  me: () => request('/me'),
  listApps: () => request('/apps'),
  createApp: (body) => fetch(BASE + '/apps', { method: 'POST', headers: headers(), body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),

  // Dashboard
  dashboard: () => request('/dashboard'),

  // Orders
  orders: () => request('/orders'),
  orderDetail: (id) => request('/orders/' + id),
  events: () => request('/events'),

  // Settings
  updateSettings: (body) => fetch(BASE + '/settings', { method: 'PUT', headers: headers(), body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),

  // Payment Links
  paymentLinks: () => request('/payment-links'),
  createPaymentLink: (body) => fetch(BASE + '/payment-links', { method: 'POST', headers: headers(), body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
  expirePaymentLink: (id) => fetch(BASE + '/payment-links/' + id + '/expire', { method: 'POST', headers: headers() }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
  deletePaymentLink: (id) => fetch(BASE + '/payment-links/' + id, { method: 'DELETE', headers: headers() }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),

  // Subscriptions
  subscriptions: () => request('/subscriptions'),

  // Onboarding
  initiateOnboarding: (body) => fetch(BASE + '/onboarding', { method: 'POST', headers: headers(), body: JSON.stringify(body) }).then(r => r.json()).then(j => { if (!j.success) throw new Error(j.error?.message); return j.data }),
  getOnboardingStatus: () => request('/onboarding'),
}

export function login(token, merchant) {
  localStorage.setItem('portal_token', token)
  if (merchant) localStorage.setItem('portal_merchant', JSON.stringify(merchant))
}

export function logout() {
  localStorage.removeItem('portal_token')
  localStorage.removeItem('portal_merchant')
  window.location.href = '/portal/login'
}

export function isLoggedIn() {
  return !!localStorage.getItem('portal_token')
}

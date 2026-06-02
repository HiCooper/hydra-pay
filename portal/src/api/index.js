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

class AuthError extends Error {
  constructor(message) { super(message); this.name = 'AuthError' }
}

async function request(path) {
  let res
  try {
    res = await fetch(BASE + path, { headers: headers() })
  } catch (e) {
    // Network error (server down / restart) — don't redirect, just throw
    throw new Error('网络连接失败，请稍后重试')
  }
  const json = await res.json()
  if (!json.success) {
    if (res.status === 401) {
      localStorage.removeItem('portal_token')
      localStorage.removeItem('portal_merchant')
      throw new AuthError(json.error?.message || '认证已过期')
    }
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

  // Channels
  listChannels: () => request('/channels'),
}

export function saveLogin(token, merchant) {
  localStorage.setItem('portal_token', token)
  if (merchant) localStorage.setItem('portal_merchant', JSON.stringify(merchant))
}

export function doLogout() {
  localStorage.removeItem('portal_token')
  localStorage.removeItem('portal_merchant')
}

export function isLoggedIn() {
  return !!localStorage.getItem('portal_token')
}

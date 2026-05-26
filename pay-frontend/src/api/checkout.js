const SERVER = import.meta.env.VITE_SERVER_BASE || 'http://localhost:8082'

async function request(path, opts = {}) {
  const res = await fetch(SERVER + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  const json = await res.json()
  if (!json.success) {
    throw new Error(json.error?.message || 'Request failed')
  }
  return json.data
}

export async function getCheckout(sessionId) {
  return request('/api/checkout/' + sessionId)
}

export async function activatePayment(sessionId, channel) {
  return request('/api/checkout/' + sessionId + '/activate', {
    method: 'POST',
    body: JSON.stringify({ channel }),
  })
}

export async function getPaymentStatus(sessionId) {
  return request('/api/checkout/' + sessionId + '/payment-status')
}

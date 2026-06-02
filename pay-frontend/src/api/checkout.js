const SERVER = import.meta.env.VITE_SERVER_BASE || 'http://localhost:8082'

async function request(path, opts = {}) {
  let res
  try {
    res = await fetch(SERVER + path, {
      headers: { 'Content-Type': 'application/json' },
      ...opts,
    })
  } catch (e) {
    throw new Error('网络连接失败，请稍后重试')
  }
  const json = await res.json()
  if (!json.success) {
    throw new Error(json.error?.message || 'Request failed')
  }
  return json.data
}

export async function getCheckout(sessionId) {
  return request('/api/checkout/' + sessionId)
}

export async function activatePayment(sessionId, channel, tradeType) {
  const body = { channel }
  if (tradeType) body.trade_type = tradeType
  return request('/api/checkout/' + sessionId + '/activate', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function getPaymentStatus(sessionId) {
  return request('/api/checkout/' + sessionId + '/payment-status')
}

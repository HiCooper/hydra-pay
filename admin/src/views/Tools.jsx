import { useState } from 'react'

const H = { 'X-Admin-Key': 'admin-dev-key', 'Content-Type': 'application/json' }

export default function Tools() {
  const [tab, setTab] = useState('quickpay')

  return (
    <div>
      <h2 className="text-lg font-bold mb-5" style={{ color: 'var(--color-text-primary)' }}>测试工具</h2>

      <div className="flex gap-1 mb-5 p-1 rounded-lg" style={{ background: '#f1f5f9', width: 'fit-content' }}>
        {[
          { key: 'quickpay', label: '快速下单' },
          { key: 'callback', label: '回调模拟' },
          { key: 'webhook', label: 'Webhook 测试' },
          { key: 'connectivity', label: '连通性检查' },
        ].map(t => (
          <button key={t.key} onClick={() => setTab(t.key)}
            className="px-4 py-1.5 rounded-md text-xs font-medium transition-all"
            style={tab === t.key ? { background: '#fff', color: 'var(--color-text-primary)', boxShadow: '0 1px 3px rgba(0,0,0,0.1)' } : { background: 'transparent', color: 'var(--color-text-secondary)' }}
          >{t.label}</button>
        ))}
      </div>

      {tab === 'quickpay' && <QuickPay />}
      {tab === 'callback' && <CallbackSimulator />}
      {tab === 'webhook' && <WebhookTester />}
      {tab === 'connectivity' && <Connectivity />}
    </div>
  )
}

function QuickPay() {
  const [form, setForm] = useState({ channel: 'alipay', trade_type: 'native', amount: 1, description: '测试订单', app_key: 'test-pay-key-001' })
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setLoading(true)
    const res = await fetch('/v1/payments/create', {
      method: 'POST', headers: { 'X-API-Key': form.app_key, 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: 'test_user', amount: parseInt(form.amount), channel: form.channel, trade_type: form.trade_type, description: form.description, channel_app_id: form.channel === 'wechat' ? 'wx_test' : undefined, sub_merchant_id: form.sub_merchant_id || undefined, notify_url: form.notify_url || undefined })
    })
    const d = await res.json()
    setResult(d)
    setLoading(false)
  }

  return (
    <div className="grid grid-cols-2 gap-5">
      <div className="card p-5">
        <h3 className="text-sm font-semibold mb-4">创建测试支付</h3>
        <form onSubmit={submit} className="flex flex-col gap-3">
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>渠道</label>
            <select value={form.channel} onChange={e => setForm({ ...form, channel: e.target.value })} className="w-full">
              <option value="alipay">支付宝</option>
              <option value="wechat">微信</option>
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>支付方式</label>
            <select value={form.trade_type} onChange={e => setForm({ ...form, trade_type: e.target.value })} className="w-full">
              <option value="native">扫码支付</option>
              <option value="h5">H5 支付</option>
              <option value="jsapi">JSAPI 支付</option>
              <option value="app">App 支付</option>
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>金额 (分)</label>
            <input type="number" value={form.amount} onChange={e => setForm({ ...form, amount: e.target.value })} className="w-full" min="1" />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>描述</label>
            <input value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} className="w-full" />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>子商户 ID (服务商模式)</label>
            <input value={form.sub_merchant_id || ''} onChange={e => setForm({ ...form, sub_merchant_id: e.target.value })} className="w-full" placeholder="可选" />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>回调地址 (覆盖默认)</label>
            <input value={form.notify_url || ''} onChange={e => setForm({ ...form, notify_url: e.target.value })} className="w-full" placeholder="留空使用默认回调地址" />
          </div>
          <div>
            <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>API Key</label>
            <input value={form.app_key} onChange={e => setForm({ ...form, app_key: e.target.value })} className="w-full" />
          </div>
          <button type="submit" disabled={loading} className="btn btn-primary w-full justify-center mt-2">
            {loading ? '创建中...' : '创建支付'}
          </button>
        </form>
      </div>
      <div className="card p-5">
        <h3 className="text-sm font-semibold mb-4">结果</h3>
        {result ? (
          <div className="text-xs font-mono" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
            <div className="mb-2"><span className="badge" style={{ background: result.success ? '#ecfdf5' : '#fef2f2', color: result.success ? '#059669' : '#dc2626' }}>{result.success ? '成功' : '失败'}</span></div>
            {result.success ? (
              <div className="flex flex-col gap-2">
                <div><span className="text-text-muted">Payment ID:</span> <code>{result.data?.payment_id}</code></div>
                <div><span className="text-text-muted">Status:</span> {result.data?.status}</div>
                {result.data?.qr_code_url && <div><span className="text-text-muted">QR Code:</span> <a href={result.data.qr_code_url} target="_blank" className="text-accent">{result.data.qr_code_url.slice(0, 50)}...</a></div>}
                {result.data?.payment_url && <div><span className="text-text-muted">Payment URL:</span> <a href={result.data.payment_url} target="_blank" className="text-accent underline">打开支付页</a></div>}
              </div>
            ) : (
              <div className="flex flex-col gap-1">
                <div><span className="text-text-muted">Error Code:</span> {result.error?.code}</div>
                <div><span className="text-text-muted">Message:</span> {result.error?.message}</div>
              </div>
            )}
          </div>
        ) : <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>填写表单并点击「创建支付」查看结果</p>}
      </div>
    </div>
  )
}

function CallbackSimulator() {
  const [paymentId, setPaymentId] = useState('')
  const [result, setResult] = useState(null)

  async function simulate() {
    const res = await fetch('/api/admin/tools/simulate-callback', { method: 'POST', headers: H, body: JSON.stringify({ payment_id: paymentId, status: 'paid' }) })
    setResult(await res.json())
  }

  return (
    <div className="card p-5 max-w-lg">
      <h3 className="text-sm font-semibold mb-4">回调模拟</h3>
      <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>直接标记指定订单为已支付状态，用于测试 webhook 通知和状态变更流程。不会真正调用支付宝/微信接口。</p>
      <div className="flex gap-3 mb-4">
        <input value={paymentId} onChange={e => setPaymentId(e.target.value)} placeholder="输入 payment_id" className="flex-1" />
        <button onClick={simulate} className="btn btn-primary">模拟回调</button>
      </div>
      {result && (
        <div className="text-xs font-mono p-3 rounded-lg" style={{ background: result.success ? '#ecfdf5' : '#fef2f2' }}>
          {result.success ? '✅ ' + result.data?.message : '❌ ' + (result.error?.message || '失败')}
          {result.data?.payment && <div className="mt-1" style={{ color: 'var(--color-text-secondary)' }}>Status: {result.data.payment.Status}</div>}
        </div>
      )}
    </div>
  )
}

function WebhookTester() {
  const [appId, setAppId] = useState('')
  const [apps, setApps] = useState([])
  const [result, setResult] = useState(null)

  async function loadApps() {
    const res = await fetch('/api/admin/apps', { headers: H })
    const d = await res.json()
    setApps(d.data || [])
  }
  useState(() => { loadApps() }, [])

  async function test() {
    const res = await fetch('/api/admin/tools/test-webhook', { method: 'POST', headers: H, body: JSON.stringify({ app_id: appId }) })
    setResult(await res.json())
  }

  return (
    <div className="card p-5 max-w-lg">
      <h3 className="text-sm font-semibold mb-4">Webhook 推送测试</h3>
      <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>向已配置 webhook_url 的应用发送一条测试消息，验证回调地址可达性。</p>
      <div className="flex gap-3 mb-4">
        <select value={appId} onChange={e => setAppId(e.target.value)} className="flex-1">
          <option value="">选择应用</option>
          {apps.filter(a => a.WebhookURL).map(a => <option key={a.ID} value={a.ID}>{a.Name} ({a.WebhookURL?.slice(0, 40)})</option>)}
        </select>
        <button onClick={test} disabled={!appId} className="btn btn-primary">发送测试</button>
      </div>
      {result && (
        <div className="text-xs font-mono p-3 rounded-lg" style={{ background: result.success ? '#ecfdf5' : '#fef2f2' }}>
          <div>{result.success ? '✅ ' + result.data?.message : '❌ ' + (result.error?.message || '失败')}</div>
          {result.data?.response_code && <div className="mt-1">HTTP {result.data.response_code}</div>}
        </div>
      )}
    </div>
  )
}

function Connectivity() {
  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)

  async function check() {
    setLoading(true)
    const res = await fetch('/api/admin/tools/connectivity', { headers: H })
    setResults((await res.json()).data?.results || [])
    setLoading(false)
  }

  return (
    <div className="card p-5 max-w-lg">
      <h3 className="text-sm font-semibold mb-4">网关连通性检查</h3>
      <p className="text-xs mb-4" style={{ color: 'var(--color-text-muted)' }}>检测支付宝和微信支付网关的可达性。</p>
      <button onClick={check} disabled={loading} className="btn btn-primary mb-4">{loading ? '检测中...' : '开始检测'}</button>
      {results && (
        <table>
          <thead><tr><th>渠道</th><th>网关</th><th>状态</th><th>延迟</th></tr></thead>
          <tbody>
            {results.map((r, i) => (
              <tr key={i}>
                <td className="font-medium">{r.channel}</td>
                <td className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>{r.gateway}</td>
                <td><span className="badge" style={{ background: r.status.startsWith('HTTP 2') || r.status.startsWith('HTTP 3') ? '#ecfdf5' : '#fef2f2', color: r.status === 'unreachable' ? '#dc2626' : '#059669' }}>{r.status}</span></td>
                <td className="text-xs font-mono">{r.latency}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

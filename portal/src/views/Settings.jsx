import { useState } from 'react'
import { api } from '../api/index.js'

export default function Settings({ app, onUpdate }) {
  const [form, setForm] = useState({
    webhook_url: app.webhook_url || '',
    alipay_pid: app.alipay_pid || '',
    wechat_sub_mchid: app.wechat_sub_mchid || '',
    wechat_sub_appid: app.wechat_sub_appid || '',
  })
  const [saved, setSaved] = useState(false)

  async function save(e) {
    e.preventDefault()
    await api.updateSettings(form)
    const updated = await api.me()
    onUpdate(updated)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div>
      <h2 className="text-lg font-bold mb-5" style={{ color: 'var(--color-text-primary)' }}>设置</h2>

      <div className="card p-5 mb-5">
        <h3 className="text-sm font-semibold mb-4">API Key</h3>
        <div className="flex items-center gap-3 mb-2">
          <code className="text-sm flex-1 p-2 rounded" style={{ background: '#f8fafc' }}>{app.api_key_full}</code>
          <button className="btn btn-ghost text-xs" onClick={() => { navigator.clipboard.writeText(app.api_key_full) }}>复制</button>
        </div>
        <p className="text-xs" style={{ color: 'var(--color-text-muted)' }}>此 Key 用于调用 /v1/* 支付 API，请妥善保管</p>
      </div>

      <div className="card p-5 mb-5">
        <h3 className="text-sm font-semibold mb-4">子商户配置</h3>
        <form onSubmit={save}>
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>支付宝子商户 PID</label>
              <input value={form.alipay_pid} onChange={e => setForm({ ...form, alipay_pid: e.target.value })} className="w-full" placeholder="2088..." />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户号</label>
              <input value={form.wechat_sub_mchid} onChange={e => setForm({ ...form, wechat_sub_mchid: e.target.value })} className="w-full" placeholder="子商户 mchid" />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户 AppID</label>
              <input value={form.wechat_sub_appid} onChange={e => setForm({ ...form, wechat_sub_appid: e.target.value })} className="w-full" placeholder="wx..." />
            </div>
            <div>
              <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>Webhook 回调地址</label>
              <input value={form.webhook_url} onChange={e => setForm({ ...form, webhook_url: e.target.value })} className="w-full" placeholder="https://your-app.com/payment/callback" />
            </div>
          </div>
          <button type="submit" className="btn btn-primary">{saved ? '✓ 已保存' : '保存'}</button>
        </form>
      </div>
    </div>
  )
}

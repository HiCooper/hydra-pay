import { useState, useEffect } from 'react'
import { api } from '../api/index.js'

export default function Apps() {
  const [apps, setApps] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [copied, setCopied] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ name: '', alipay_pid: '', wechat_sub_mchid: '', wechat_sub_appid: '', webhook_url: '' })
  const [editing, setEditing] = useState(null)

  async function load() { setLoading(true); setApps(await api.listApps()); setLoading(false) }
  useEffect(() => { load() }, [])

  async function create(e) {
    e.preventDefault()
    if (!form.name) { setError('应用名称不能为空'); return }
    try { await api.createApp(form); setForm({ name: '', alipay_pid: '', wechat_sub_mchid: '', wechat_sub_appid: '', webhook_url: '' }); setShowCreate(false); setError(''); load() }
    catch (err) { setError(err.message) }
  }

  async function saveEdit(e) {
    e.preventDefault()
    await api.updateApp(editing.ID, editing)
    setEditing(null); load()
  }

  async function copy(text) {
    try { await navigator.clipboard.writeText(text); setCopied(text); setTimeout(() => setCopied(''), 2000) } catch (e) { }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-bold" style={{ color: 'var(--color-text-primary)' }}>应用管理</h2>
        <button className="btn btn-primary" onClick={() => setShowCreate(!showCreate)}>
          {showCreate ? '取消' : '+ 创建应用'}
        </button>
      </div>

      {showCreate && (
        <div className="card p-5 mb-5">
          <h3 className="text-sm font-semibold mb-4">创建应用</h3>
          <form onSubmit={create}>
            <div className="grid grid-cols-4 gap-3 mb-4">
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>应用名称 *</label>
                <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} placeholder="接入方名称" className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>支付宝子商户 PID</label>
                <input value={form.alipay_pid} onChange={e => setForm({ ...form, alipay_pid: e.target.value })} placeholder="2088..." className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户号</label>
                <input value={form.wechat_sub_mchid} onChange={e => setForm({ ...form, wechat_sub_mchid: e.target.value })} placeholder="子商户 mchid" className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户 AppID</label>
                <input value={form.wechat_sub_appid} onChange={e => setForm({ ...form, wechat_sub_appid: e.target.value })} placeholder="wx..." className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>Webhook 回调地址</label>
                <input value={form.webhook_url} onChange={e => setForm({ ...form, webhook_url: e.target.value })} placeholder="https://..." className="w-full" />
              </div>
            </div>
            <div className="flex items-center gap-3">
              <button type="submit" className="btn btn-primary">创建</button>
              {error && <span className="text-xs" style={{ color: 'var(--color-danger)' }}>{error}</span>}
            </div>
          </form>
        </div>
      )}

      {editing && (
        <div className="card p-5 mb-5">
          <h3 className="text-sm font-semibold mb-4">编辑 · {editing.Name}</h3>
          <form onSubmit={saveEdit}>
            <div className="grid grid-cols-5 gap-3 mb-4">
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>名称</label>
                <input value={editing.Name} onChange={e => setEditing({ ...editing, Name: e.target.value })} className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>支付宝 PID</label>
                <input value={editing.AlipayPID || ''} onChange={e => setEditing({ ...editing, AlipayPID: e.target.value })} className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户号</label>
                <input value={editing.WechatSubMchid || ''} onChange={e => setEditing({ ...editing, WechatSubMchid: e.target.value })} className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>微信子商户 AppID</label>
                <input value={editing.WechatSubAppid || ''} onChange={e => setEditing({ ...editing, WechatSubAppid: e.target.value })} className="w-full" />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>Webhook 回调地址</label>
                <input value={editing.WebhookURL || ''} onChange={e => setEditing({ ...editing, WebhookURL: e.target.value })} className="w-full" placeholder="https://..." />
              </div>
              <div>
                <label className="block text-xs font-medium mb-1" style={{ color: 'var(--color-text-secondary)' }}>状态</label>
                <select value={editing.Status} onChange={e => setEditing({ ...editing, Status: e.target.value })} className="w-full">
                  <option value="active">active</option>
                  <option value="disabled">disabled</option>
                </select>
              </div>
            </div>
            <div className="flex gap-2">
              <button type="submit" className="btn btn-primary">保存</button>
              <button type="button" className="btn btn-ghost" onClick={() => setEditing(null)}>取消</button>
            </div>
          </form>
        </div>
      )}

      <div className="card">
        <table>
          <thead><tr><th>名称</th><th>API Key</th><th>支付宝 PID</th><th>微信子商户号</th><th>状态</th><th className="w-24">操作</th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan="6" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>加载中...</td></tr>}
            {!loading && apps.length === 0 && <tr><td colSpan="6" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>暂无应用，点击右上角创建</td></tr>}
            {!loading && apps.map(a => (
              <tr key={a.ID}>
                <td className="font-medium">{a.Name}</td>
                <td>
                  <code className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>{a.APIKey.slice(0, 16)}...</code>
                  <button className="btn btn-ghost btn-sm ml-1" onClick={() => copy(a.APIKey)} title="复制">
                    {copied === a.APIKey ? '✓' : '⎘'}
                  </button>
                </td>
                <td style={{ color: a.AlipayPID ? 'var(--color-text-primary)' : 'var(--color-text-muted)' }}>{a.AlipayPID || '—'}</td>
                <td style={{ color: a.WechatSubMchid ? 'var(--color-text-primary)' : 'var(--color-text-muted)' }}>{a.WechatSubMchid || '—'}</td>
                <td><span className={'badge ' + (a.Status === 'active' ? 'badge-success' : 'badge-neutral')}>{a.Status}</span></td>
                <td><button className="btn btn-ghost btn-sm" onClick={() => setEditing({ ...a })}>编辑</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

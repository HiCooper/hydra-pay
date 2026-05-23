import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/index.js'

const badgeMap = { paid: 'badge-success', processing: 'badge-info', pending: 'badge-warning', failed: 'badge-danger', cancelled: 'badge-neutral', refunded: 'badge-neutral' }

export default function Orders() {
  const [orders, setOrders] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [f, setF] = useState({ status: '', channel: '' })

  async function load() {
    setLoading(true)
    const params = new URLSearchParams()
    if (f.status) params.set('status', f.status)
    if (f.channel) params.set('channel', f.channel)
    const d = await api.listOrders(params.toString())
    setOrders(d.orders || [])
    setTotal(d.total || 0)
    setLoading(false)
  }
  useEffect(() => { load() }, [f])

  function exportCSV() {
    const params = new URLSearchParams()
    if (f.status) params.set('status', f.status)
    if (f.channel) params.set('channel', f.channel)
    window.open('/api/admin/orders/export?' + params, '_blank')
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-bold" style={{ color: 'var(--color-text-primary)' }}>支付订单</h2>
        <button className="btn btn-ghost" onClick={exportCSV}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3"/></svg>
          导出 CSV
        </button>
      </div>

      <div className="card">
        <div className="flex items-center gap-3 p-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <select value={f.status} onChange={e => setF({ ...f, status: e.target.value })} className="w-28">
            <option value="">全部状态</option>
            <option value="pending">待支付</option>
            <option value="processing">处理中</option>
            <option value="paid">已支付</option>
            <option value="failed">失败</option>
          </select>
          <select value={f.channel} onChange={e => setF({ ...f, channel: e.target.value })} className="w-28">
            <option value="">全部渠道</option>
            <option value="alipay">支付宝</option>
            <option value="wechat">微信</option>
          </select>
          <span className="text-xs ml-auto" style={{ color: 'var(--color-text-muted)' }}>共 {total} 条</span>
        </div>
        <table>
          <thead><tr><th>订单 ID</th><th>渠道</th><th>金额</th><th>状态</th><th>外部交易号</th><th>时间</th><th className="w-16"></th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan="7" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>加载中...</td></tr>}
            {!loading && orders.length === 0 && <tr><td colSpan="7" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>暂无数据</td></tr>}
            {!loading && orders.map(o => (
              <tr key={o.ID}>
                <td><code className="text-xs font-mono" style={{ color: 'var(--color-text-secondary)' }}>{o.ID.slice(0, 8)}</code></td>
                <td className="font-medium">{o.Channel === 'alipay' ? '支付宝' : o.Channel === 'wechat' ? '微信' : o.Channel}</td>
                <td className="font-mono font-medium">¥{(o.Amount / 100).toFixed(2)}</td>
                <td><span className={'badge ' + (badgeMap[o.Status] || 'badge-neutral')}>{o.Status}</span></td>
                <td className="text-xs" style={{ color: o.ExternalID ? 'var(--color-text-secondary)' : 'var(--color-text-muted)', maxWidth: 180 }} title={o.ExternalID}>
                  <span className="block truncate">{o.ExternalID || '—'}</span>
                </td>
                <td className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>{new Date(o.CreatedAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</td>
                <td><Link to={'/orders/' + o.ID} className="text-xs font-medium" style={{ color: 'var(--color-accent)' }}>详情</Link></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

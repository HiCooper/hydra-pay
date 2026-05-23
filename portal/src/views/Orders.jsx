import { useState, useEffect } from 'react'
import { api } from '../api/index.js'

const badgeMap = { paid: 'badge-success', processing: 'badge-info', pending: 'badge-warning', failed: 'badge-danger' }

export default function Orders() {
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(true)
  const [expanded, setExpanded] = useState(null)

  useEffect(() => { api.orders().then(d => { setOrders(d.orders || []); setLoading(false) }) }, [])

  async function viewEvents(id) {
    if (expanded === id) { setExpanded(null); return }
    setExpanded(id)
  }

  return (
    <div>
      <h2 className="text-lg font-bold mb-5" style={{ color: 'var(--color-text-primary)' }}>支付订单</h2>
      <div className="card">
        <table>
          <thead><tr><th>订单 ID</th><th>渠道</th><th>金额</th><th>状态</th><th>时间</th><th></th></tr></thead>
          <tbody>
            {loading && <tr><td colSpan="6" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>加载中...</td></tr>}
            {!loading && orders.length === 0 && <tr><td colSpan="6" className="text-center py-10" style={{ color: 'var(--color-text-muted)' }}>暂无订单</td></tr>}
            {orders.map(o => (
              <tr key={o.ID}>
                <td><code className="text-xs">{o.ID.slice(0, 8)}</code></td>
                <td>{o.Channel}</td>
                <td className="font-mono font-medium">¥{(o.Amount / 100).toFixed(2)}</td>
                <td><span className={'badge ' + (badgeMap[o.Status] || 'badge-neutral')}>{o.Status}</span></td>
                <td className="text-xs" style={{ color: 'var(--color-text-secondary)' }}>{new Date(o.CreatedAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</td>
                <td><button className="btn btn-ghost text-xs" onClick={() => viewEvents(o.ID)}>{expanded === o.ID ? '收起' : '事件'}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {expanded && <EventTimeline paymentId={expanded} />}
    </div>
  )
}

function EventTimeline({ paymentId }) {
  const [events, setEvents] = useState([])
  useEffect(() => { api.orderDetail(paymentId).then(d => setEvents(d.events || [])) }, [paymentId])
  const labels = { created: '创建', channel_request: '渠道请求', callback_received: '回调到达', status_changed: '状态变更', webhook_sent: 'Webhook 通知' }

  return (
    <div className="card mt-4 p-5">
      <h3 className="text-sm font-semibold mb-3">事件时间线</h3>
      <table>
        <thead><tr><th>时间</th><th>事件</th><th>详情</th></tr></thead>
        <tbody>
          {events.map(e => (
            <tr key={e.ID}>
              <td className="text-xs whitespace-nowrap">{new Date(e.CreatedAt).toLocaleString('zh-CN')}</td>
              <td><span className="badge badge-info">{labels[e.Type] || e.Type}</span></td>
              <td className="text-xs" style={{ maxWidth: 300 }}>
                {e.Error ? <span style={{ color: 'var(--color-danger)' }}>{e.Error}</span> : e.Result ? <code className="text-xs">{e.Result.slice(0, 80)}</code> : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '../api/index.js'

const badgeMap = { paid: 'badge-success', processing: 'badge-info', pending: 'badge-warning', failed: 'badge-danger' }

export default function OrderDetail() {
  const { id } = useParams()
  const [order, setOrder] = useState(null)
  const [events, setEvents] = useState([])
  const [appName, setAppName] = useState('')

  useEffect(() => {
    api.getOrder(id).then(d => { setOrder(d.payment); setEvents(d.events || []); setAppName(d.app_name || '-') })
  }, [id])

  if (!order) return <div className="flex items-center justify-center h-64" style={{ color: 'var(--color-text-muted)' }}>加载中...</div>

  const fields = [
    ['订单 ID', <code key="id" className="text-xs">{order.ID}</code>],
    ['应用', appName],
    ['渠道', order.Channel === 'alipay' ? '支付宝' : order.Channel === 'wechat' ? '微信支付' : order.Channel],
    ['金额', <span key="amt" className="font-mono font-medium">¥{(order.Amount / 100).toFixed(2)} <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{order.Currency}</span></span>],
    ['状态', <span key="st" className={'badge ' + (badgeMap[order.Status] || 'badge-neutral')}>{order.Status}</span>],
    ['外部交易号', <code key="ext" className="text-xs">{order.ExternalID || '—'}</code>],
    ['描述', order.Description || '—'],
    ['创建时间', new Date(order.CreatedAt).toLocaleString('zh-CN')],
    ['支付时间', order.PaidAt ? new Date(order.PaidAt).toLocaleString('zh-CN') : '—'],
  ]

  const eventTypeLabel = { created: '创建', channel_request: '渠道请求', callback_received: '回调到达', status_changed: '状态变更', webhook_sent: 'Webhook' }

  return (
    <div>
      <Link to="/orders" className="inline-flex items-center gap-1 text-xs font-medium mb-4" style={{ color: 'var(--color-accent)' }}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>
        返回订单列表
      </Link>

      <div className="card mb-5">
        <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
          <h3 className="text-sm font-semibold">订单详情</h3>
        </div>
        <div className="p-5">
          <div className="grid grid-cols-3 gap-y-3 gap-x-8">
            {fields.map(([label, value], i) => (
              <div key={i} className="flex flex-col gap-0.5">
                <span className="text-xs" style={{ color: 'var(--color-text-muted)' }}>{label}</span>
                <span className="text-sm">{value}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {events.length > 0 && (
        <div className="card">
          <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--color-border)' }}>
            <h3 className="text-sm font-semibold">事件时间线</h3>
          </div>
          <table>
            <thead><tr><th>时间</th><th>事件</th><th>渠道</th><th>详情</th></tr></thead>
            <tbody>
              {events.map(e => (
                <tr key={e.ID}>
                  <td className="text-xs whitespace-nowrap" style={{ color: 'var(--color-text-secondary)' }}>{new Date(e.CreatedAt).toLocaleString('zh-CN')}</td>
                  <td><span className="badge badge-info">{eventTypeLabel[e.Type] || e.Type}</span></td>
                  <td className="text-xs">{e.Channel}</td>
                  <td className="text-xs" style={{ maxWidth: 300 }}>
                    {e.Error ? <span style={{ color: 'var(--color-danger)' }}>{e.Error}</span>
                      : e.Result ? <code className="text-xs block truncate" title={e.Result}>{e.Result.slice(0, 80)}</code>
                        : <span style={{ color: 'var(--color-text-muted)' }}>—</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

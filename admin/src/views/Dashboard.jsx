import { useState, useEffect } from 'react'
import { api } from '../api/index.js'

export default function Dashboard() {
  const [d, setD] = useState(null)
  useEffect(() => { api.dashboard().then(setD) }, [])
  if (!d) return <Loading />

  const stats = [
    { label: '今日订单', value: d.today_orders },
    { label: '支付成功', value: d.today_paid },
    { label: '成功率', value: (d.success_rate || 0).toFixed(1) + '%' },
    { label: '今日收入', value: '¥' + (d.today_revenue || 0).toFixed(2) },
  ]

  return (
    <div>
      <h2 className="text-lg font-bold mb-5" style={{ color: 'var(--color-text-primary)' }}>仪表盘</h2>
      <div className="grid grid-cols-4 gap-4 mb-6">
        {stats.map((s, i) => (
          <div key={i} className="stat-card">
            <div className="stat-value">{s.value}</div>
            <div className="stat-label">{s.label}</div>
          </div>
        ))}
      </div>
      <div className="card p-5">
        <h3 className="text-sm font-semibold mb-4" style={{ color: 'var(--color-text-primary)' }}>渠道分布 · 今日</h3>
        <table>
          <thead><tr><th>渠道</th><th className="text-right">订单数</th></tr></thead>
          <tbody>
            {(d.channel_stats || []).length === 0
              ? <tr><td colSpan="2" className="text-center" style={{ color: 'var(--color-text-muted)' }}>暂无数据</td></tr>
              : (d.channel_stats || []).map(c => (
                  <tr key={c.Channel}>
                    <td className="font-medium">{c.Channel === 'alipay' ? '支付宝' : c.Channel === 'wechat' ? '微信支付' : c.Channel}</td>
                    <td className="text-right font-mono">{c.Count}</td>
                  </tr>
                ))
            }
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Loading() {
  return <div className="flex items-center justify-center h-64" style={{ color: 'var(--color-text-muted)' }}>
    <div className="flex items-center gap-2">
      <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" /><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
      加载中...
    </div>
  </div>
}

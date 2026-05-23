import { useState, useEffect } from 'react'
import { api } from '../api/index.js'

export default function Dashboard({ app }) {
  const [d, setD] = useState(null)
  useEffect(() => { api.dashboard().then(setD) }, [])

  if (!d) return <p style={{ color: 'var(--color-text-muted)' }}>加载中...</p>

  return (
    <div>
      <h2 className="text-lg font-bold mb-1" style={{ color: 'var(--color-text-primary)' }}>{app.name}</h2>
      <p className="text-xs mb-5" style={{ color: 'var(--color-text-muted)' }}>API Key: {app.api_key_preview}</p>

      <div className="grid grid-cols-3 gap-4 mb-6">
        {[
          { label: '今日订单', value: d.today_orders },
          { label: '支付成功', value: d.today_paid },
          { label: '成功率', value: (d.success_rate || 0).toFixed(1) + '%' },
          { label: '今日收入', value: '¥' + (d.today_revenue || 0).toFixed(2) },
        ].map((s, i) => (
          <div key={i} className="stat-card">
            <div className="stat-value">{s.value}</div>
            <div className="stat-label">{s.label}</div>
          </div>
        ))}
      </div>

      <div className="card p-5">
        <h3 className="text-sm font-semibold mb-4">快速开始</h3>
        <p className="text-xs mb-4" style={{ color: 'var(--color-text-secondary)' }}>使用以下命令创建一笔测试支付：</p>
        <pre><code>{`curl -X POST https://your-domain.com/v1/payments/create \\
  -H "X-API-Key: ${app.api_key_full}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "test_user",
    "amount": 1,
    "channel": "alipay",
    "trade_type": "native",
    "description": "测试订单"
  }'`}</code></pre>
        <p className="text-xs mt-3" style={{ color: 'var(--color-text-muted)' }}>更多示例请查看 <a href="/docs-site" target="_blank" style={{ color: 'var(--color-accent)' }}>API 文档</a></p>
      </div>
    </div>
  )
}

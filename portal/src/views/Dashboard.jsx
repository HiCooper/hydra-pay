import { useState, useEffect } from 'react'
import { Card, Statistic, Row, Col, Spin } from 'antd'
import { ShoppingCartOutlined, CheckCircleOutlined, PercentageOutlined, DollarOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Dashboard({ data }) {
  const [d, setD] = useState(null)
  useEffect(() => { api.dashboard().then(setD).catch(() => {}) }, [])

  if (!d) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  const { merchant, apps } = data || {}
  const firstApp = apps?.[0]

  const statCards = [
    {
      title: '今日订单',
      value: d.today_orders,
      prefix: <ShoppingCartOutlined style={{ fontSize: 20, color: '#de481b' }} />,
      color: '#de481b',
    },
    {
      title: '支付成功',
      value: d.today_paid,
      prefix: <CheckCircleOutlined style={{ fontSize: 20, color: '#04d66f' }} />,
      color: '#04d66f',
    },
    {
      title: '成功率',
      value: (d.success_rate || 0).toFixed(1),
      suffix: '%',
      prefix: <PercentageOutlined style={{ fontSize: 20, color: '#f2921b' }} />,
      color: '#f2921b',
      precision: 1,
    },
    {
      title: '今日收入',
      value: d.today_revenue || 0,
      prefix: <DollarOutlined style={{ fontSize: 20, color: '#1a1a1a' }} />,
      color: '#1a1a1a',
      precision: 2,
      prefixText: '¥',
    },
  ]

  return (
    <div>
      <h2 className="page-heading">{merchant?.name || '概览'}</h2>
      <p className="page-subtitle">
        {apps?.length || 0} 个应用 · 数据截止至当前
      </p>

      {/* Stats row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 28 }}>
        {statCards.map((s, i) => (
          <Col xs={24} sm={12} lg={6} key={i}>
            <Card
              style={{ borderRadius: 8, height: '100%' }}
              styles={{ body: { padding: '20px 22px' } }}
            >
              <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12, color: '#6b6b6b', fontWeight: 450, marginBottom: 8 }}>
                    {s.title}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 2 }}>
                    {s.prefixText && (
                      <span style={{ fontSize: 16, fontWeight: 500, color: '#6b6b6b' }}>{s.prefixText}</span>
                    )}
                    <span style={{ fontSize: 28, fontWeight: 600, color: '#1a1a1a', lineHeight: 1.2 }}>
                      {typeof s.value === 'number' ? s.value.toLocaleString('zh-CN', {
                        minimumFractionDigits: s.precision || 0,
                        maximumFractionDigits: s.precision || 0,
                      }) : s.value}
                    </span>
                    {s.suffix && (
                      <span style={{ fontSize: 14, fontWeight: 500, color: '#6b6b6b' }}>{s.suffix}</span>
                    )}
                  </div>
                </div>
                <div style={{
                  width: 40, height: 40, borderRadius: 8,
                  background: `${s.color}08`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  flexShrink: 0, marginLeft: 12,
                }}>
                  <span style={{ color: s.color, fontSize: 20, lineHeight: 1 }}>{s.prefix}</span>
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Quick start card */}
      {firstApp && (
        <Card
          title={<span style={{ fontSize: 15, fontWeight: 600, color: '#1a1a1a' }}>快速开始</span>}
          style={{ borderRadius: 8 }}
          styles={{ header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' }, body: { padding: '20px 24px' } }}
        >
          <p style={{ color: '#6b6b6b', fontSize: 13, marginBottom: 14, lineHeight: 1.6 }}>
            使用以下命令创建一笔测试支付：
          </p>
          <pre style={{
            background: '#1a1a1a',
            color: '#e0e0e0',
            borderRadius: 8,
            padding: '18px 22px',
            overflowX: 'auto',
            fontSize: 13,
            lineHeight: 1.7,
            margin: 0,
          }}>
            <code style={{
              background: 'none', padding: 0, color: '#04d66f',
              fontSize: 'inherit', fontFamily: "'SF Mono', Menlo, Monaco, 'Courier New', monospace",
            }}>{`curl -X POST https://your-domain.com/v1/payments/create \\
  -H "X-API-Key: ${firstApp.api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "test_user",
    "amount": 1,
    "channel": "alipay",
    "trade_type": "native",
    "description": "测试订单"
  }'`}</code>
          </pre>
        </Card>
      )}
    </div>
  )
}

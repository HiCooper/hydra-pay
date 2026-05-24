import { useState, useEffect } from 'react'
import { Card, Statistic, Row, Col, Spin } from 'antd'
import { ShoppingCartOutlined, CheckCircleOutlined, PercentageOutlined, DollarOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Dashboard({ app }) {
  const [d, setD] = useState(null)
  useEffect(() => { api.dashboard().then(setD) }, [])

  if (!d) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 4 }}>{app.name}</h2>
      <p style={{ color: '#94a3b8', fontSize: 13, marginBottom: 20 }}>API Key: {app.api_key_preview}</p>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card><Statistic title="今日订单" value={d.today_orders} prefix={<ShoppingCartOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="支付成功" value={d.today_paid} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#10b981' }} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="成功率" value={(d.success_rate || 0).toFixed(1)} suffix="%" prefix={<PercentageOutlined />} precision={1} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="今日收入" value={d.today_revenue || 0} prefix={<DollarOutlined />} precision={2} prefix="¥" /></Card>
        </Col>
      </Row>

      <Card title="快速开始">
        <p style={{ color: '#64748b', fontSize: 13, marginBottom: 12 }}>使用以下命令创建一笔测试支付：</p>
        <pre style={{ background: '#1e293b', color: '#e2e8f0', borderRadius: 8, padding: '16px 20px', overflowX: 'auto', fontSize: 13, lineHeight: 1.6 }}>
          <code style={{ background: 'none', padding: 0, color: 'inherit', fontSize: 'inherit' }}>{`curl -X POST https://your-domain.com/v1/payments/create \\
  -H "X-API-Key: ${app.api_key_full}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "test_user",
    "amount": 1,
    "channel": "alipay",
    "trade_type": "native",
    "description": "测试订单"
  }'`}</code>
        </pre>
        <p style={{ color: '#94a3b8', fontSize: 13, marginTop: 12 }}>更多示例请查看 <a href="/docs-site" target="_blank" style={{ color: '#635bff' }}>API 文档</a></p>
      </Card>
    </div>
  )
}
import { useState, useEffect } from 'react'
import { Card, Statistic, Table, Row, Col, Spin } from 'antd'
import { ShoppingCartOutlined, CheckCircleOutlined, PercentageOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Dashboard() {
  const [d, setD] = useState(null)
  useEffect(() => { api.dashboard().then(setD) }, [])

  if (!d) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  const channelColumns = [
    { title: '渠道', dataIndex: 'Channel', render: v => v === 'alipay' ? '支付宝' : v === 'wechat' ? '微信支付' : v },
    { title: '订单数', dataIndex: 'Count', align: 'right' },
  ]

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>仪表盘</h2>

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
          <Card><Statistic title="今日收入" value={d.today_revenue || 0} prefix="¥" precision={2} /></Card>
        </Col>
      </Row>

      <Card title="渠道分布 · 今日">
        <Table
          columns={channelColumns}
          dataSource={d.channel_stats || []}
          rowKey="Channel"
          size="small"
          pagination={false}
          locale={{ emptyText: '暂无数据' }}
        />
      </Card>
    </div>
  )
}
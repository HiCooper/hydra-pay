import { useState, useEffect } from 'react'
import { Table, Tag, Spin } from 'antd'
import { api } from '../api/index.js'

const statusLabel = { pending: '待支付', processing: '支付中', paid: '支付成功', failed: '支付失败', cancelled: '已取消', refunded: '已退款' }
const statusColor = { pending: 'orange', processing: 'blue', paid: 'green', failed: 'red', cancelled: 'default', refunded: 'default' }
const chLabel = { alipay: '支付宝', wechat: '微信' }
const eventTypeLabel = { created: '创建', channel_request: '渠道请求', callback_received: '回调到达', status_changed: '状态变更', webhook_sent: 'Webhook' }
function fmtResult(v) { if (!v) return ''; return typeof v === 'string' ? v : JSON.stringify(v) }

export default function Orders() {
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.orders().then(d => { setOrders(d.orders || []); setLoading(false) })
  }, [])

  const columns = [
    {
      title: '订单 ID', dataIndex: 'TradeNo', width: 220,
      render: v => <code style={{ fontSize: 12 }}>{v}</code>,
    },
    { title: '渠道', dataIndex: 'Channel', width: 80, render: v => chLabel[v] || v },
    {
      title: '金额', dataIndex: 'Amount', width: 100,
      render: v => <span style={{ fontFamily: 'monospace' }}>¥{(v / 100).toFixed(2)}</span>,
    },
    {
      title: '状态', dataIndex: 'Status', width: 150,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}({v})</Tag>,
    },
    {
      title: '时间', dataIndex: 'CreatedAt', width: 170,
      render: v => new Date(v).toLocaleString('zh-CN'),
    },
  ]

  if (loading) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>支付订单</h2>
      <Table
        columns={columns}
        dataSource={orders}
        rowKey="ID"
        size="middle"
        locale={{ emptyText: '暂无订单' }}
        expandable={{
          rowExpandable: () => true,
          expandedRowRender: record => <EventTimeline paymentId={record.ID} />,
        }}
      />
    </div>
  )
}

function EventTimeline({ paymentId }) {
  const [events, setEvents] = useState(null)
  useEffect(() => { api.orderDetail(paymentId).then(d => setEvents(d.events || [])) }, [paymentId])

  if (!events) return <Spin size="small" />
  if (events.length === 0) return <span style={{ color: '#94a3b8', fontSize: 13 }}>暂无事件</span>

  const columns = [
    { title: '时间', dataIndex: 'CreatedAt', width: 170, render: v => new Date(v).toLocaleString('zh-CN') },
    { title: '事件', dataIndex: 'Type', width: 100, render: v => <Tag color="blue">{eventTypeLabel[v] || v}</Tag> },
    { title: '详情', dataIndex: 'Error', render: (err, record) => {
      if (err) return <span style={{ color: '#ef4444' }}>{err}</span>
      const text = fmtResult(record.Result)
      return text ? <code style={{ fontSize: 12 }} title={text}>{text.slice(0, 80)}</code>
        : <span style={{ color: '#94a3b8' }}>—</span>
    }},
  ]

  return <Table columns={columns} dataSource={events} rowKey="ID" size="small" pagination={false} style={{ margin: 0 }} />
}
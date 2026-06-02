import { useState, useEffect } from 'react'
import { Table, Tag, Spin } from 'antd'
import { api } from '../api/index.js'

const statusLabel = { pending: '待支付', processing: '支付中', paid: '支付成功', failed: '支付失败', create_failed: '创单失败', expired: '已过期', cancelled: '已取消', refunded: '已退款' }
const statusColor = { pending: 'orange', processing: 'blue', paid: 'green', failed: 'red', create_failed: 'red', expired: 'default', cancelled: 'default', refunded: 'default' }
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
      render: v => <code style={{ fontSize: 12, color: '#6b6b6b' }}>{v}</code>,
    },
    {
      title: '渠道', dataIndex: 'Channel', width: 80,
      render: v => (
        <span style={{ fontSize: 13, color: '#1a1a1a' }}>{chLabel[v] || v}</span>
      ),
    },
    {
      title: '金额', dataIndex: 'Amount', width: 100,
      render: v => (
        <span style={{ fontFamily: "'SF Mono', Menlo, Monaco, monospace", fontSize: 13, fontWeight: 500, color: '#1a1a1a' }}>
          ¥{(v / 100).toFixed(2)}
        </span>
      ),
    },
    {
      title: '状态', dataIndex: 'Status', width: 110,
      render: v => (
        <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>
      ),
    },
    {
      title: '时间', dataIndex: 'CreatedAt', width: 170,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{new Date(v).toLocaleString('zh-CN')}</span>,
    },
  ]

  if (loading) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  return (
    <div>
      <h2 className="page-heading">支付订单</h2>
      <p className="page-subtitle">所有支付订单记录</p>
      <div style={{
        background: '#fff',
        borderRadius: 8,
        border: '1px solid #e6e6e6',
        overflow: 'hidden',
      }}>
        <Table
          columns={columns}
          dataSource={orders}
          rowKey="ID"
          size="middle"
          pagination={{
            style: { padding: '0 16px' },
          }}
          locale={{ emptyText: <span style={{ color: '#999' }}>暂无订单</span> }}
          expandable={{
            rowExpandable: () => true,
            expandedRowRender: record => <EventTimeline paymentId={record.ID} />,
          }}
          style={{ margin: 0 }}
        />
      </div>
    </div>
  )
}

function EventTimeline({ paymentId }) {
  const [detail, setDetail] = useState(null)
  useEffect(() => { api.orderDetail(paymentId).then(d => setDetail(d)) }, [paymentId])

  if (!detail) return <Spin size="small" />

  const events = detail.events || []
  const refunds = detail.refunds || []

  if (events.length === 0 && refunds.length === 0) return <span style={{ color: '#999', fontSize: 13 }}>暂无事件</span>

  const eventColumns = [
    {
      title: '时间', dataIndex: 'CreatedAt', width: 170,
      render: v => <span style={{ fontSize: 12, color: '#6b6b6b' }}>{new Date(v).toLocaleString('zh-CN')}</span>,
    },
    {
      title: '事件', dataIndex: 'Type', width: 100,
      render: v => <Tag color="blue">{eventTypeLabel[v] || v}</Tag>,
    },
    {
      title: '详情', dataIndex: 'Error', render: (err, record) => {
        if (err) return <span style={{ color: '#df1b41', fontSize: 12 }}>{err}</span>
        const text = fmtResult(record.Result)
        return text
          ? <code style={{ fontSize: 12, color: '#6b6b6b' }} title={text}>{text.slice(0, 80)}</code>
          : <span style={{ color: '#999', fontSize: 12 }}>—</span>
      },
    },
  ]

  const refundStatusLabel = { processing: '退款中', success: '已退款', failed: '退款失败' }
  const refundStatusColor = { processing: 'blue', success: 'green', failed: 'red' }

  const refundColumns = [
    {
      title: '时间', dataIndex: 'CreatedAt', width: 170,
      render: v => <span style={{ fontSize: 12, color: '#6b6b6b' }}>{v ? new Date(v).toLocaleString('zh-CN') : '—'}</span>,
    },
    {
      title: '退款金额', dataIndex: 'RefundAmount', width: 100,
      render: v => v ? <span style={{ fontSize: 12, color: '#1a1a1a', fontWeight: 500 }}>¥{(v / 100).toFixed(2)}</span> : '—',
    },
    {
      title: '状态', dataIndex: 'Status', width: 90,
      render: v => <Tag color={refundStatusColor[v] || 'default'}>{refundStatusLabel[v] || v}</Tag>,
    },
    {
      title: '原因', dataIndex: 'RefundReason', width: 120,
      render: v => <span style={{ fontSize: 12, color: '#6b6b6b' }}>{v || '—'}</span>,
    },
    {
      title: '渠道退款号', dataIndex: 'ChannelRefundID', width: 140,
      render: v => v ? <code style={{ fontSize: 11, color: '#6b6b6b' }}>{v.slice(0, 20)}</code> : <span style={{ fontSize: 12, color: '#999' }}>—</span>,
    },
    {
      title: '备注', dataIndex: 'ErrorMsg', width: 120,
      render: v => v ? <span style={{ fontSize: 12, color: '#df1b41' }}>{v}</span> : <span style={{ fontSize: 12, color: '#999' }}>—</span>,
    },
  ]

  return (
    <div style={{ background: '#fafafa', borderRadius: 6, padding: 12, margin: '4px 0' }}>
      {refunds.length > 0 && (
        <div style={{ marginBottom: events.length > 0 ? 16 : 0 }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#6b6b6b', marginBottom: 8 }}>退款记录</div>
          <Table
            columns={refundColumns}
            dataSource={refunds}
            rowKey="ID"
            size="small"
            pagination={false}
            style={{ margin: 0 }}
          />
        </div>
      )}
      {events.length > 0 && (
        <div>
          {refunds.length > 0 && <div style={{ fontSize: 12, fontWeight: 600, color: '#6b6b6b', marginBottom: 8 }}>事件日志</div>}
          <Table
            columns={eventColumns}
            dataSource={events}
            rowKey="ID"
            size="small"
            pagination={false}
            style={{ margin: 0 }}
          />
        </div>
      )}
    </div>
  )
}

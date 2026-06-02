import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Descriptions, Tag, Table, Button, Result, Statistic } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

const statusLabel = { pending: '待支付', processing: '支付中', paid: '支付成功', failed: '支付失败', create_failed: '创建失败', expired: '已过期', cancelled: '已取消', refunded: '已退款' }
const statusColor = { pending: 'orange', processing: 'blue', paid: 'green', failed: 'red', create_failed: 'red', expired: 'default', cancelled: 'default', refunded: 'default' }
const chLabel = { alipay: '支付宝', wechat: '微信支付' }
const eventTypeLabel = { created: '创建', channel_request: '渠道请求', callback_received: '回调到达', status_changed: '状态变更', webhook_sent: 'Webhook', refund: '退款' }
function fmtResult(v) { if (!v) return ''; return typeof v === 'string' ? v : JSON.stringify(v) }

export default function OrderDetail() {
  const [searchParams] = useSearchParams()
  const id = searchParams.get('id')
  const navigate = useNavigate()
  const [order, setOrder] = useState(null)
  const [events, setEvents] = useState([])
  const [alipayCbs, setAlipayCbs] = useState([])
  const [wechatCbs, setWechatCbs] = useState([])
  const [refunds, setRefunds] = useState([])
  const [appName, setAppName] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) { setError('缺少订单 ID'); return }
    api.getOrder(id)
      .then(d => {
        setOrder(d.payment); setEvents(d.events || [])
        setAlipayCbs(d.alipay_callbacks || []); setWechatCbs(d.wechat_callbacks || [])
	        setRefunds(d.refunds || [])
        setAppName(d.app_name || '-')
      })
      .catch(err => setError(err.message))
  }, [id])

  if (error) return (
    <Result
      status="error" title="加载失败" subTitle={error}
      extra={<Button onClick={() => navigate('/orders')}>返回订单列表</Button>}
    />
  )

  if (!order) return <div style={{ textAlign: 'center', padding: 80, color: '#94a3b8' }}>加载中...</div>

  const deadline = new Date(order.CreatedAt).getTime() + 15 * 60 * 1000

  const items = [
    { key: 'trade_no', label: '订单 ID', children: <code>{order.TradeNo}</code> },
    { key: 'app', label: '应用', children: appName },
    { key: 'channel', label: '渠道', children: chLabel[order.Channel] || order.Channel },
    { key: 'amount', label: '金额', children: <span style={{ fontFamily: 'monospace' }}>¥{(order.Amount / 100).toFixed(2)} <span style={{ color: '#94a3b8', fontSize: 12 }}>{order.Currency}</span></span> },
    { key: 'status', label: '状态', children: <Tag color={statusColor[order.Status] || 'default'}>{statusLabel[order.Status] || order.Status}({order.Status})</Tag> },
    ...(order.Status === 'processing' ? [{ key: 'countdown', label: '支付倒计时', children:
      <Statistic.Countdown
        value={deadline}
        valueStyle={{ fontSize: 14, color: deadline < Date.now() ? '#ef4444' : '#f59e0b' }}
        format="mm:ss"
        onFinish={() => {}} />
    }] : []),
    { key: 'external_id', label: '渠道交易号', children: order.ExternalID ? <code>{order.ExternalID}</code> : '—' },
    { key: 'description', label: '描述', children: order.Description || '—' },
    { key: 'created', label: '创建时间', children: new Date(order.CreatedAt).toLocaleString('zh-CN') },
    { key: 'paid', label: '支付时间', children: order.PaidAt ? new Date(order.PaidAt).toLocaleString('zh-CN') : '—' },
  ]

  const eventColumns = [
    { title: '时间', dataIndex: 'CreatedAt', width: 170, render: v => new Date(v).toLocaleString('zh-CN') },
    { title: '事件', dataIndex: 'Type', width: 100, render: v => <Tag color="blue">{eventTypeLabel[v] || v}</Tag> },
    { title: '渠道', dataIndex: 'Channel', width: 80 },
    { title: '详情', dataIndex: 'Error', render: (err, record) => {
      if (err) return <span style={{ color: '#ef4444' }}>{err}</span>
      const text = fmtResult(record.Result)
      return text ? <code style={{ fontSize: 12 }} title={text}>{text.slice(0, 80)}</code>
        : <span style={{ color: '#94a3b8' }}>—</span>
    }},
  ]

  return (
    <div>
      <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/orders')} style={{ padding: 0, marginBottom: 16 }}>
        返回订单列表
      </Button>

      <Descriptions title="订单详情" bordered column={3} size="small" items={items} style={{ marginBottom: 24, background: '#fff' }} />

      {events.length > 0 && (
        <div style={{ marginBottom: 24 }}>
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>事件时间线</h3>
          <Table columns={eventColumns} dataSource={events} rowKey="ID" size="small" pagination={false} />
        </div>
      )}

      {alipayCbs.length > 0 && alipayCbs.map(cb => (
        <div key={cb.ID} style={{ marginBottom: 24 }}>
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>支付宝回调 · {cb.NotifyID}</h3>
          <Descriptions bordered column={3} size="small">
            <Descriptions.Item label="通知 ID">{cb.NotifyID}</Descriptions.Item>
            <Descriptions.Item label="通知类型">{cb.NotifyType}</Descriptions.Item>
            <Descriptions.Item label="通知时间">{cb.NotifyTime}</Descriptions.Item>
            <Descriptions.Item label="支付宝交易号"><code>{cb.TradeNo}</code></Descriptions.Item>
            <Descriptions.Item label="交易状态"><Tag color="blue">{cb.TradeStatus}</Tag></Descriptions.Item>
            <Descriptions.Item label="订单金额">¥{cb.TotalAmount}</Descriptions.Item>
            <Descriptions.Item label="实收金额">¥{cb.ReceiptAmount || '—'}</Descriptions.Item>
            <Descriptions.Item label="买家付款金额">¥{cb.BuyerPayAmount || '—'}</Descriptions.Item>
            <Descriptions.Item label="买家 ID">{cb.BuyerID || '—'}</Descriptions.Item>
            <Descriptions.Item label="买家账号">{cb.BuyerLogonID || '—'}</Descriptions.Item>
            <Descriptions.Item label="交易创建时间">{cb.GmtCreate || '—'}</Descriptions.Item>
            <Descriptions.Item label="交易付款时间">{cb.GmtPayment || '—'}</Descriptions.Item>
            <Descriptions.Item label="商品标题">{cb.Subject || '—'}</Descriptions.Item>
            <Descriptions.Item label="集分宝金额">{cb.PointAmount || '—'}</Descriptions.Item>
            <Descriptions.Item label="回传参数">{cb.PassbackParams || '—'}</Descriptions.Item>
          </Descriptions>
        </div>
      ))}

      {wechatCbs.length > 0 && wechatCbs.map(cb => (
        <div key={cb.ID} style={{ marginBottom: 24 }}>
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>微信回调 · {cb.NotificationID}</h3>
          <Descriptions bordered column={3} size="small">
            <Descriptions.Item label="通知 ID">{cb.NotificationID}</Descriptions.Item>
            <Descriptions.Item label="事件类型">{cb.EventType}</Descriptions.Item>
            <Descriptions.Item label="微信交易号"><code>{cb.TransactionID}</code></Descriptions.Item>
            <Descriptions.Item label="交易状态"><Tag color="blue">{cb.TradeState}</Tag></Descriptions.Item>
            <Descriptions.Item label="状态描述">{cb.TradeStateDesc || '—'}</Descriptions.Item>
            <Descriptions.Item label="交易类型">{cb.TradeType || '—'}</Descriptions.Item>
            <Descriptions.Item label="金额">¥{(cb.AmountTotal / 100).toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="用户支付金额">¥{(cb.AmountPayerTotal / 100).toFixed(2)}</Descriptions.Item>
            <Descriptions.Item label="币种">{cb.AmountCurrency}</Descriptions.Item>
            <Descriptions.Item label="付款银行">{cb.BankType || '—'}</Descriptions.Item>
            <Descriptions.Item label="支付成功时间">{cb.SuccessTime || '—'}</Descriptions.Item>
            <Descriptions.Item label="用户 OpenID">{cb.PayerOpenid || '—'}</Descriptions.Item>
            {cb.SubMchid && <Descriptions.Item label="子商户号">{cb.SubMchid}</Descriptions.Item>}
            {cb.SpMchid && <Descriptions.Item label="服务商号">{cb.SpMchid}</Descriptions.Item>}
            {cb.SpAppid && <Descriptions.Item label="服务商 AppID">{cb.SpAppid}</Descriptions.Item>}
            {cb.SubAppid && <Descriptions.Item label="子商户 AppID">{cb.SubAppid}</Descriptions.Item>}
            {cb.Attach && <Descriptions.Item label="附加数据">{cb.Attach}</Descriptions.Item>}
          </Descriptions>
        </div>
      ))}

      {refunds.length > 0 && refunds.map(r => (
        <div key={r.ID} style={{ marginBottom: 24 }}>
          <h3 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>退款记录 · {r.OutRequestNo}</h3>
          <Descriptions bordered column={3} size="small">
            <Descriptions.Item label="渠道">{r.Channel === 'alipay' ? '支付宝' : '微信'}</Descriptions.Item>
            <Descriptions.Item label="退款金额">¥{r.RefundFee || r.RefundAmount}</Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={r.Status === 'success' ? 'green' : 'red'}>{r.Status}</Tag></Descriptions.Item>
            <Descriptions.Item label="请求号"><code>{r.OutRequestNo}</code></Descriptions.Item>
            <Descriptions.Item label="渠道退款号"><code>{r.ChannelRefundID || '—'}</code></Descriptions.Item>
            <Descriptions.Item label="原因">{r.RefundReason || '—'}</Descriptions.Item>
            <Descriptions.Item label="时间">{new Date(r.CreatedAt).toLocaleString('zh-CN')}</Descriptions.Item>
          </Descriptions>
        </div>
      ))}
    </div>
  )
}
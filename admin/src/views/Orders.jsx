import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Table, Select, Input, Button, Tag, Space } from 'antd'
import { ExportOutlined, SearchOutlined } from '@ant-design/icons'
import { api, exportCSV as downloadCSV } from '../api/index.js'
import OrderDetail from './OrderDetail.jsx'

const statusLabel = { pending: '待支付', processing: '支付中', paid: '支付成功', failed: '支付失败', cancelled: '已取消', refunded: '已退款' }
const statusColor = { pending: 'orange', processing: 'blue', paid: 'green', failed: 'red', cancelled: 'default', refunded: 'default' }
const chLabel = { alipay: '支付宝', wechat: '微信' }

function remainingTime(createdAt, status) {
  if (status !== 'processing') return null
  const elapsed = (Date.now() - new Date(createdAt).getTime()) / 1000
  const left = 900 - elapsed // 15 min = 900s
  if (left <= 0) return <span style={{ color: '#ef4444', fontSize: 12 }}>已超时</span>
  const m = Math.floor(left / 60), s = Math.floor(left % 60)
  return <span style={{ color: '#f59e0b', fontSize: 12 }}>{m}:{String(s).padStart(2,'0')}</span>
}

export default function Orders() {
  const [searchParams, setSearchParams] = useSearchParams()
  const detailId = searchParams.get('id')

  const pageFromUrl = parseInt(searchParams.get('page')) || 1

  const [orders, setOrders] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [apps, setApps] = useState([])
  const [status, setStatus] = useState('')
  const [channel, setChannel] = useState('')
  const [appId, setAppId] = useState('')
  const [tradeNo, setTradeNo] = useState('')
  const sizeFromUrl = parseInt(searchParams.get('page_size')) || 10

  useEffect(() => { api.listApps().then(setApps) }, [])
  const [page, setPage] = useState(pageFromUrl)
  const [pageSize, setPageSize] = useState(sizeFromUrl)

  const load = useCallback(async () => {
    setLoading(true)
    const params = new URLSearchParams()
    if (status) params.set('status', status)
    if (channel) params.set('channel', channel)
    if (appId) params.set('app_id', appId)
    if (tradeNo) params.set('trade_no', tradeNo)
    params.set('page', page)
    params.set('page_size', pageSize)
    const d = await api.listOrders(params.toString())
    setOrders(d.orders || [])
    setTotal(d.total || 0)
    setLoading(false)
  }, [status, channel, appId, tradeNo, page, pageSize])

  const [, setTick] = useState(0)
  useEffect(() => {
    const timer = setInterval(() => setTick(t => t + 1), 30000)
    return () => clearInterval(timer)
  }, [])

  useEffect(() => {
    const sp = new URLSearchParams()
    if (status) sp.set('status', status)
    if (channel) sp.set('channel', channel)
    if (appId) sp.set('app_id', appId)
    if (tradeNo) sp.set('trade_no', tradeNo)
    if (page > 1) sp.set('page', page)
    if (pageSize !== 10) sp.set('page_size', pageSize)
    setSearchParams(sp, { replace: true })
    load()
  }, [status, channel, appId, tradeNo, page, pageSize])

  function handleSearch() {
    setPage(1)
    load()
  }

  function handleExport() {
    const params = new URLSearchParams()
    if (status) params.set('status', status)
    if (channel) params.set('channel', channel)
    if (appId) params.set('app_id', appId)
    downloadCSV(params.toString())
  }

  const columns = [
    {
      title: '订单 ID', dataIndex: 'TradeNo', width: 220,
      render: v => <code style={{ fontSize: 12 }}>{v}</code>,
    },
    {
      title: '应用', dataIndex: 'AppID', width: 120,
      render: v => (apps.find(a => a.ID === v) || {}).Name || v?.slice(0,8) || '—',
    },
    {
      title: '渠道', dataIndex: 'Channel', width: 80,
      render: v => chLabel[v] || v,
    },
    {
      title: '金额', dataIndex: 'Amount', width: 100,
      render: v => <span style={{ fontFamily: 'monospace' }}>¥{(v / 100).toFixed(2)}</span>,
    },
    {
      title: '状态', dataIndex: 'Status', width: 150,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}({v})</Tag>,
    },
    {
      title: '渠道交易号', dataIndex: 'ExternalID', width: 180,
      render: v => v ? <code style={{ fontSize: 12, color: '#64748b' }} title={v}>{v.slice(0, 20)}...</code> : <span style={{ color: '#94a3b8' }}>—</span>,
    },
    {
      title: '剩余', dataIndex: 'CreatedAt', width: 70,
      render: (v, record) => remainingTime(v, record.Status),
    },
    {
      title: '时间', dataIndex: 'CreatedAt', width: 170,
      render: v => new Date(v).toLocaleString('zh-CN'),
    },
    {
      title: '', dataIndex: 'ID', width: 60,
      render: id => <Button type="link" size="small" onClick={() => setSearchParams({ id })}>详情</Button>,
    },
  ]

  if (detailId) return <OrderDetail />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>支付订单</h2>
        <Button icon={<ExportOutlined />} onClick={handleExport}>导出 CSV</Button>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            placeholder="全部状态" style={{ width: 120 }} allowClear value={status || undefined}
            onChange={v => { setPage(1); setStatus(v || '') }}
            options={[
              { value: 'pending', label: '待支付' },
              { value: 'processing', label: '处理中' },
              { value: 'paid', label: '已支付' },
              { value: 'failed', label: '失败' },
            ]}
          />
          <Select
            placeholder="全部渠道" style={{ width: 120 }} allowClear value={channel || undefined}
            onChange={v => { setPage(1); setChannel(v || '') }}
            options={[
              { value: 'alipay', label: '支付宝' },
              { value: 'wechat', label: '微信' },
            ]}
          />
          <Select
            placeholder="全部应用" style={{ width: 160 }} allowClear value={appId || undefined}
            onChange={v => { setPage(1); setAppId(v || '') }}
            options={apps.map(a => ({ value: a.ID, label: a.Name }))}
          />
          <Input.Search
            placeholder="搜索订单 ID …"
            value={tradeNo}
            onChange={e => setTradeNo(e.target.value)}
            onSearch={handleSearch}
            style={{ width: 220 }}
            enterButton={<SearchOutlined />}
          />
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={orders}
        rowKey="ID"
        loading={loading}
        size="middle"
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: t => `共 ${t} 条`,
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50', '100'],
          onChange: (p, ps) => { setPage(p); setPageSize(ps) },
        }}
      />
    </div>
  )
}
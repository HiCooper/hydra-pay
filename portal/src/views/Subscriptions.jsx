import { useState, useEffect } from 'react'
import { Table, Tag } from 'antd'
import { api } from '../api/index.js'

export default function Subscriptions() {
  const [subs, setSubs] = useState([])
  const [loading, setLoading] = useState(true)

  async function load() { setLoading(true); const r = await api.subscriptions(); setSubs(r.subscriptions || []); setLoading(false) }
  useEffect(() => { load() }, [])

  const statusColor = { active: 'green', past_due: 'orange', cancelled: 'default', expired: 'default' }
  const statusLabel = { active: '进行中', past_due: '逾期', cancelled: '已取消', expired: '已过期' }

  const columns = [
    {
      title: '用户 ID', dataIndex: 'UserID', width: 140,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{v || '—'}</span>,
    },
    {
      title: '状态', dataIndex: 'Status', width: 80,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>,
    },
    {
      title: '周期开始', dataIndex: 'CurrentPeriodStart', width: 160,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{v ? new Date(v).toLocaleString() : '—'}</span>,
    },
    {
      title: '周期结束', dataIndex: 'CurrentPeriodEnd', width: 160,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{v ? new Date(v).toLocaleString() : '—'}</span>,
    },
    {
      title: '创建时间', dataIndex: 'CreatedAt', width: 160,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{new Date(v).toLocaleString()}</span>,
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
        <div>
          <h2 className="page-heading">订阅管理</h2>
          <p className="page-subtitle">所有订阅记录与状态</p>
        </div>
      </div>
      <div style={{
        background: '#fff',
        borderRadius: 8,
        border: '1px solid #e6e6e6',
        overflow: 'hidden',
      }}>
        <Table
          columns={columns}
          dataSource={subs}
          rowKey="ID"
          loading={loading}
          size="middle"
          locale={{ emptyText: <span style={{ color: '#999' }}>暂无订阅记录</span> }}
          style={{ margin: 0 }}
        />
      </div>
    </div>
  )
}

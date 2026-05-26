import { useState, useEffect } from 'react'
import { Table, Tag } from 'antd'
import { api } from '../api/index.js'

export default function Subscriptions() {
  const [subs, setSubs] = useState([])
  const [loading, setLoading] = useState(true)

  async function load() { setLoading(true); const r = await api.subscriptions(); setSubs(r.subscriptions || []); setLoading(false) }
  useEffect(() => { load() }, [])

  const statusColor = { active: 'blue', past_due: 'orange', cancelled: 'default', expired: 'default' }
  const statusLabel = { active: '进行中', past_due: '逾期', cancelled: '已取消', expired: '已过期' }

  const columns = [
    { title: '用户 ID', dataIndex: 'UserID', width: 140, render: v => v || '—' },
    {
      title: '状态', dataIndex: 'Status', width: 80,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>,
    },
    {
      title: '周期开始', dataIndex: 'CurrentPeriodStart', width: 160,
      render: v => v ? new Date(v).toLocaleString() : '—',
    },
    {
      title: '周期结束', dataIndex: 'CurrentPeriodEnd', width: 160,
      render: v => v ? new Date(v).toLocaleString() : '—',
    },
    {
      title: '创建时间', dataIndex: 'CreatedAt', width: 160,
      render: v => new Date(v).toLocaleString(),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>订阅管理</h2>
      </div>
      <Table columns={columns} dataSource={subs} rowKey="ID" loading={loading} size="middle"
        locale={{ emptyText: '暂无订阅记录' }}
      />
    </div>
  )
}

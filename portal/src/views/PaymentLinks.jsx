import { useState, useEffect } from 'react'
import { Table, Button, Tag, Modal, Form, Input, InputNumber, Select, Space, message } from 'antd'
import { PlusOutlined, LinkOutlined, StopOutlined, DeleteOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function PaymentLinks({ apps }) {
  const [links, setLinks] = useState([])
  const [loading, setLoading] = useState(true)
  const [openCreate, setOpenCreate] = useState(false)
  const [form] = Form.useForm()

  async function load() { setLoading(true); const r = await api.paymentLinks(); setLinks(r.payment_links); setLoading(false) }
  useEffect(() => { load() }, [])

  async function handleCreate(values) {
    await api.createPaymentLink({ ...values, amount: Math.round(values.amount_yuan * 100) })
    setOpenCreate(false)
    form.resetFields()
    message.success('支付链接已创建')
    load()
  }

  async function copy(url) {
    await navigator.clipboard.writeText(window.location.origin + url)
    message.success('链接已复制')
  }

  async function handleExpire(record) {
    Modal.confirm({
      title: '过期此链接？',
      content: `金额 ¥${(record.amount / 100).toFixed(2)}，过期后用户将无法继续支付。`,
      okText: '确认过期',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await api.expirePaymentLink(record.id)
        message.success('链接已过期')
        load()
      },
    })
  }

  async function handleDelete(record) {
    Modal.confirm({
      title: '删除此链接？',
      content: '删除后不可恢复。',
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        await api.deletePaymentLink(record.id)
        message.success('链接已删除')
        load()
      },
    })
  }

  const statusColor = { open: 'green', completed: 'blue', expired: 'default' }
  const statusLabel = { open: '进行中', completed: '已完成', expired: '已过期' }

  const columns = [
    {
      title: '应用', dataIndex: 'app_id', width: 120,
      render: v => {
        const app = apps?.find(a => a.id === v)
        return <span style={{ fontSize: 13, color: '#1a1a1a' }}>{app?.name || (v || '—').slice(0, 8)}</span>
      },
    },
    {
      title: '金额', dataIndex: 'amount', width: 100,
      render: v => (
        <span style={{ fontFamily: "'SF Mono', Menlo, Monaco, monospace", fontSize: 13, fontWeight: 500, color: '#1a1a1a' }}>
          ¥{(v / 100).toFixed(2)}
        </span>
      ),
    },
    {
      title: '描述', dataIndex: 'description', width: 180,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{v || '—'}</span>,
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>,
    },
    {
      title: '过期时间', dataIndex: 'expires_at', width: 160,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{new Date(v).toLocaleString()}</span>,
    },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: v => <span style={{ fontSize: 13, color: '#6b6b6b' }}>{new Date(v).toLocaleString()}</span>,
    },
    {
      title: '操作', width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link" size="small"
            icon={<LinkOutlined />}
            onClick={() => copy(record.checkout_url)}
            style={{ color: '#de481b', fontSize: 12, fontWeight: 500 }}
          >
            复制链接
          </Button>
          {record.status === 'open' && (
            <Button
              type="link" size="small" danger
              icon={<StopOutlined />}
              onClick={() => handleExpire(record)}
              style={{ fontSize: 12 }}
            >
              过期
            </Button>
          )}
          {record.status === 'expired' && (
            <Button
              type="link" size="small" danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
              style={{ fontSize: 12 }}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
        <div>
          <h2 className="page-heading">支付链接</h2>
          <p className="page-subtitle">创建和管理支付链接</p>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>
          创建链接
        </Button>
      </div>

      <div style={{
        background: '#fff',
        borderRadius: 8,
        border: '1px solid #e6e6e6',
        overflow: 'hidden',
      }}>
        <Table
          columns={columns}
          dataSource={links}
          rowKey="id"
          loading={loading}
          size="middle"
          locale={{ emptyText: <span style={{ color: '#999' }}>暂无支付链接，点击右上角创建</span> }}
          style={{ margin: 0 }}
        />
      </div>

      <Modal
        title={<span style={{ fontSize: 16, fontWeight: 600 }}>创建支付链接</span>}
        open={openCreate}
        onCancel={() => setOpenCreate(false)}
        onOk={() => form.submit()}
        destroyOnClose
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 8 }}>
          <Form.Item name="app_id" label="所属应用" rules={[{ required: true, message: '请选择应用' }]}>
            <Select
              placeholder="选择应用"
              options={(apps || []).map(a => ({ value: a.id, label: a.name }))}
            />
          </Form.Item>
          <Form.Item name="amount_yuan" label="金额（元）" rules={[{ required: true, message: '请输入金额' }]}>
            <InputNumber min={0.01} step={0.01} precision={2} placeholder="0.01" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="商品或服务说明" />
          </Form.Item>
          <Form.Item name="success_url" label="支付成功跳转">
            <Input placeholder="https://your-site.com/success" />
          </Form.Item>
          <Form.Item name="cancel_url" label="取消支付跳转">
            <Input placeholder="https://your-site.com/cancel" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Table, Button, Modal, Form, Input, Space, Tag, message } from 'antd'
import { PlusOutlined, CopyOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Apps({ apps: initialApps, onUpdate }) {
  const [apps, setApps] = useState(initialApps || [])
  const [loading, setLoading] = useState(false)
  const [openCreate, setOpenCreate] = useState(false)
  const [form] = Form.useForm()

  // Fetch apps list independently on mount
  useEffect(() => {
    setLoading(true)
    api.listApps()
      .then(data => { if (Array.isArray(data)) setApps(data) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  async function handleCreate(values) {
    const app = await api.createApp(values)
    // Reload from server to get fresh list
    const data = await api.listApps()
    if (Array.isArray(data)) setApps(data)
    if (onUpdate) {
      const me = await api.me()
      onUpdate(me)
    }
    setOpenCreate(false)
    form.resetFields()
    message.success('应用创建成功')
  }

  async function copy(text) {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  const columns = [
    {
      title: '应用名称', dataIndex: 'name', width: 160,
      render: v => <span style={{ fontSize: 13, fontWeight: 500, color: '#1a1a1a' }}>{v}</span>,
    },
    {
      title: 'API Key', dataIndex: 'api_key', width: 280,
      render: v => (
        <Space size={8}>
          <code style={{
            fontSize: 12, color: '#6b6b6b',
            fontFamily: "'SF Mono', Menlo, Monaco, monospace",
          }}>
            {v?.slice(0, 16)}...
          </code>
          <Button type="link" size="small"
            onClick={() => copy(v)}
            style={{ color: '#de481b', fontSize: 12 }}
          >
            复制
          </Button>
        </Space>
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: v => (
        <Tag color={v === 'active' ? 'green' : 'default'}>{v === 'active' ? '启用' : v}</Tag>
      ),
    },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: v => (
        <span style={{ fontSize: 13, color: '#6b6b6b' }}>
          {v ? new Date(v).toLocaleString('zh-CN') : '-'}
        </span>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
        <div>
          <h2 className="page-heading">我的应用</h2>
          <p className="page-subtitle">管理 API 密钥和应用配置</p>
        </div>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>
          创建应用
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
          dataSource={apps}
          rowKey="id"
          loading={loading}
          size="middle"
          locale={{ emptyText: <span style={{ color: '#999' }}>暂无应用，点击右上角创建</span> }}
          pagination={false}
          style={{ margin: 0 }}
        />
      </div>

      <Modal
        title={<span style={{ fontSize: 16, fontWeight: 600 }}>创建应用</span>}
        open={openCreate}
        onCancel={() => setOpenCreate(false)}
        onOk={() => form.submit()}
        destroyOnClose
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" onFinish={handleCreate} style={{ marginTop: 8 }}>
          <Form.Item name="name" label="应用名称" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="例如：我的网站、iOS App" />
          </Form.Item>
          <Form.Item name="webhook_url" label="Webhook 回调地址">
            <Input placeholder="https://your-server.com/webhook" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

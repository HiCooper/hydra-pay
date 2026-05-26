import { useState } from 'react'
import { Table, Button, Modal, Form, Input, Space, Tag, message } from 'antd'
import { PlusOutlined, CopyOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Apps({ apps: initialApps, onUpdate }) {
  const [apps, setApps] = useState(initialApps || [])
  const [openCreate, setOpenCreate] = useState(false)
  const [form] = Form.useForm()

  async function handleCreate(values) {
    const app = await api.createApp(values)
    const newApps = [app, ...apps]
    setApps(newApps)
    if (onUpdate) {
      const data = await api.me()
      onUpdate(data)
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
    { title: '应用名称', dataIndex: 'name', width: 160 },
    {
      title: 'API Key', dataIndex: 'api_key', width: 240,
      render: v => (
        <Space>
          <code style={{ fontSize: 12 }}>{v?.slice(0, 12)}...</code>
          <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copy(v)} />
        </Space>
      ),
    },
    {
      title: 'API Key 完整', width: 80,
      render: (_, record) => (
        <Button type="link" size="small" onClick={() => copy(record.api_key)}>复制</Button>
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: v => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag>,
    },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: v => v ? new Date(v).toLocaleString('zh-CN') : '-',
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>我的应用</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>创建应用</Button>
      </div>

      <Table columns={columns} dataSource={apps} rowKey="id" size="middle"
        locale={{ emptyText: '暂无应用，点击右上角创建' }}
        pagination={false}
      />

      <Modal title="创建应用" open={openCreate} onCancel={() => setOpenCreate(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
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

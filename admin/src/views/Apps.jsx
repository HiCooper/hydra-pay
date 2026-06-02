import { useState, useEffect } from 'react'
import { Table, Button, Tag, Modal, Form, Input, Select, Space, message } from 'antd'
import { PlusOutlined, CopyOutlined, EditOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Apps() {
  const [apps, setApps] = useState([])
  const [merchants, setMerchants] = useState([])
  const [loading, setLoading] = useState(true)
  const [openCreate, setOpenCreate] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()

  async function load() {
    setLoading(true)
    const [a, m] = await Promise.all([api.listApps(), api.listMerchants()])
    setApps(a)
    setMerchants(m)
    setLoading(false)
  }
  useEffect(() => { load() }, [])

  async function handleCreate(values) {
    await api.createApp(values)
    setOpenCreate(false)
    form.resetFields()
    message.success('应用创建成功')
    load()
  }

  async function handleEdit(values) {
    const payload = {}
    if (values.name !== undefined) payload.name = values.name
    if (values.status !== undefined) payload.status = values.status
    if (values.webhook_url !== undefined) payload.webhook_url = values.webhook_url
    await api.updateApp(editing.id, payload)
    setEditing(null)
    message.success('已保存')
    load()
  }

  async function copy(text) {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  const merchantMap = {}
  merchants.forEach(m => { merchantMap[m.id] = m.name })

  const columns = [
    { title: '名称', dataIndex: 'name', width: 160 },
    { title: '归属商户', dataIndex: 'merchant_id', width: 160, render: v => merchantMap[v] || v || '—' },
    {
      title: 'API Key', dataIndex: 'api_key', width: 220,
      render: v => (
        <Space>
          <code style={{ fontSize: 12 }}>{v?.slice(0, 16)}...</code>
          <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copy(v)} />
        </Space>
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: v => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag>,
    },
    {
      title: '操作', width: 80,
      render: (_, record) => (
        <Button type="link" size="small" icon={<EditOutlined />} onClick={() => { setEditing(record); editForm.setFieldsValue(record) }}>编辑</Button>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>应用管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>创建应用</Button>
      </div>

      <Table columns={columns} dataSource={apps} rowKey="id" loading={loading} size="middle"
        locale={{ emptyText: '暂无应用，请先创建商户' }}
      />

      <Modal title="创建应用" open={openCreate} onCancel={() => setOpenCreate(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="merchant_id" label="归属商户" rules={[{ required: true, message: '请选择商户' }]}>
            <Select
              placeholder="选择商户"
              options={merchants.filter(m => m.status === 'active').map(m => ({ value: m.id, label: m.name }))}
              showSearch
              filterOption={(input, option) => option.label.toLowerCase().includes(input.toLowerCase())}
            />
          </Form.Item>
          <Form.Item name="name" label="应用名称" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="接入方名称" />
          </Form.Item>
          <Form.Item name="webhook_url" label="Webhook 回调地址"><Input placeholder="https://..." /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`编辑 · ${editing?.name || ''}`} open={!!editing} onCancel={() => setEditing(null)} onOk={() => editForm.submit()} destroyOnClose>
        <Form form={editForm} layout="vertical" onFinish={handleEdit} initialValues={editing || {}}>
          <Form.Item name="name" label="名称"><Input /></Form.Item>
          <Form.Item name="webhook_url" label="Webhook 回调地址"><Input /></Form.Item>
          <Form.Item name="status" label="状态">
            <Select options={[{ value: 'active', label: 'active' }, { value: 'disabled', label: 'disabled' }]} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

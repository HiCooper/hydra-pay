import { useState, useEffect } from 'react'
import { Table, Button, Tag, Modal, Form, Input, Select, Space, message } from 'antd'
import { PlusOutlined, CopyOutlined, EditOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Apps() {
  const [apps, setApps] = useState([])
  const [loading, setLoading] = useState(true)
  const [openCreate, setOpenCreate] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()

  async function load() { setLoading(true); setApps(await api.listApps()); setLoading(false) }
  useEffect(() => { load() }, [])

  async function handleCreate(values) {
    await api.createApp(values)
    setOpenCreate(false)
    form.resetFields()
    message.success('应用创建成功')
    load()
  }

  async function handleEdit(values) {
    await api.updateApp(editing.ID, values)
    setEditing(null)
    message.success('已保存')
    load()
  }

  async function copy(text) {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  const columns = [
    { title: '名称', dataIndex: 'Name', width: 160 },
    {
      title: 'API Key', dataIndex: 'APIKey', width: 220,
      render: v => (
        <Space>
          <code style={{ fontSize: 12 }}>{v?.slice(0, 16)}...</code>
          <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copy(v)} />
        </Space>
      ),
    },
    { title: '支付宝 PID', dataIndex: 'AlipayPID', width: 120, render: v => v || '—' },
    { title: '微信子商户号', dataIndex: 'WechatSubMchid', width: 130, render: v => v || '—' },
    {
      title: '状态', dataIndex: 'Status', width: 80,
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

      <Table columns={columns} dataSource={apps} rowKey="ID" loading={loading} size="middle"
        locale={{ emptyText: '暂无应用，点击右上角创建' }}
      />

      <Modal title="创建应用" open={openCreate} onCancel={() => setOpenCreate(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="应用名称" rules={[{ required: true, message: '请输入应用名称' }]}>
            <Input placeholder="接入方名称" />
          </Form.Item>
          <Form.Item name="alipay_pid" label="支付宝子商户 PID"><Input placeholder="2088..." /></Form.Item>
          <Form.Item name="wechat_sub_mchid" label="微信子商户号"><Input placeholder="子商户 mchid" /></Form.Item>
          <Form.Item name="wechat_sub_appid" label="微信子商户 AppID"><Input placeholder="wx..." /></Form.Item>
          <Form.Item name="webhook_url" label="Webhook 回调地址"><Input placeholder="https://..." /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`编辑 · ${editing?.Name || ''}`} open={!!editing} onCancel={() => setEditing(null)} onOk={() => editForm.submit()} destroyOnClose>
        <Form form={editForm} layout="vertical" onFinish={handleEdit} initialValues={editing || {}}>
          <Form.Item name="Name" label="名称"><Input /></Form.Item>
          <Form.Item name="AlipayPID" label="支付宝 PID"><Input /></Form.Item>
          <Form.Item name="WechatSubMchid" label="微信子商户号"><Input /></Form.Item>
          <Form.Item name="WechatSubAppid" label="微信子商户 AppID"><Input /></Form.Item>
          <Form.Item name="WebhookURL" label="Webhook 回调地址"><Input /></Form.Item>
          <Form.Item name="Status" label="状态">
            <Select options={[{ value: 'active', label: 'active' }, { value: 'disabled', label: 'disabled' }]} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
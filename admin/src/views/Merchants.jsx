import { useState, useEffect } from 'react'
import { Table, Button, Tag, Modal, Form, Input, Select, Space, message, Descriptions } from 'antd'
import { PlusOutlined, EditOutlined, IdcardOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

const statusColor = { active: 'green', disabled: 'red' }
const statusLabel = { active: '正常', disabled: '已禁用' }
const obStatusMap = {
  pending: { color: 'default', label: '待提交' },
  submitted: { color: 'processing', label: '已提交' },
  auditing: { color: 'orange', label: '审核中' },
  approved: { color: 'green', label: '已通过' },
  rejected: { color: 'red', label: '已拒绝' },
}

export default function Merchants() {
  const [merchants, setMerchants] = useState([])
  const [loading, setLoading] = useState(true)
  const [openCreate, setOpenCreate] = useState(false)
  const [editing, setEditing] = useState(null)
  const [onboarding, setOnboarding] = useState(null)
  const [onbOpen, setOnbOpen] = useState(false)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()

  async function load() { setLoading(true); setMerchants(await api.listMerchants()); setLoading(false) }
  useEffect(() => { load() }, [])

  async function handleCreate(values) {
    await api.createMerchant(values)
    setOpenCreate(false)
    form.resetFields()
    message.success('商户创建成功')
    load()
  }

  async function handleEdit(values) {
    await api.updateMerchant(editing.ID, values)
    setEditing(null)
    message.success('已保存')
    load()
  }

  async function viewOnboarding(record) {
    setOnbOpen(true)
    try {
      const ob = await api.getMerchantOnboarding(record.ID)
      setOnboarding(ob)
    } catch {
      setOnboarding(null)
    }
  }

  const columns = [
    { title: '商户名称', dataIndex: 'Name', width: 180 },
    { title: '邮箱', dataIndex: 'Email', width: 200 },
    { title: '联系人', dataIndex: 'ContactName', width: 100, render: v => v || '—' },
    { title: '联系电话', dataIndex: 'ContactPhone', width: 130, render: v => v || '—' },
    {
      title: '支付宝 PID', dataIndex: 'AlipayPID', width: 130,
      render: v => v ? <code style={{ fontSize: 12 }}>{v}</code> : <span style={{ color: '#94a3b8' }}>—</span>,
    },
    {
      title: '微信子商户', dataIndex: 'WechatSubMchid', width: 130,
      render: v => v ? <code style={{ fontSize: 12 }}>{v}</code> : <span style={{ color: '#94a3b8' }}>—</span>,
    },
    {
      title: '状态', dataIndex: 'Status', width: 80,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>,
    },
    {
      title: '操作', width: 120,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => { setEditing(record); editForm.setFieldsValue(record) }}>编辑</Button>
          <Button type="link" size="small" icon={<IdcardOutlined />} onClick={() => viewOnboarding(record)}>进件</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>商户管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>创建商户</Button>
      </div>

      <Table columns={columns} dataSource={merchants} rowKey="ID" loading={loading} size="middle"
        locale={{ emptyText: '暂无商户' }}
      />

      {/* Create Modal */}
      <Modal title="创建商户" open={openCreate} onCancel={() => setOpenCreate(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="name" label="商户名称" rules={[{ required: true }]}>
            <Input placeholder="企业名称" />
          </Form.Item>
          <Form.Item name="email" label="登录邮箱" rules={[{ required: true, type: 'email' }]}>
            <Input placeholder="merchant@example.com" />
          </Form.Item>
          <Form.Item name="password" label="登录密码" rules={[{ required: true, min: 6 }]}>
            <Input.Password placeholder="至少 6 位" />
          </Form.Item>
          <Form.Item name="contact_name" label="联系人"><Input placeholder="法人或经办人" /></Form.Item>
          <Form.Item name="contact_phone" label="联系电话"><Input placeholder="手机号" /></Form.Item>
        </Form>
      </Modal>

      {/* Edit Modal */}
      <Modal title={`编辑 · ${editing?.Name || ''}`} open={!!editing} onCancel={() => setEditing(null)} onOk={() => editForm.submit()} destroyOnClose>
        <Form form={editForm} layout="vertical" onFinish={handleEdit} initialValues={editing || {}}>
          <Form.Item name="Name" label="商户名称"><Input /></Form.Item>
          <Form.Item name="Email" label="邮箱"><Input /></Form.Item>
          <Form.Item name="ContactName" label="联系人"><Input /></Form.Item>
          <Form.Item name="ContactPhone" label="联系电话"><Input /></Form.Item>
          <Form.Item name="password" label="新密码（留空不修改）"><Input.Password placeholder="留空则不修改" /></Form.Item>
          <Form.Item name="Status" label="状态">
            <Select options={[{ value: 'active', label: '正常' }, { value: 'disabled', label: '禁用' }]} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Onboarding Status Modal */}
      <Modal title="进件状态" open={onbOpen} onCancel={() => { setOnbOpen(false); setOnboarding(null) }} footer={null} destroyOnClose>
        {onboarding ? (
          <Descriptions column={1} size="small">
            <Descriptions.Item label="渠道">{onboarding.channel === 'alipay' ? '支付宝' : '微信支付'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={obStatusMap[onboarding.status]?.color}>{obStatusMap[onboarding.status]?.label || onboarding.status}</Tag>
            </Descriptions.Item>
            {onboarding.applyment_id && <Descriptions.Item label="申请单号">{onboarding.applyment_id}</Descriptions.Item>}
            {onboarding.sub_merchant_id && <Descriptions.Item label="子商户号">{onboarding.sub_merchant_id}</Descriptions.Item>}
            {onboarding.sign_url && (
              <Descriptions.Item label="签约链接">
                <a href={onboarding.sign_url} target="_blank" rel="noopener noreferrer">前往签约</a>
              </Descriptions.Item>
            )}
            {onboarding.error_message && <Descriptions.Item label="错误信息"><span style={{ color: '#ef4444' }}>{onboarding.error_message}</span></Descriptions.Item>}
          </Descriptions>
        ) : (
          <div style={{ color: '#94a3b8', textAlign: 'center', padding: 24 }}>暂无进件记录</div>
        )}
      </Modal>
    </div>
  )
}

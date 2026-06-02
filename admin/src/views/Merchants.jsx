import { useState, useEffect, useCallback } from 'react'
import { Table, Button, Tag, Modal, Form, Input, Select, Space, message, Descriptions, Tabs, Switch } from 'antd'
import { PlusOutlined, EditOutlined, IdcardOutlined, InfoCircleOutlined, CopyOutlined, SearchOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

const statusColor = { active: 'green', disabled: 'red' }
const statusLabel = { active: '正常', disabled: '已禁用' }
const chLabel = { alipay: '支付宝', wechat: '微信', unionpay: '云闪付', ecny: '数字人民币' }
const obStatusMap = {
  pending: { color: 'default', label: '待提交' },
  submitted: { color: 'processing', label: '已提交' },
  auditing: { color: 'orange', label: '审核中' },
  approved: { color: 'green', label: '已通过' },
  rejected: { color: 'red', label: '已拒绝' },
}

export default function Merchants() {
  const [merchants, setMerchants] = useState([])
  const [apps, setApps] = useState([])
  const [channels, setChannels] = useState([])
  const [appChannels, setAppChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [openCreate, setOpenCreate] = useState(false)
  const [editing, setEditing] = useState(false)
  const [onboarding, setOnboarding] = useState(null)
  const [detail, setDetail] = useState(null)
  const [openChannelApp, setOpenChannelApp] = useState(null)
  const [appSearch, setAppSearch] = useState('')
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  const [channelForm] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    const params = new URLSearchParams()
    if (search) params.set('q', search)
    const [m, a, ch] = await Promise.all([
      api.listMerchants(params.toString()),
      api.listApps(),
      api.listChannels().then(d => d.channels || []).catch(() => []),
    ])
    setMerchants(m)
    setApps(a)
    setChannels(ch)
    setLoading(false)
  }, [search])

  useEffect(() => { load() }, [load])

  async function handleCreate(values) {
    await api.createMerchant(values)
    setOpenCreate(false)
    form.resetFields()
    message.success('商户创建成功')
    load()
  }

  async function handleEdit(values) {
    const payload = {}
    if (values.name !== undefined) payload.name = values.name
    if (values.email !== undefined) payload.email = values.email
    if (values.contact_name !== undefined) payload.contact_name = values.contact_name
    if (values.contact_phone !== undefined) payload.contact_phone = values.contact_phone
    if (values.status !== undefined) payload.status = values.status
    if (values.password) payload.password = values.password
    await api.updateMerchant(detail.id, payload)
    setEditing(false)
    message.success('已保存')
    load()
    const m = await api.getMerchant(detail.id)
    setDetail(m)
  }

  async function refreshOnboarding() {
    try {
      const ob = await api.getMerchantOnboarding(detail.id)
      setOnboarding(ob)
    } catch {
      setOnboarding(null)
    }
  }

  async function openDetail(record) {
    setDetail(record)
    setEditing(false)
    setOnboarding(null)
    setAppSearch('')
    // Load app channels for this merchant
    try {
      const ac = await api.listMerchantAppChannels(record.id)
      setAppChannels(ac.channels || [])
    } catch {
      setAppChannels([])
    }
  }

  async function toggleAppChannel(record, checked) {
    try {
      await api.updateAppChannel(record.id, { enabled: checked })
      message.success(`${chLabel[record.channel_key] || record.channel_key} 已${checked ? '启用' : '停用'}`)
      const ac = await api.listMerchantAppChannels(detail.id)
      setAppChannels(ac.channels || [])
    } catch (err) {
      message.error(err.message)
    }
  }

  async function handleOpenChannel(app) {
    setOpenChannelApp(app)
    const existing = appChannels.filter(ac => ac.app_id === app.id).map(ac => ac.channel_key)
    // Only show channels the merchant has onboarded (has sub_merchant_id)
    // or channels that don't require onboarding
    const mchSubIds = {
      alipay: detail.alipay_pid,
      wechat: detail.wechat_sub_mchid,
      unionpay: detail.unionpay_sub_mer_id,
      ecny: detail.ecny_sub_mer_id,
    }
    const available = channels.filter(c =>
      c.enabled &&
      !existing.includes(c.key) &&
      (!c.supports_onboarding || mchSubIds[c.key])
    )
    channelForm.setFieldsValue({ app_id: app.id, channel_key: available[0]?.key || '' })
  }

  async function handleCreateChannel(values) {
    try {
      await api.createAppChannel(values)
      setOpenChannelApp(null)
      channelForm.resetFields()
      message.success('渠道已开通')
      const ac = await api.listMerchantAppChannels(detail.id)
      setAppChannels(ac.channels || [])
    } catch (err) {
      message.error(err.message)
    }
  }

  async function copy(text) {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  function handleSearch() {
    setSearch(searchInput)
  }

  const merchantApps = detail
    ? apps.filter(a => a.merchant_id === detail.id && (!appSearch || a.name.toLowerCase().includes(appSearch.toLowerCase())))
    : []

  const columns = [
    { title: '商户名称', dataIndex: 'name', width: 180 },
    { title: '邮箱', dataIndex: 'email', width: 200 },
    { title: '联系人', dataIndex: 'contact_name', width: 100, render: v => v || '—' },
    { title: '联系电话', dataIndex: 'contact_phone', width: 130, render: v => v || '—' },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: v => <Tag color={statusColor[v] || 'default'}>{statusLabel[v] || v}</Tag>,
    },
    {
      title: '操作', width: 80,
      render: (_, record) => (
        <Button type="link" size="small" icon={<InfoCircleOutlined />} onClick={() => openDetail(record)}>查看</Button>
      ),
    },
  ]

  const appColumns = [
    { title: '应用名称', dataIndex: 'name', width: 140 },
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
      title: 'Webhook', dataIndex: 'webhook_url', width: 160,
      render: v => v ? <code style={{ fontSize: 11 }} title={v}>{v.slice(0, 24)}...</code> : <span style={{ color: '#94a3b8' }}>—</span>,
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>商户管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpenCreate(true)}>创建商户</Button>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="搜索商户名称、邮箱、联系电话 …"
          value={searchInput}
          onChange={e => setSearchInput(e.target.value)}
          onSearch={handleSearch}
          style={{ width: 360 }}
          enterButton={<SearchOutlined />}
          allowClear
        />
      </div>

      <Table columns={columns} dataSource={merchants} rowKey="id" loading={loading} size="middle"
        locale={{ emptyText: '暂无商户' }}
      />

      {/* Create Merchant Modal */}
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

      {/* Merchant Detail Modal */}
      <Modal
        title={`商户详情 · ${detail?.name || ''}`}
        open={!!detail}
        onCancel={() => { setDetail(null); setEditing(false); setOpenChannelApp(null) }}
        width={900}
        footer={null}
        destroyOnClose
      >
        {detail && (
          <Tabs defaultActiveKey="info" items={[
            {
              key: 'info',
              label: '基本信息',
              children: editing ? (
                <Form form={editForm} layout="vertical" onFinish={handleEdit} initialValues={detail}>
                  <Form.Item name="name" label="商户名称" rules={[{ required: true }]}><Input /></Form.Item>
                  <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email' }]}><Input /></Form.Item>
                  <Form.Item name="contact_name" label="联系人"><Input /></Form.Item>
                  <Form.Item name="contact_phone" label="联系电话"><Input /></Form.Item>
                  <Form.Item name="password" label="新密码（留空不修改）"><Input.Password placeholder="留空则不修改" /></Form.Item>
                  <Form.Item name="status" label="状态">
                    <Select options={[{ value: 'active', label: '正常' }, { value: 'disabled', label: '禁用' }]} />
                  </Form.Item>
                  <Space>
                    <Button type="primary" htmlType="submit">保存</Button>
                    <Button onClick={() => setEditing(false)}>取消</Button>
                  </Space>
                </Form>
              ) : (
                <div>
                  <div style={{ marginBottom: 12, textAlign: 'right' }}>
                    <Button icon={<EditOutlined />} onClick={() => { editForm.setFieldsValue(detail); setEditing(true) }}>编辑商户</Button>
                  </div>
                  <Descriptions column={2} size="small" bordered>
                    <Descriptions.Item label="商户名称">{detail.name}</Descriptions.Item>
                    <Descriptions.Item label="邮箱">{detail.email}</Descriptions.Item>
                    <Descriptions.Item label="联系人">{detail.contact_name || '—'}</Descriptions.Item>
                    <Descriptions.Item label="联系电话">{detail.contact_phone || '—'}</Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <Tag color={statusColor[detail.status] || 'default'}>{statusLabel[detail.status] || detail.status}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间">{new Date(detail.created_at).toLocaleString('zh-CN')}</Descriptions.Item>
                  </Descriptions>
                </div>
              ),
            },
            {
              key: 'apps',
              label: `应用列表 (${merchantApps.length})`,
              children: (
                <div>
                  <div style={{ marginBottom: 12 }}>
                    <Input.Search
                      placeholder="搜索应用名称 …"
                      value={appSearch}
                      onChange={e => setAppSearch(e.target.value)}
                      style={{ width: 220 }}
                      size="small"
                      allowClear
                    />
                  </div>
                  <Table columns={appColumns} dataSource={merchantApps} rowKey="id" size="small"
                    pagination={false}
                    locale={{ emptyText: '暂无应用' }}
                    expandable={{
                      expandedRowRender: (app) => {
                        const appChs = appChannels.filter(ac => ac.app_id === app.id)
                        const availableChannels = channels.filter(
                          c => c.enabled && !appChs.some(ac => ac.channel_key === c.key)
                        )
                        return (
                          <div style={{ padding: '8px 0' }}>
                            <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <span style={{ fontSize: 12, color: '#6b6b6b' }}>{app.name} 已开通的支付渠道</span>
                              {availableChannels.length > 0 && (
                                <Button size="small" type="dashed" icon={<PlusOutlined />}
                                  onClick={() => handleOpenChannel(app)}>
                                  开通渠道
                                </Button>
                              )}
                            </div>
                            {appChs.length === 0 ? (
                              <div style={{ color: '#94a3b8', fontSize: 12, padding: '8px 0' }}>暂未开通任何渠道</div>
                            ) : (
                              <Table
                                columns={[
                                  { title: '渠道', dataIndex: 'channel_key', width: 100, render: v => chLabel[v] || v },
                                  { title: '子商户号', width: 160, render: (_, ac) => {
                                    const subIds = { alipay: detail.alipay_pid, wechat: detail.wechat_sub_mchid, unionpay: detail.unionpay_sub_mer_id, ecny: detail.ecny_sub_mer_id }
                                    const sid = subIds[ac.channel_key]
                                    return sid ? <code style={{ fontSize: 12 }}>{sid}</code> : <span style={{ color: '#94a3b8' }}>—</span>
                                  }},
                                  { title: '状态', dataIndex: 'enabled', width: 70,
                                    render: v => <Tag color={v ? 'green' : 'default'}>{v ? '已启用' : '已停用'}</Tag>,
                                  },
                                  { title: '启用', dataIndex: 'enabled', width: 70,
                                    render: (v, record) => <Switch checked={v} size="small" onChange={(c) => toggleAppChannel(record, c)} />,
                                  },
                                ]}
                                dataSource={appChs}
                                rowKey="id"
                                size="small"
                                pagination={false}
                              />
                            )}
                          </div>
                        )
                      },
                      rowExpandable: () => true,
                      defaultExpandAllRows: false,
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'onboarding',
              label: '进件状态',
              children: (() => {
                const subIdMap = {
                  alipay: detail.alipay_pid,
                  wechat: detail.wechat_sub_mchid,
                  wechat_appid: detail.wechat_sub_appid,
                  unionpay: detail.unionpay_sub_mer_id,
                  ecny: detail.ecny_sub_mer_id,
                }
                const onboardedChannels = channels.filter(c => subIdMap[c.key])
                return (
                <div>
                  <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8 }}>已进件渠道</h4>
                  {onboardedChannels.length === 0 ? (
                    <div style={{ color: '#94a3b8', fontSize: 13, marginBottom: 16 }}>暂无已进件的渠道，请先通过商户端发起进件申请</div>
                  ) : (
                    <Table
                      columns={[
                        { title: '渠道', dataIndex: 'label', width: 100 },
                        { title: '子商户号', width: 180, render: (_, c) => <code style={{ fontSize: 12 }}>{subIdMap[c.key] || '—'}</code> },
                        { title: '支持进件', dataIndex: 'supports_onboarding', width: 80, render: v => v ? <Tag color="green">是</Tag> : <Tag>否</Tag> },
                      ]}
                      dataSource={onboardedChannels}
                      rowKey="id"
                      size="small"
                      pagination={false}
                      style={{ marginBottom: 20 }}
                    />
                  )}

                  <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 8 }}>最近进件申请</h4>
                  <div style={{ marginBottom: 12, textAlign: 'right' }}>
                    <Button icon={<IdcardOutlined />} onClick={refreshOnboarding}>刷新</Button>
                  </div>
                  {onboarding ? (
                    <Descriptions column={1} size="small" bordered>
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
                    <div style={{ color: '#94a3b8', textAlign: 'center', padding: 24 }}>点击刷新按钮加载进件申请状态</div>
                  )}
                </div>
                )
              })(),
            },
          ]} />
        )}
      </Modal>

      {/* Open Channel Modal */}
      <Modal title="开通支付渠道" open={!!openChannelApp}
        onCancel={() => setOpenChannelApp(null)} onOk={() => channelForm.submit()} destroyOnClose>
        <Form form={channelForm} layout="vertical" onFinish={handleCreateChannel}>
          <Form.Item name="app_id" hidden><Input /></Form.Item>
          <Form.Item label="应用">
            <Input value={openChannelApp?.name || ''} disabled />
          </Form.Item>
          <Form.Item name="channel_key" label="支付渠道" rules={[{ required: true, message: '请选择渠道' }]}>
            <Select
              placeholder="选择渠道（仅显示已进件的渠道）"
              options={channels
                .filter(c => {
                  if (!c.enabled) return false
                  if (appChannels.some(ac => ac.app_id === openChannelApp?.id && ac.channel_key === c.key)) return false
                  if (!c.supports_onboarding) return true
                  const mchSubIds = { alipay: detail?.alipay_pid, wechat: detail?.wechat_sub_mchid, unionpay: detail?.unionpay_sub_mer_id, ecny: detail?.ecny_sub_mer_id }
                  return !!mchSubIds[c.key]
                })
                .map(c => ({ value: c.key, label: c.label }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

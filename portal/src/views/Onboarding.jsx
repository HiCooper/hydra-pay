import { useState, useEffect, useCallback } from 'react'
import { Card, Form, Input, Select, Button, Tag, Alert, QRCode, Space, message } from 'antd'
import { LinkOutlined, ReloadOutlined, CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons'
import { api } from '../api/index.js'

const statusMap = {
  pending: { color: 'default', label: '待提交' },
  submitted: { color: 'processing', label: '已提交' },
  auditing: { color: 'orange', label: '审核中' },
  approved: { color: 'green', label: '已通过' },
  rejected: { color: 'red', label: '已拒绝' },
}

export default function Onboarding({ merchant }) {
  const [record, setRecord] = useState(null)
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    try {
      const [r, ch] = await Promise.all([
        api.getOnboardingStatus().catch(() => null),
        api.listChannels().catch(() => ({ channels: [] })),
      ])
      setRecord(r)
      setChannels(ch?.channels || [])
    } catch (e) {
      setRecord(null)
    }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  async function handleSubmit(values) {
    setSubmitting(true)
    const selected = values.channels
    const errors = []
    for (let i = 0; i < selected.length; i++) {
      try {
        const r = await api.initiateOnboarding({ ...values, channel: selected[i] })
        if (i === selected.length - 1) setRecord(r)
      } catch (err) {
        const label = channels.find(c => c.key === selected[i])?.label || selected[i]
        errors.push(`${label}: ${err.message}`)
      }
    }
    if (errors.length === selected.length) {
      message.error(errors.join('；'))
    } else if (errors.length > 0) {
      message.warning(`部分提交失败：${errors.join('；')}`)
    } else {
      message.success(`已提交 ${selected.length} 个渠道申请`)
    }
    setSubmitting(false)
  }

  if (loading) return (
    <div style={{ color: '#6b6b6b', textAlign: 'center', padding: 64, fontSize: 14 }}>加载中...</div>
  )

  const hasPID = channels.some(ch => ch.configured)

  return (
    <div>
      <h2 className="page-heading">支付渠道管理</h2>
      <p className="page-subtitle">提交商户资料，开通支付渠道</p>

      {/* Approved banner */}
      {hasPID && (
        <Alert
          type="success"
          showIcon
          icon={<CheckCircleFilled style={{ color: '#04d66f' }} />}
          message={<span style={{ fontWeight: 600, color: '#1a1a1a' }}>商户已认证</span>}
          description={
            <div style={{ fontSize: 13, color: '#6b6b6b' }}>
              {merchant.alipay_pid && <div style={{ marginBottom: 2 }}>支付宝 PID：<code style={{ color: '#1a1a1a' }}>{merchant.alipay_pid}</code></div>}
              {merchant.wechat_sub_mchid && <div>微信子商户号：<code style={{ color: '#1a1a1a' }}>{merchant.wechat_sub_mchid}</code></div>}
            </div>
          }
          style={{
            marginBottom: 20, borderRadius: 8,
            background: '#f0fdf6', border: '1px solid #d4f5e4',
          }}
        />
      )}

      {record && record.status !== 'rejected' ? (
        <Card
          style={{ borderRadius: 8 }}
          styles={{
            header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' },
            body: { padding: '20px 24px' },
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <Space size={12}>
              <span style={{ fontWeight: 500, fontSize: 14, color: '#1a1a1a' }}>申请状态</span>
              <Tag color={statusMap[record.status]?.color}>{statusMap[record.status]?.label}</Tag>
              <span style={{
                color: '#6b6b6b', fontSize: 12,
                background: '#f7f7f7', padding: '2px 8px', borderRadius: 4,
              }}>
                {channels.find(c => c.key === record.channel)?.label || record.channel}
              </span>
            </Space>
            {record.status !== 'approved' && (
              <Button icon={<ReloadOutlined />} onClick={load} size="small">刷新</Button>
            )}
          </div>

          {record.sign_url && (
            <div style={{
              background: '#fafafa', borderRadius: 8, padding: '16px 20px',
              marginBottom: 12,
            }}>
              <div style={{ fontWeight: 500, fontSize: 13, color: '#1a1a1a', marginBottom: 8 }}>签约链接</div>
              <a
                href={record.sign_url}
                target="_blank"
                rel="noopener noreferrer"
                style={{ color: '#de481b', fontSize: 13, fontWeight: 500 }}
              >
                <LinkOutlined style={{ marginRight: 4 }} />
                前往 {channels.find(c => c.key === record.channel)?.label || record.channel} 完成签约
              </a>
            </div>
          )}

          {record.qr_code_url && (
            <div style={{ textAlign: 'center', marginTop: 20 }}>
              <div style={{ fontWeight: 500, fontSize: 13, color: '#1a1a1a', marginBottom: 12 }}>扫码签约</div>
              <QRCode value={record.qr_code_url} size={180}
                style={{ padding: 12, background: '#fff', borderRadius: 8, border: '1px solid #f0f0f0' }}
              />
            </div>
          )}

          {record.error_message && (
            <Alert
              type="error"
              message={record.error_message}
              style={{ marginTop: 16, borderRadius: 6 }}
              icon={<CloseCircleFilled style={{ color: '#df1b41' }} />}
            />
          )}
        </Card>
      ) : (
        <Card
          title={<span style={{ fontSize: 15, fontWeight: 600, color: '#1a1a1a' }}>{record ? '重新申请' : '申请支付渠道'}</span>}
          style={{ borderRadius: 8 }}
          styles={{
            header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' },
            body: { padding: '20px 24px' },
          }}
        >
          {record?.status === 'rejected' && (
            <Alert
              type="error"
              icon={<CloseCircleFilled style={{ color: '#df1b41' }} />}
              message={`上次申请被拒绝：${record.error_message || '未知原因'}`}
              style={{ marginBottom: 20, borderRadius: 6 }}
            />
          )}
          <Form form={form} layout="vertical" onFinish={handleSubmit}>
            <Form.Item name="channels" label="渠道" rules={[{ required: true, message: '请选择至少一个渠道', type: 'array' }]}>
              <Select mode="multiple" options={
                channels.map(ch => ({ value: ch.key, label: ch.label, disabled: ch.configured }))
              } placeholder="选择支付渠道（可多选）" />
            </Form.Item>
            <Form.Item name="merchant_name" label="商户名称" rules={[{ required: true, message: '请输入商户名称' }]}>
              <Input placeholder="营业执照上的企业名称" />
            </Form.Item>
            <Form.Item name="contact_name" label="联系人" rules={[{ required: true, message: '请输入联系人' }]}>
              <Input placeholder="法人或经办人姓名" />
            </Form.Item>
            <Form.Item name="contact_phone" label="联系电话" rules={[{ required: true, message: '请输入联系电话' }]}>
              <Input placeholder="手机号" />
            </Form.Item>
            <Form.Item name="contact_email" label="联系邮箱">
              <Input placeholder="联系邮箱" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} block style={{ height: 44, fontWeight: 500 }}>
              提交申请
            </Button>
          </Form>
        </Card>
      )}
    </div>
  )
}

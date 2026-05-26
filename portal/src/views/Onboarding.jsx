import { useState, useEffect, useCallback } from 'react'
import { Card, Form, Input, Select, Button, Tag, Alert, QRCode, Space, message } from 'antd'
import { LinkOutlined, ReloadOutlined } from '@ant-design/icons'
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
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    try {
      const r = await api.getOnboardingStatus()
      setRecord(r)
    } catch (e) {
      setRecord(null)
    }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  async function handleSubmit(values) {
    setSubmitting(true)
    try {
      const r = await api.initiateOnboarding(values)
      setRecord(r)
      message.success('进件申请已提交')
    } catch (err) {
      message.error(err.message)
    }
    setSubmitting(false)
  }

  if (loading) return <div style={{ color: '#94a3b8', textAlign: 'center', padding: 64 }}>加载中...</div>

  const hasPID = merchant.alipay_pid || merchant.wechat_sub_mchid

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 16 }}>商户进件</h2>

      {hasPID && (
        <Alert
          type="success"
          showIcon
          message="商户已认证"
          description={
            <div>
              {merchant.alipay_pid && <div>支付宝 PID：{merchant.alipay_pid}</div>}
              {merchant.wechat_sub_mchid && <div>微信子商户号：{merchant.wechat_sub_mchid}</div>}
            </div>
          }
          style={{ marginBottom: 16 }}
        />
      )}

      {record && record.status !== 'rejected' ? (
        <Card>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <Space>
              <span style={{ fontWeight: 500 }}>进件状态</span>
              <Tag color={statusMap[record.status]?.color}>{statusMap[record.status]?.label}</Tag>
              <span style={{ color: '#94a3b8', fontSize: 12 }}>渠道：{record.channel === 'alipay' ? '支付宝' : '微信支付'}</span>
            </Space>
            {record.status !== 'approved' && (
              <Button icon={<ReloadOutlined />} onClick={load} size="small">刷新</Button>
            )}
          </div>

          {record.sign_url && (
            <div style={{ marginBottom: 12 }}>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>签约链接</div>
              <a href={record.sign_url} target="_blank" rel="noopener noreferrer">
                <LinkOutlined /> 点击前往 {record.channel === 'alipay' ? '支付宝' : '微信支付'} 完成签约
              </a>
            </div>
          )}

          {record.qr_code_url && (
            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>扫码签约</div>
              <QRCode value={record.qr_code_url} size={180} />
            </div>
          )}

          {record.error_message && (
            <Alert type="error" message={record.error_message} style={{ marginTop: 12 }} />
          )}
        </Card>
      ) : (
        <Card title={record ? '重新发起进件' : '发起商户进件'}>
          {record?.status === 'rejected' && (
            <Alert type="error" message={`上次进件被拒绝：${record.error_message || '未知原因'}`} style={{ marginBottom: 16 }} />
          )}
          <Form form={form} layout="vertical" onFinish={handleSubmit}>
            <Form.Item name="channel" label="渠道" rules={[{ required: true }]}>
              <Select options={[
                { value: 'alipay', label: '支付宝', disabled: !!merchant.alipay_pid },
                { value: 'wechat', label: '微信支付', disabled: !!merchant.wechat_sub_mchid },
              ]} placeholder="选择进件渠道" />
            </Form.Item>
            <Form.Item name="merchant_name" label="商户名称" rules={[{ required: true }]}>
              <Input placeholder="营业执照上的企业名称" />
            </Form.Item>
            <Form.Item name="contact_name" label="联系人" rules={[{ required: true }]}>
              <Input placeholder="法人或经办人姓名" />
            </Form.Item>
            <Form.Item name="contact_phone" label="联系电话" rules={[{ required: true }]}>
              <Input placeholder="手机号" />
            </Form.Item>
            <Form.Item name="contact_email" label="联系邮箱">
              <Input placeholder="联系邮箱" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={submitting} block>提交进件申请</Button>
          </Form>
        </Card>
      )}
    </div>
  )
}

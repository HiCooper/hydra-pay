import { useState } from 'react'
import { Card, Form, Input, Button, message, Typography } from 'antd'
import { CopyOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

const { Text } = Typography

export default function Settings({ app, onUpdate }) {
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)

  async function handleSave(values) {
    setSaving(true)
    await api.updateSettings(values)
    const updated = await api.me()
    onUpdate(updated)
    message.success('已保存')
    setSaving(false)
  }

  async function copy(text) {
    await navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>设置</h2>

      <Card title="API Key" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
          <code style={{ flex: 1, padding: '8px 12px', background: '#f8fafc', borderRadius: 6, fontSize: 14, wordBreak: 'break-all' }}>{app.api_key_full}</code>
          <Button icon={<CopyOutlined />} onClick={() => copy(app.api_key_full)}>复制</Button>
        </div>
        <Text type="secondary" style={{ fontSize: 12 }}>此 Key 用于调用 /v1/* 支付 API，请妥善保管</Text>
      </Card>

      <Card title="子商户配置">
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            webhook_url: app.webhook_url || '',
            alipay_pid: app.alipay_pid || '',
            wechat_sub_mchid: app.wechat_sub_mchid || '',
            wechat_sub_appid: app.wechat_sub_appid || '',
          }}
        >
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
            <Form.Item name="alipay_pid" label="支付宝子商户 PID"><Input placeholder="2088..." /></Form.Item>
            <Form.Item name="wechat_sub_mchid" label="微信子商户号"><Input placeholder="子商户 mchid" /></Form.Item>
            <Form.Item name="wechat_sub_appid" label="微信子商户 AppID"><Input placeholder="wx..." /></Form.Item>
            <Form.Item name="webhook_url" label="Webhook 回调地址"><Input placeholder="https://your-app.com/payment/callback" /></Form.Item>
          </div>
          <Button type="primary" htmlType="submit" loading={saving}>保存</Button>
        </Form>
      </Card>
    </div>
  )
}
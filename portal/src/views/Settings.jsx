import { useState } from 'react'
import { Card, Form, Input, Button, message, Typography, Descriptions } from 'antd'
import { api } from '../api/index.js'

const { Text } = Typography

export default function Settings({ merchant, onUpdate }) {
  const [form] = Form.useForm()
  const [saving, setSaving] = useState(false)

  async function handleSave(values) {
    setSaving(true)
    const updated = await api.updateSettings(values)
    onUpdate(updated)
    message.success('已保存')
    setSaving(false)
  }

  return (
    <div>
      <h2 className="page-heading">账户设置</h2>
      <p className="page-subtitle">管理联系信息和账户安全</p>

      {/* Basic Info */}
      <Card
        title={<span style={{ fontSize: 15, fontWeight: 600, color: '#1a1a1a' }}>基本信息</span>}
        style={{ borderRadius: 8, marginBottom: 16 }}
        styles={{
          header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' },
          body: { padding: '20px 24px' },
        }}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            contact_name: merchant.contact_name || '',
            contact_phone: merchant.contact_phone || '',
          }}
        >
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
            <Form.Item name="contact_name" label="联系人">
              <Input placeholder="法人或经办人姓名" />
            </Form.Item>
            <Form.Item name="contact_phone" label="联系电话">
              <Input placeholder="手机号" />
            </Form.Item>
          </div>
          <Button type="primary" htmlType="submit" loading={saving}>保存</Button>
        </Form>
      </Card>

      {/* Change Password */}
      <Card
        title={<span style={{ fontSize: 15, fontWeight: 600, color: '#1a1a1a' }}>修改密码</span>}
        style={{ borderRadius: 8, marginBottom: 16 }}
        styles={{
          header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' },
          body: { padding: '20px 24px' },
        }}
      >
        <Form
          layout="vertical"
          onFinish={async (values) => {
            if (values.new_password !== values.confirm_password) {
              message.error('两次密码不一致')
              return
            }
            setSaving(true)
            await api.updateSettings({ password: values.new_password })
            message.success('密码已修改')
            setSaving(false)
          }}
        >
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 24px' }}>
            <Form.Item name="new_password" label="新密码" rules={[{ required: true, message: '请输入新密码' }]}>
              <Input.Password placeholder="至少 6 位" />
            </Form.Item>
            <Form.Item name="confirm_password" label="确认密码" rules={[{ required: true, message: '请确认新密码' }]}>
              <Input.Password placeholder="再次输入新密码" />
            </Form.Item>
          </div>
          <Button type="primary" htmlType="submit" loading={saving}>修改密码</Button>
        </Form>
      </Card>

      {/* Channel Info */}
      <Card
        title={<span style={{ fontSize: 15, fontWeight: 600, color: '#1a1a1a' }}>渠道信息</span>}
        style={{ borderRadius: 8 }}
        styles={{
          header: { borderBottom: '1px solid #f0f0f0', padding: '18px 24px' },
          body: { padding: '20px 24px' },
        }}
      >
        <Descriptions column={1} size="small" colon={false}>
          <Descriptions.Item
            label={<span style={{ color: '#6b6b6b', fontSize: 13 }}>支付宝 PID</span>}
          >
            {merchant.alipay_pid ? (
              <code style={{
                fontSize: 13, color: '#1a1a1a',
                fontFamily: "'SF Mono', Menlo, Monaco, monospace",
              }}>
                {merchant.alipay_pid}
              </code>
            ) : (
              <Text type="secondary" style={{ fontSize: 13 }}>未配置（请先申请支付渠道）</Text>
            )}
          </Descriptions.Item>
          <Descriptions.Item
            label={<span style={{ color: '#6b6b6b', fontSize: 13 }}>微信子商户号</span>}
          >
            {merchant.wechat_sub_mchid ? (
              <code style={{
                fontSize: 13, color: '#1a1a1a',
                fontFamily: "'SF Mono', Menlo, Monaco, monospace",
              }}>
                {merchant.wechat_sub_mchid}
              </code>
            ) : (
              <Text type="secondary" style={{ fontSize: 13 }}>未配置（请先申请支付渠道）</Text>
            )}
          </Descriptions.Item>
          <Descriptions.Item
            label={<span style={{ color: '#6b6b6b', fontSize: 13 }}>微信子商户 AppID</span>}
          >
            {merchant.wechat_sub_appid ? (
              <code style={{
                fontSize: 13, color: '#1a1a1a',
                fontFamily: "'SF Mono', Menlo, Monaco, monospace",
              }}>
                {merchant.wechat_sub_appid}
              </code>
            ) : (
              <Text type="secondary" style={{ fontSize: 13 }}>未配置</Text>
            )}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}

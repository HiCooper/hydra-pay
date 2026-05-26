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
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>账户设置</h2>

      <Card title="基本信息" style={{ marginBottom: 16 }}>
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

      <Card title="修改密码" style={{ marginBottom: 16 }}>
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

      <Card title="渠道信息">
        <Descriptions column={1} size="small">
          <Descriptions.Item label="支付宝 PID">
            {merchant.alipay_pid ? <code>{merchant.alipay_pid}</code> : <Text type="secondary">未配置（需完成进件）</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="微信子商户号">
            {merchant.wechat_sub_mchid ? <code>{merchant.wechat_sub_mchid}</code> : <Text type="secondary">未配置（需完成进件）</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="微信子商户 AppID">
            {merchant.wechat_sub_appid ? <code>{merchant.wechat_sub_appid}</code> : <Text type="secondary">未配置</Text>}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}

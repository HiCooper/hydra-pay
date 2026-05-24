import { useState, useEffect } from 'react'
import { Descriptions, Tag, Card, Spin } from 'antd'
import { CheckCircleOutlined, MinusCircleOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Config() {
  const [cfg, setCfg] = useState(null)
  useEffect(() => { api.config().then(setCfg) }, [])

  if (!cfg) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>

  const ok = <Tag icon={<CheckCircleOutlined />} color="success">已配置</Tag>
  const no = <Tag icon={<MinusCircleOutlined />} color="default">未配置</Tag>

  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>渠道配置</h2>

      <Card title="支付宝 ISV" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="App ID">{cfg.alipay?.app_id || '—'}</Descriptions.Item>
          <Descriptions.Item label="环境">{cfg.alipay?.sandbox === 'true' ? '沙箱' : '生产'}</Descriptions.Item>
          <Descriptions.Item label="ISV 私钥">{cfg.alipay?.key_loaded ? ok : no}</Descriptions.Item>
          <Descriptions.Item label="支付宝公钥">{cfg.alipay?.pub_loaded ? ok : no}</Descriptions.Item>
          <Descriptions.Item label="异步通知地址"><code>{cfg.alipay?.notify_url || '— 未配置'}</code></Descriptions.Item>
          <Descriptions.Item label="同步跳转地址"><code>{cfg.alipay?.return_url || '— 未配置'}</code></Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="微信支付服务商" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="small">
          <Descriptions.Item label="商户号">{cfg.wechat?.mch_id || '—'}</Descriptions.Item>
          <Descriptions.Item label="证书序列号">{cfg.wechat?.serial_no || '—'}</Descriptions.Item>
          <Descriptions.Item label="服务商私钥">{cfg.wechat?.key_loaded ? ok : no}</Descriptions.Item>
          <Descriptions.Item label="异步通知地址"><code>{cfg.wechat?.notify_url || '— 未配置'}</code></Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="全局 Webhook">
        <Descriptions column={1} size="small">
          <Descriptions.Item label="兜底 Webhook 地址（应用未单独配置时使用）">
            <code>{cfg.global_webhook || '— 未配置'}</code>
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}
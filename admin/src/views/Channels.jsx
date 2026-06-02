import { useState, useEffect } from 'react'
import { Table, Switch, Tag, Card, Descriptions, Spin, message } from 'antd'
import { CheckCircleOutlined, MinusCircleOutlined } from '@ant-design/icons'
import { api } from '../api/index.js'

export default function Channels() {
  const [channels, setChannels] = useState([])
  const [cfg, setCfg] = useState(null)
  const [loading, setLoading] = useState(true)

  async function load() {
    setLoading(true)
    try {
      const [chData, cfgData] = await Promise.all([api.listChannels(), api.config()])
      setChannels(chData.channels || [])
      setCfg(cfgData)
    } catch (err) {
      message.error(err.message)
    }
    setLoading(false)
  }

  useEffect(() => { load() }, [])

  async function toggleEnabled(record, checked) {
    try {
      await api.updateChannel(record.id, { enabled: checked })
      message.success(`${record.label} 已${checked ? '启用' : '停用'}`)
      load()
    } catch (err) {
      message.error(err.message)
    }
  }

  async function toggleOnboarding(record, checked) {
    try {
      await api.updateChannel(record.id, { supports_onboarding: checked })
      message.success(`${record.label} 进件支持已更新`)
      load()
    } catch (err) {
      message.error(err.message)
    }
  }

  const columns = [
    { title: '排序', dataIndex: 'sort_order', width: 60 },
    { title: '编码', dataIndex: 'code', width: 70, render: v => <code style={{ fontSize: 12 }}>{v || '—'}</code> },
    { title: '标识', dataIndex: 'key', width: 110, render: v => <code style={{ fontSize: 12 }}>{v}</code> },
    { title: '名称', dataIndex: 'label', width: 130 },
    {
      title: '支持进件', dataIndex: 'supports_onboarding', width: 100,
      render: (v, record) => (
        <Switch checked={v} size="small" onChange={(c) => toggleOnboarding(record, c)} />
      ),
    },
    {
      title: '状态', dataIndex: 'enabled', width: 80,
      render: (v) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '停用'}</Tag>,
    },
    {
      title: '启用', dataIndex: 'enabled', width: 80,
      render: (v, record) => (
        <Switch checked={v} size="small" onChange={(c) => toggleEnabled(record, c)} />
      ),
    },
  ]

  const ok = <Tag icon={<CheckCircleOutlined />} color="success">已配置</Tag>
  const no = <Tag icon={<MinusCircleOutlined />} color="default">未配置</Tag>

  return (
    <div>
      <h2 style={{ margin: '0 0 16px', fontSize: 18, fontWeight: 600 }}>支付渠道</h2>
      <p style={{ color: '#6b6b6b', fontSize: 13, marginBottom: 20 }}>
        管理 Hydra-Pay 集成的支付渠道，控制商户端可见性和进件能力
      </p>

      <Table
        columns={columns}
        dataSource={channels}
        rowKey="id"
        loading={loading}
        size="middle"
        pagination={false}
        locale={{ emptyText: '暂无渠道' }}
        style={{ marginBottom: 24 }}
      />

      {!cfg ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin size="default" /></div>
      ) : (
        <>
          <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 16 }}>渠道参数配置</h3>

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

          <Card title="银联 / 云闪付" style={{ marginBottom: 16 }}>
            <Descriptions column={2} size="small">
              <Descriptions.Item label="App ID">{cfg.unionpay?.app_id || '—'}</Descriptions.Item>
              <Descriptions.Item label="商户号">{cfg.unionpay?.mch_id || '—'}</Descriptions.Item>
              <Descriptions.Item label="商户私钥">{cfg.unionpay?.key_loaded ? ok : no}</Descriptions.Item>
              <Descriptions.Item label="银联公钥">{cfg.unionpay?.pub_loaded ? ok : no}</Descriptions.Item>
              <Descriptions.Item label="异步通知地址"><code>{cfg.unionpay?.notify_url || '— 未配置'}</code></Descriptions.Item>
              <Descriptions.Item label="同步跳转地址"><code>{cfg.unionpay?.return_url || '— 未配置'}</code></Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="全局 Webhook">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="兜底 Webhook 地址（应用未单独配置时使用）">
                <code>{cfg.global_webhook || '— 未配置'}</code>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </>
      )}
    </div>
  )
}

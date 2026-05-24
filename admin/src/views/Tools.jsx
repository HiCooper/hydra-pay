import { useState, useEffect } from 'react'
import { Tabs, Form, Input, Select, Button, Card, Table, Tag, message } from 'antd'
import { SERVER } from '../api/index.js'

const H = { 'X-Admin-Key': 'admin-dev-key', 'Content-Type': 'application/json' }

export default function Tools() {
  return (
    <div>
      <h2 style={{ fontSize: 18, fontWeight: 600, marginBottom: 20 }}>测试工具</h2>
      <Tabs
        items={[
          { key: 'quickpay', label: '快速下单', children: <QuickPay /> },
          { key: 'refund', label: '退款测试', children: <RefundTool /> },
	          { key: 'callback', label: '回调模拟', children: <CallbackSimulator /> },
          { key: 'webhook', label: 'Webhook 测试', children: <WebhookTester /> },
          { key: 'connectivity', label: '连通性检查', children: <Connectivity /> },
        ]}
      />
    </div>
  )
}

function QuickPay() {
  const [form] = Form.useForm()
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)

  async function submit(values) {
    setLoading(true)
    const res = await fetch(SERVER + '/v1/payments/create', {
      method: 'POST', headers: { 'X-API-Key': values.app_key, 'Content-Type': 'application/json' },
      body: JSON.stringify({
        user_id: 'test_user', amount: parseInt(values.amount), channel: values.channel,
        trade_type: values.trade_type, description: values.description,
        channel_app_id: values.channel === 'wechat' ? 'wx_test' : undefined,
        sub_merchant_id: values.sub_merchant_id || undefined,
        notify_url: values.notify_url || undefined,
      })
    })
    setResult(await res.json())
    setLoading(false)
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
      <Card title="创建测试支付">
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ channel: 'alipay', trade_type: 'native', amount: 1, description: '测试订单', sub_merchant_id: '2088721101618715', app_key: 'test-pay-key-001' }}>
          <Form.Item name="channel" label="渠道"><Select options={[{ value: 'alipay', label: '支付宝' }, { value: 'wechat', label: '微信' }]} /></Form.Item>
          <Form.Item name="trade_type" label="支付方式"><Select options={[{ value: 'native', label: '扫码支付' }, { value: 'h5', label: 'H5 支付' }, { value: 'jsapi', label: 'JSAPI 支付' }, { value: 'app', label: 'App 支付' }]} /></Form.Item>
          <Form.Item name="amount" label="金额 (分)"><Input type="number" min={1} /></Form.Item>
          <Form.Item name="description" label="描述"><Input /></Form.Item>
          <Form.Item name="sub_merchant_id" label="子商户 ID (服务商模式)"><Input placeholder="可选" /></Form.Item>
          <Form.Item name="notify_url" label="回调地址 (覆盖默认)"><Input placeholder="留空使用默认回调地址" /></Form.Item>
          <Form.Item name="app_key" label="API Key"><Input /></Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>创建支付</Button>
        </Form>
      </Card>
      <Card title="结果">
        {result ? (
          <div style={{ fontSize: 13 }}>
            <Tag color={result.success ? 'green' : 'red'}>{result.success ? '成功' : '失败'}</Tag>
            {result.success ? (() => {
                const qrUrl = result.data?.qr_code_url || result.data?.payment_url
                return (
                  <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>
                    <div>Trade No: <code>{result.data?.trade_no}</code></div>
                    <div>Status: {result.data?.status}</div>
                    {qrUrl && (
                      <div style={{ textAlign: 'center' }}>
                        <img
                          src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(qrUrl)}`}
                          alt="支付二维码" style={{ border: '1px solid #e2e8f0', borderRadius: 8, maxWidth: 180 }}
                        />
                        <div style={{ marginTop: 4 }}>
                          <a href={qrUrl} target="_blank" rel="noreferrer" style={{ fontSize: 12 }}>在浏览器中打开 ↗</a>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })() : (
              <div style={{ marginTop: 12 }}>
                <div>Error: {result.error?.code}</div>
                <div>Message: {result.error?.message}</div>
              </div>
            )}
          </div>
        ) : <span style={{ color: '#94a3b8', fontSize: 13 }}>填写表单并点击「创建支付」查看结果</span>}
      </Card>
    </div>
  )
}

function CallbackSimulator() {
  const [paymentId, setPaymentId] = useState('')
  const [result, setResult] = useState(null)

  async function simulate() {
    if (!paymentId) { message.warning('请输入 payment_id'); return }
    const res = await fetch(SERVER + '/api/admin/tools/simulate-callback', { method: 'POST', headers: H, body: JSON.stringify({ payment_id: paymentId, status: 'paid' }) })
    setResult(await res.json())
  }

  return (
    <Card title="回调模拟" style={{ maxWidth: 520 }}>
      <p style={{ color: '#94a3b8', fontSize: 13, marginBottom: 16 }}>直接标记指定订单为已支付状态，用于测试 webhook 通知和状态变更流程。不会真正调用支付宝/微信接口。</p>
      <Form.Item label="交易号 (TradeNo)"><Input.Search value={paymentId} onChange={e => setPaymentId(e.target.value)} onSearch={simulate} enterButton="模拟回调" placeholder="22位交易号，如 2026052400114411571487" /></Form.Item>
      {result && (
        <div style={{ padding: 12, borderRadius: 8, background: result.success ? '#ecfdf5' : '#fef2f2', fontSize: 13 }}>
          {result.success ? '✅ ' + result.data?.message : '❌ ' + (result.error?.message || '失败')}
          {result.data?.payment && <div style={{ color: '#64748b', marginTop: 4 }}>Status: {result.data.payment.Status}</div>}
        </div>
      )}
    </Card>
  )
}

function WebhookTester() {
  const [apps, setApps] = useState([])
  const [appId, setAppId] = useState('')
  const [result, setResult] = useState(null)

  useEffect(() => {
    fetch(SERVER + '/api/admin/apps', { headers: H }).then(r => r.json()).then(d => setApps(d.data || []))
  }, [])

  async function test() {
    if (!appId) { message.warning('请选择应用'); return }
    const res = await fetch(SERVER + '/api/admin/tools/test-webhook', { method: 'POST', headers: H, body: JSON.stringify({ app_id: appId }) })
    setResult(await res.json())
  }

  return (
    <Card title="Webhook 推送测试" style={{ maxWidth: 520 }}>
      <p style={{ color: '#94a3b8', fontSize: 13, marginBottom: 16 }}>向已配置 webhook_url 的应用发送一条测试消息，验证回调地址可达性。</p>
      <Form.Item label="选择应用">
        <Select
          value={appId || undefined} onChange={setAppId} style={{ width: '100%' }}
          placeholder="选择应用"
          options={apps.filter(a => a.WebhookURL).map(a => ({ value: a.ID, label: `${a.Name} (${a.WebhookURL?.slice(0, 40)})` }))}
        />
      </Form.Item>
      <Button type="primary" onClick={test} disabled={!appId}>发送测试</Button>
      {result && (
        <div style={{ marginTop: 16, padding: 12, borderRadius: 8, background: result.success ? '#ecfdf5' : '#fef2f2', fontSize: 13 }}>
          <div>{result.success ? '✅ ' + result.data?.message : '❌ ' + (result.error?.message || '失败')}</div>
          {result.data?.response_code && <div style={{ marginTop: 4 }}>HTTP {result.data.response_code}</div>}
        </div>
      )}
    </Card>
  )
}

function Connectivity() {
  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)

  async function check() {
    setLoading(true)
    const res = await fetch(SERVER + '/api/admin/tools/connectivity', { headers: H })
    setResults((await res.json()).data?.results || [])
    setLoading(false)
  }

  const columns = [
    { title: '渠道', dataIndex: 'channel', width: 80 },
    { title: '网关', dataIndex: 'gateway', render: v => <code style={{ fontSize: 12 }}>{v}</code> },
    { title: '状态', dataIndex: 'status', width: 100, render: v => <Tag color={v === 'unreachable' ? 'red' : 'green'}>{v}</Tag> },
    { title: '延迟', dataIndex: 'latency', width: 100 },
  ]

  return (
    <Card title="网关连通性检查" style={{ maxWidth: 520 }}>
      <p style={{ color: '#94a3b8', fontSize: 13, marginBottom: 16 }}>检测支付宝和微信支付网关的可达性。</p>
      <Button type="primary" onClick={check} loading={loading} style={{ marginBottom: 16 }}>开始检测</Button>
      {results && <Table columns={columns} dataSource={results} rowKey={(_, i) => i} size="small" pagination={false} />}
    </Card>
  )
}

function RefundTool() {
  const [form] = Form.useForm()
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)

  async function submit(values) {
    setLoading(true)
    const res = await fetch(SERVER + '/api/admin/tools/test-refund', {
      method: 'POST', headers: H,
      body: JSON.stringify({ trade_no: values.trade_no, refund_amount: values.refund_amount, refund_reason: values.refund_reason || '' }),
    })
    setResult(await res.json())
    setLoading(false)
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
      <Card title="退款测试（自动识别渠道）">
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ refund_amount: '0.01' }}>
          <Form.Item name="trade_no" label="交易号 (TradeNo)" rules={[{ required: true, message: '请输入交易号' }]}>
            <Input placeholder="22位交易号，如 2026052400114411571487" />
          </Form.Item>
          <Form.Item name="refund_amount" label="退款金额（元）" rules={[{ required: true, message: '请输入退款金额' }]}>
            <Input placeholder="0.01" />
          </Form.Item>
          <Form.Item name="refund_reason" label="退款原因">
            <Input placeholder="测试退款" />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={loading} block>申请退款</Button>
        </Form>
      </Card>
      <Card title="结果">
        {result ? (
          <div style={{ fontSize: 13 }}>
            <Tag color={result.success ? 'green' : 'red'}>{result.success ? '退款成功' : '退款失败'}</Tag>
            {result.success ? (
              <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>
                <div>渠道: {result.data?.channel}</div>
                <div>退款金额: ¥{result.data?.refund_fee || (result.data?.amount?.refund ? (result.data.amount.refund / 100).toFixed(2) : '—')}</div>
                {result.data?.trade_no && <div>支付宝交易号: <code>{result.data.trade_no}</code></div>}
                {result.data?.refund_id && <div>微信退款单号: <code>{result.data.refund_id}</code></div>}
                {result.data?.transaction_id && <div>微信交易号: <code>{result.data.transaction_id}</code></div>}
                {result.data?.status && <div>退款状态: <Tag color="green">{result.data.status}</Tag></div>}
              </div>
            ) : (
              <div style={{ marginTop: 12 }}>
                <div>Error: {result.error?.code}</div>
                <div>Message: {result.error?.message}</div>
              </div>
            )}
          </div>
        ) : <span style={{ color: '#94a3b8', fontSize: 13 }}>输入已支付订单的交易号，点击「申请退款」</span>}
      </Card>
    </div>
  )
}
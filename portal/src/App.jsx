import { Routes, Route, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { Layout, Menu, Button, Input, Form, message } from 'antd'
import { DashboardOutlined, OrderedListOutlined, SettingOutlined, LinkOutlined, SyncOutlined, LogoutOutlined, IdcardOutlined, AppstoreOutlined } from '@ant-design/icons'
import { api, login, logout, isLoggedIn } from './api/index.js'
import Dashboard from './views/Dashboard.jsx'
import Orders from './views/Orders.jsx'
import Settings from './views/Settings.jsx'
import PaymentLinks from './views/PaymentLinks.jsx'
import Subscriptions from './views/Subscriptions.jsx'
import Onboarding from './views/Onboarding.jsx'
import Apps from './views/Apps.jsx'
import './index.css'

const { Sider, Content } = Layout

function LoginPage() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function doLogin(values) {
    setLoading(true)
    try {
      const data = await api.login(values.email, values.password)
      login(data.token, { id: data.merchant_id, name: data.merchant_name })
      navigate('/')
    } catch (err) {
      message.error(err.message || '登录失败')
      setLoading(false)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg, #0f1322 0%, #1a1f36 50%, #2d3748 100%)' }}>
      <div style={{ width: 400, padding: '0 16px' }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ color: '#fff', fontSize: 24, fontWeight: 700, margin: 0 }}>星河支付</h1>
          <p style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14, marginTop: 8 }}>商户管理平台</p>
        </div>
        <div style={{ background: '#fff', borderRadius: 12, padding: 24 }}>
          <Form onFinish={doLogin} layout="vertical">
            <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }]}>
              <Input placeholder="admin@example.com" autoFocus />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password placeholder="输入密码" />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={loading} block>登录</Button>
            </Form.Item>
          </Form>
          <p style={{ color: '#94a3b8', fontSize: 12, textAlign: 'center', marginTop: 16, marginBottom: 0 }}>还没有账户？请联系管理员创建商户</p>
        </div>
      </div>
    </div>
  )
}

function PortalLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [data, setData] = useState(null)
  useEffect(() => { api.me().then(setData).catch(() => logout()) }, [])
  if (!data) return <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', color: '#94a3b8' }}>加载中...</div>

  const { merchant, apps } = data
  const selectedKey = '/' + location.pathname.split('/').filter(Boolean)[0]

  const navItems = [
    { key: '/', icon: <DashboardOutlined />, label: '概览' },
    { key: '/orders', icon: <OrderedListOutlined />, label: '支付订单' },
    { key: '/payment-links', icon: <LinkOutlined />, label: '支付链接' },
    { key: '/subscriptions', icon: <SyncOutlined />, label: '订阅管理' },
    { key: '/onboarding', icon: <IdcardOutlined />, label: '商户进件' },
    { key: '/apps', icon: <AppstoreOutlined />, label: '我的应用' },
    { key: '/settings', icon: <SettingOutlined />, label: '设置' },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} style={{ background: '#0f1322' }} theme="dark">
        <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
          <div style={{ color: '#fff', fontSize: 16, fontWeight: 700 }}>星河支付</div>
          <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: 12, marginTop: 4 }}>{merchant.name}</div>
        </div>
        <Menu
          theme="dark" mode="inline" selectedKeys={[selectedKey]}
          items={navItems}
          onClick={({ key }) => navigate(key)}
          style={{ background: 'transparent', borderInlineEnd: 'none', marginTop: 8 }}
        />
        <div style={{ padding: '12px 24px', borderTop: '1px solid rgba(255,255,255,0.06)', position: 'absolute', bottom: 0, width: '100%' }}>
          <div style={{ color: 'rgba(255,255,255,0.3)', fontSize: 12, marginBottom: 8 }}>{merchant.email || ''}</div>
          <Button type="text" icon={<LogoutOutlined />} onClick={logout} style={{ color: 'rgba(255,255,255,0.3)', fontSize: 12, padding: 0 }}>退出</Button>
        </div>
      </Sider>
      <Content style={{ padding: 32, background: '#f5f5f5', overflow: 'auto' }}>
        <Routes>
          <Route path="/" element={<Dashboard data={data} />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/payment-links" element={<PaymentLinks apps={apps} />} />
          <Route path="/subscriptions" element={<Subscriptions />} />
          <Route path="/onboarding" element={<Onboarding merchant={merchant} />} />
          <Route path="/apps" element={<Apps apps={apps} onUpdate={setData} />} />
          <Route path="/settings" element={<Settings merchant={merchant} onUpdate={(m) => setData(d => ({ ...d, merchant: { ...d.merchant, ...m } }))} />} />
        </Routes>
      </Content>
    </Layout>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={isLoggedIn() ? <Navigate to="/" /> : <LoginPage />} />
      <Route path="/*" element={isLoggedIn() ? <PortalLayout /> : <Navigate to="/login" />} />
    </Routes>
  )
}

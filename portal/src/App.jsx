import { Routes, Route, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { Layout, Menu, Button, Input, message } from 'antd'
import { DashboardOutlined, OrderedListOutlined, SettingOutlined, LogoutOutlined } from '@ant-design/icons'
import { api, login, logout, isLoggedIn } from './api/index.js'
import Dashboard from './views/Dashboard.jsx'
import Orders from './views/Orders.jsx'
import Settings from './views/Settings.jsx'
import './index.css'

const { Sider, Content } = Layout

function LoginPage() {
  const [key, setKey] = useState('')
  const [loading, setLoading] = useState(false)

  async function doLogin(e) {
    e.preventDefault()
    setLoading(true)
    login(key)
    try { await api.me(); window.location.href = '/portal' }
    catch (err) { logout(); message.error('API Key 无效'); setLoading(false) }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg, #0f1322 0%, #1a1f36 50%, #2d3748 100%)' }}>
      <div style={{ width: 400, padding: '0 16px' }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <h1 style={{ color: '#fff', fontSize: 24, fontWeight: 700, margin: 0 }}>星河支付</h1>
          <p style={{ color: 'rgba(255,255,255,0.5)', fontSize: 14, marginTop: 8 }}>开发者门户</p>
        </div>
        <div style={{ background: '#fff', borderRadius: 12, padding: 24 }}>
          <form onSubmit={doLogin}>
            <label style={{ display: 'block', fontSize: 12, fontWeight: 500, color: '#64748b', marginBottom: 8 }}>API Key</label>
            <Input.Password value={key} onChange={e => setKey(e.target.value)} placeholder="sk_..." style={{ marginBottom: 16 }} autoFocus />
            <Button type="primary" htmlType="submit" loading={loading} block>登录</Button>
          </form>
          <p style={{ color: '#94a3b8', fontSize: 12, textAlign: 'center', marginTop: 16, marginBottom: 0 }}>还没有 API Key？请联系管理员创建应用</p>
        </div>
      </div>
    </div>
  )
}

function PortalLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [app, setApp] = useState(null)
  useEffect(() => { api.me().then(setApp).catch(() => logout()) }, [])
  if (!app) return <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', color: '#94a3b8' }}>加载中...</div>

  const selectedKey = '/' + location.pathname.split('/').filter(Boolean)[0]

  const navItems = [
    { key: '/', icon: <DashboardOutlined />, label: '概览' },
    { key: '/orders', icon: <OrderedListOutlined />, label: '支付订单' },
    { key: '/settings', icon: <SettingOutlined />, label: '设置' },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider width={220} style={{ background: '#0f1322' }} theme="dark">
        <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
          <div style={{ color: '#fff', fontSize: 16, fontWeight: 700 }}>星河支付</div>
          <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: 12, marginTop: 4 }}>{app.name}</div>
        </div>
        <Menu
          theme="dark" mode="inline" selectedKeys={[selectedKey]}
          items={navItems}
          onClick={({ key }) => navigate(key)}
          style={{ background: 'transparent', borderInlineEnd: 'none', marginTop: 8 }}
        />
        <div style={{ padding: '12px 24px', borderTop: '1px solid rgba(255,255,255,0.06)', position: 'absolute', bottom: 0, width: '100%' }}>
          <code style={{ color: 'rgba(255,255,255,0.3)', fontSize: 12, display: 'block', marginBottom: 8 }}>{app.api_key_preview}</code>
          <Button type="text" icon={<LogoutOutlined />} onClick={logout} style={{ color: 'rgba(255,255,255,0.3)', fontSize: 12, padding: 0 }}>退出</Button>
        </div>
      </Sider>
      <Content style={{ padding: 32, background: '#f5f5f5', overflow: 'auto' }}>
        <Routes>
          <Route path="/" element={<Dashboard app={app} />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/settings" element={<Settings app={app} onUpdate={setApp} />} />
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
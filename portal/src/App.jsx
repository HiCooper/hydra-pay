import { Routes, Route, useNavigate, useLocation, Navigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { Layout, Menu, Button, Input, Form, message } from 'antd'
import { DashboardOutlined, OrderedListOutlined, SettingOutlined, LinkOutlined, SyncOutlined, LogoutOutlined, IdcardOutlined, AppstoreOutlined } from '@ant-design/icons'
import { api, saveLogin, doLogout, isLoggedIn } from './api/index.js'
import Dashboard from './views/Dashboard.jsx'
import Orders from './views/Orders.jsx'
import Settings from './views/Settings.jsx'
import PaymentLinks from './views/PaymentLinks.jsx'
import Subscriptions from './views/Subscriptions.jsx'
import Onboarding from './views/Onboarding.jsx'
import Apps from './views/Apps.jsx'
import './index.css'

const { Sider, Content } = Layout

function LoginPage({ onLoginSuccess }) {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function doLogin(values) {
    setLoading(true)
    try {
      const data = await api.login(values.email, values.password)
      saveLogin(data.token, { id: data.merchant_id, name: data.merchant_name })
      onLoginSuccess()
      navigate('/', { replace: true })
    } catch (err) {
      message.error(err.message || '登录失败')
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#f7f7f7',
    }}>
      <div style={{
        width: 420,
        padding: '0 16px',
      }}>
        {/* Brand */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{
            width: 48, height: 48, borderRadius: 12,
            background: '#de481b',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            marginBottom: 16,
          }}>
            <span style={{ color: '#fff', fontSize: 20, fontWeight: 700 }}>星</span>
          </div>
          <h1 style={{
            color: '#1a1a1a', fontSize: 22, fontWeight: 600, margin: '0 0 4px',
            letterSpacing: '-0.3px',
          }}>星河支付</h1>
          <p style={{ color: '#6b6b6b', fontSize: 14, margin: 0 }}>商户管理平台</p>
        </div>

        {/* Login card */}
        <div style={{
          background: '#fff',
          borderRadius: 12,
          padding: '28px 28px 24px',
          border: '1px solid #e6e6e6',
        }}>
          <Form onFinish={doLogin} layout="vertical" size="large">
            <Form.Item
              name="email"
              label={<span style={{ fontSize: 13, fontWeight: 500, color: '#1a1a1a' }}>邮箱</span>}
              rules={[{ required: true, message: '请输入邮箱' }]}
            >
              <Input placeholder="admin@example.com" autoFocus />
            </Form.Item>
            <Form.Item
              name="password"
              label={<span style={{ fontSize: 13, fontWeight: 500, color: '#1a1a1a' }}>密码</span>}
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password placeholder="输入密码" />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0, marginTop: 8 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                block
                style={{ height: 44, fontSize: 15, fontWeight: 500, borderRadius: 6 }}
              >
                登录
              </Button>
            </Form.Item>
          </Form>
          <p style={{
            color: '#999', fontSize: 12, textAlign: 'center',
            marginTop: 20, marginBottom: 0, lineHeight: 1.5,
          }}>
            还没有账户？请联系管理员创建商户
          </p>
        </div>

        {/* Footer */}
        <p style={{
          textAlign: 'center', color: '#bbb', fontSize: 11,
          marginTop: 24,
        }}>
          Powered by HydraPay
        </p>
      </div>
    </div>
  )
}

function PortalLayout({ onLogout }) {
  const navigate = useNavigate()
  const location = useLocation()
  const [data, setData] = useState(null)
  const [loadError, setLoadError] = useState(false)
  const [retryKey, setRetryKey] = useState(0)

  useEffect(() => {
    let cancelled = false
    let retries = 0

    function load() {
      api.me().then(d => {
        if (!cancelled) { setData(d); setLoadError(false) }
      }).catch(err => {
        if (cancelled) return
        if (err.name === 'AuthError') { onLogout(); return }
        // Network error (server restart) — retry up to 5 times with backoff
        if (retries < 5) {
          retries++
          setTimeout(load, Math.min(1000 * retries, 5000))
        } else {
          setLoadError(true)
        }
      })
    }
    load()
    return () => { cancelled = true }
  }, [retryKey])

  if (!data) return loadError ? (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      minHeight: '100vh', color: '#6b6b6b', fontSize: 14, background: '#f7f7f7', gap: 16,
    }}>
      <span>网络连接失败，请检查服务是否正常运行</span>
      <button
        onClick={() => { setLoadError(false); setRetryKey(k => k + 1) }}
        style={{
          background: '#de481b', color: '#fff', border: 'none', borderRadius: 6,
          padding: '8px 24px', fontSize: 13, cursor: 'pointer', fontWeight: 500,
        }}>
        重新加载
      </button>
    </div>
  ) : (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      minHeight: '100vh', color: '#6b6b6b', fontSize: 14, background: '#f7f7f7',
    }}>
      加载中...
    </div>
  )

  const { merchant, apps } = data
  const selectedKey = '/' + location.pathname.split('/').filter(Boolean)[0]

  const navItems = [
    { key: '/', icon: <DashboardOutlined />, label: '概览' },
    { key: '/orders', icon: <OrderedListOutlined />, label: '支付订单' },
    { key: '/payment-links', icon: <LinkOutlined />, label: '支付链接' },
    { key: '/subscriptions', icon: <SyncOutlined />, label: '订阅管理' },
    { key: '/onboarding', icon: <IdcardOutlined />, label: '支付渠道管理' },
    { key: '/apps', icon: <AppstoreOutlined />, label: '我的应用' },
    { key: '/settings', icon: <SettingOutlined />, label: '设置' },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* Dark sidebar — inspired by Stripe subscription checkout left panel */}
      <Sider
        width={220}
        style={{
          background: '#0a0a0a',
          borderRight: 'none',
        }}
        theme="dark"
      >
        {/* Brand header */}
        <div style={{
          padding: '22px 24px 18px',
          borderBottom: '1px solid rgba(255,255,255,0.06)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div style={{
              width: 32, height: 32, borderRadius: 8,
              background: '#de481b',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              flexShrink: 0,
            }}>
              <span style={{ color: '#fff', fontSize: 14, fontWeight: 700 }}>星</span>
            </div>
            <div>
              <div style={{ color: '#fff', fontSize: 14, fontWeight: 600 }}>星河支付</div>
              <div style={{ color: 'rgba(255,255,255,0.3)', fontSize: 11, marginTop: 1, lineHeight: 1.2 }}>
                {merchant.name}
              </div>
            </div>
          </div>
        </div>

        {/* Navigation */}
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={navItems}
          onClick={({ key }) => navigate(key)}
          style={{
            background: 'transparent',
            borderInlineEnd: 'none',
            marginTop: 12,
            padding: '0 8px',
          }}
        />

        {/* Bottom user area */}
        <div style={{
          padding: '14px 24px',
          borderTop: '1px solid rgba(255,255,255,0.06)',
          position: 'absolute',
          bottom: 0,
          width: '100%',
        }}>
          <div style={{
            color: 'rgba(255,255,255,0.25)',
            fontSize: 11,
            marginBottom: 10,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            {merchant.email || ''}
          </div>
          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={onLogout}
            style={{
              color: 'rgba(255,255,255,0.35)',
              fontSize: 12,
              padding: '0 0 0 2px',
              height: 'auto',
            }}
          >
            退出登录
          </Button>
        </div>
      </Sider>

      {/* Content area */}
      <Content style={{
        padding: 32,
        background: '#f7f7f7',
        overflow: 'auto',
        minHeight: '100vh',
      }}>
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
  const [authed, setAuthed] = useState(() => isLoggedIn())

  function handleLogin() {
    setAuthed(true)
  }

  function handleLogout() {
    doLogout()
    setAuthed(false)
  }

  return (
    <Routes>
      <Route path="/login" element={authed ? <Navigate to="/" replace /> : <LoginPage onLoginSuccess={handleLogin} />} />
      <Route path="/*" element={authed ? <PortalLayout onLogout={handleLogout} /> : <Navigate to="/login" replace />} />
    </Routes>
  )
}

import { Routes, Route, NavLink, Navigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { api, login, logout, isLoggedIn } from './api/index.js'
import Dashboard from './views/Dashboard.jsx'
import Orders from './views/Orders.jsx'
import Settings from './views/Settings.jsx'

function LoginPage() {
  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  async function doLogin(e) {
    e.preventDefault()
    login(key)
    try { await api.me(); window.location.href = '/portal' }
    catch (err) { logout(); setError('API Key 无效') }
  }
  return (
    <div className="min-h-screen flex items-center justify-center" style={{ background: 'linear-gradient(135deg, #0f1322 0%, #1a1f36 50%, #2d3748 100%)' }}>
      <div className="w-full max-w-md px-4">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold tracking-tight" style={{ color: '#fff' }}>星河支付</h1>
          <p className="text-sm mt-2" style={{ color: 'rgba(255,255,255,0.5)' }}>开发者门户</p>
        </div>
        <div className="card p-6">
          <form onSubmit={doLogin}>
            <label className="block text-xs font-medium mb-2" style={{ color: 'var(--color-text-secondary)' }}>API Key</label>
            <input value={key} onChange={e => setKey(e.target.value)} placeholder="sk_..." className="w-full mb-3 font-mono" autoFocus />
            {error && <p className="text-xs mb-3" style={{ color: 'var(--color-danger)' }}>{error}</p>}
            <button type="submit" className="btn btn-primary w-full justify-center">登录</button>
          </form>
          <p className="text-xs mt-4 text-center" style={{ color: 'var(--color-text-muted)' }}>还没有 API Key？请联系管理员创建应用</p>
        </div>
      </div>
    </div>
  )
}

function PortalLayout() {
  const [app, setApp] = useState(null)
  useEffect(() => { api.me().then(setApp).catch(() => logout()) }, [])
  if (!app) return <div className="flex items-center justify-center min-h-screen" style={{ color: 'var(--color-text-muted)' }}>加载中...</div>

  return (
    <div className="flex min-h-screen">
      <aside className="w-[220px] flex-shrink-0 flex flex-col" style={{ background: 'var(--color-navy-900)' }}>
        <div className="px-5 py-6 border-b" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <h1 className="text-base font-bold tracking-tight" style={{ color: '#fff' }}>星河支付</h1>
          <p className="text-xs mt-1" style={{ color: 'rgba(255,255,255,0.35)' }}>{app.name}</p>
        </div>
        <nav className="flex-1 py-4">
          <NavLink to="/" end className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>概览</NavLink>
          <NavLink to="/portal/orders" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>支付订单</NavLink>
          <NavLink to="/portal/settings" className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}>设置</NavLink>
          <a href="/docs-site" target="_blank" className="nav-link" style={{ marginTop: 16, borderTop: '1px solid rgba(255,255,255,0.06)', paddingTop: 16, borderRadius: 0 }}>API 文档 ↗</a>
        </nav>
        <div className="px-5 py-4 border-t flex items-center justify-between" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <code className="text-xs" style={{ color: 'rgba(255,255,255,0.3)' }}>{app.api_key_preview}</code>
          <button onClick={logout} className="text-xs" style={{ color: 'rgba(255,255,255,0.3)', background: 'none', border: 'none', cursor: 'pointer' }}>退出</button>
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto p-8">
        <Routes>
          <Route path="/" element={<Dashboard app={app} />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/settings" element={<Settings app={app} onUpdate={setApp} />} />
        </Routes>
      </main>
    </div>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={isLoggedIn() ? <Navigate to="/" /> : <LoginPage />} />
      <Route path="/*" element={isLoggedIn() ? <PortalLayout /> : <Navigate to="/portal/login" />} />
    </Routes>
  )
}

import { Routes, Route, NavLink } from 'react-router-dom'
import Dashboard from './views/Dashboard.jsx'
import Apps from './views/Apps.jsx'
import Orders from './views/Orders.jsx'
import OrderDetail from './views/OrderDetail.jsx'
import Config from './views/Config.jsx'
import Tools from './views/Tools.jsx'

const navItems = [
  { to: '/', label: '仪表盘', icon: '◫' },
  { to: '/apps', label: '应用管理', icon: '⊞' },
  { to: '/orders', label: '支付订单', icon: '⊟' },
  { to: '/tools', label: '测试工具', icon: '▷' },
  { to: '/config', label: '渠道配置', icon: '⚙' },
]

export default function App() {
  return (
    <div className="flex min-h-screen">
      <aside className="w-[232px] flex-shrink-0 flex flex-col" style={{ background: 'var(--color-navy-900)' }}>
        <div className="px-5 py-6 border-b" style={{ borderColor: 'rgba(255,255,255,0.06)' }}>
          <h1 className="text-base font-bold tracking-tight" style={{ color: '#fff' }}>星河支付</h1>
          <p className="text-xs mt-1" style={{ color: 'rgba(255,255,255,0.35)' }}>服务商管理后台</p>
        </div>
        <nav className="flex-1 py-4">
          {navItems.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
            >
              <span className="text-base w-5 text-center" style={{ opacity: 0.7 }}>{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="px-5 py-4 border-t text-xs" style={{ borderColor: 'rgba(255,255,255,0.06)', color: 'rgba(255,255,255,0.25)' }}>
          v0.1.0
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto p-8">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/apps" element={<Apps />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/orders/:id" element={<OrderDetail />} />
          <Route path="/config" element={<Config />} />
          <Route path="/tools" element={<Tools />} />
        </Routes>
      </main>
    </div>
  )
}

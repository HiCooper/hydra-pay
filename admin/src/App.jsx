import { Routes, Route, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu } from 'antd'
import { DashboardOutlined, AppstoreOutlined, OrderedListOutlined, ToolOutlined, SettingOutlined, ShopOutlined } from '@ant-design/icons'
import Dashboard from './views/Dashboard.jsx'
import Apps from './views/Apps.jsx'
import Orders from './views/Orders.jsx'
import Config from './views/Config.jsx'
import Tools from './views/Tools.jsx'
import Merchants from './views/Merchants.jsx'
import './index.css'

const { Sider, Content } = Layout

const navItems = [
  { key: '/', label: '仪表盘', icon: <DashboardOutlined /> },
  { key: '/merchants', label: '商户管理', icon: <ShopOutlined /> },
  { key: '/apps', label: '应用管理', icon: <AppstoreOutlined /> },
  { key: '/orders', label: '支付订单', icon: <OrderedListOutlined /> },
  { key: '/tools', label: '测试工具', icon: <ToolOutlined /> },
  { key: '/config', label: '渠道配置', icon: <SettingOutlined /> },
]

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()

  const selectedKey = '/' + location.pathname.split('/').filter(Boolean)[0]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        width={232}
        style={{ background: '#0f1322' }}
        theme="dark"
      >
        <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
          <div style={{ color: '#fff', fontSize: 16, fontWeight: 700 }}>星河支付</div>
          <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: 12, marginTop: 4 }}>服务商管理后台</div>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={navItems}
          onClick={({ key }) => navigate(key)}
          style={{ background: 'transparent', borderInlineEnd: 'none', marginTop: 8 }}
        />
        <div style={{ padding: '12px 24px', borderTop: '1px solid rgba(255,255,255,0.06)', color: 'rgba(255,255,255,0.25)', fontSize: 12, position: 'absolute', bottom: 0, width: '100%' }}>
          v0.1.0
        </div>
      </Sider>
      <Content style={{ padding: 32, background: '#f5f5f5', overflow: 'auto' }}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/merchants" element={<Merchants />} />
          <Route path="/apps" element={<Apps />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/config" element={<Config />} />
          <Route path="/tools" element={<Tools />} />
        </Routes>
      </Content>
    </Layout>
  )
}
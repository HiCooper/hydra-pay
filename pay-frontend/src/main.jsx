import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ConfigProvider, App as AntdApp } from 'antd'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ConfigProvider theme={{
      token: {
        colorPrimary: '#de481b',
        colorSuccess: '#04d66f',
        colorError: '#df1b41',
        colorTextBase: '#1a1a1a',
        colorBgBase: '#ffffff',
        colorBorder: '#e6e6e6',
        borderRadius: 6,
        fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        fontSize: 14,
        colorText: '#1a1a1a',
        colorTextSecondary: '#6b6b6b',
        colorTextTertiary: '#999999',
        boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      },
      components: {
        Button: {
          borderRadius: 6,
          fontWeight: 500,
          primaryShadow: 'none',
        },
        Modal: {
          borderRadius: 12,
        },
      },
    }}>
      <AntdApp>
        <BrowserRouter basename="/pay">
          <App />
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  </React.StrictMode>
)

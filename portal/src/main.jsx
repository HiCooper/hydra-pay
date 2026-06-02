// DEBUG: catch what triggers page refresh
window.addEventListener('beforeunload', (e) => {
  console.trace('[portal] beforeunload triggered');
})
window.addEventListener('unload', () => {
  console.log('[portal] unload triggered');
})

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ConfigProvider, App as AntdApp } from 'antd'
import App from './App.jsx'
import './index.css'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <ConfigProvider theme={{
      token: {
        colorPrimary: '#de481b',
        colorSuccess: '#04d66f',
        colorWarning: '#f2921b',
        colorError: '#df1b41',
        colorInfo: '#de481b',
        colorTextBase: '#1a1a1a',
        colorBgBase: '#ffffff',
        colorBorder: '#e6e6e6',
        borderRadius: 6,
        fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        fontSize: 14,
        colorText: '#1a1a1a',
        colorTextSecondary: '#6b6b6b',
        colorTextTertiary: '#999999',
        colorBgContainer: '#ffffff',
        colorBgLayout: '#f7f7f7',
        colorBorderSecondary: '#e6e6e6',
        controlHeight: 38,
        lineHeight: 1.5,
        paddingContentHorizontal: 20,
        paddingContentVertical: 16,
        boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
        boxShadowSecondary: '0 2px 8px rgba(0,0,0,0.06)',
      },
      components: {
        Button: {
          borderRadius: 6,
          controlHeight: 38,
          paddingContentHorizontal: 20,
          fontWeight: 500,
          primaryShadow: 'none',
        },
        Card: {
          borderRadius: 8,
          padding: 20,
          colorBorderSecondary: '#e6e6e6',
        },
        Table: {
          borderRadius: 8,
          borderColor: '#e6e6e6',
          headerBg: '#fafafa',
          headerColor: '#6b6b6b',
          rowHoverBg: '#fafafa',
          cellPaddingBlock: 10,
          cellPaddingInline: 16,
        },
        Input: {
          borderRadius: 6,
          controlHeight: 38,
          colorBorder: '#e0e0e0',
          hoverBorderColor: '#c0c0c0',
          activeBorderColor: '#de481b',
        },
        Menu: {
          itemBg: 'transparent',
          subMenuItemBg: 'transparent',
          darkItemBg: 'transparent',
          darkSubMenuItemBg: 'transparent',
          darkItemColor: '#a0a0a0',
          darkItemHoverColor: '#ffffff',
          darkItemSelectedColor: '#ffffff',
          darkItemHoverBg: 'rgba(255,255,255,0.04)',
        },
        Layout: {
          siderBg: '#0a0a0a',
        },
        Statistic: {
          contentFontSize: 28,
          titleFontSize: 13,
        },
        Tag: {
          borderRadius: 4,
        },
        Modal: {
          borderRadius: 12,
        },
      },
    }}>
      <AntdApp>
        <BrowserRouter basename="/portal">
          <App />
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  </StrictMode>
)

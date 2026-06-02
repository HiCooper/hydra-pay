import { Routes, Route } from 'react-router-dom'
import CheckoutPage from './pages/CheckoutPage'
import SuccessPage from './pages/SuccessPage'
import CheckoutPageV2 from './pages/CheckoutPageV2'
import SuccessPageV2 from './pages/SuccessPageV2'

export default function App() {
  return (
    <Routes>
      {/* Original checkout */}
      <Route path="/checkout/:sessionId" element={<CheckoutPage />} />
      <Route path="/checkout/:sessionId/success" element={<SuccessPage />} />

      {/* V2 — Stripe Checkout clone */}
      <Route path="/v2/checkout/:sessionId" element={<CheckoutPageV2 />} />
      <Route path="/v2/checkout/:sessionId/success" element={<SuccessPageV2 />} />
    </Routes>
  )
}

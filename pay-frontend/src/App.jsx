import { Routes, Route } from 'react-router-dom'
import CheckoutPage from './pages/CheckoutPage'
import SuccessPage from './pages/SuccessPage'

export default function App() {
  return (
    <Routes>
      <Route path="/checkout/:sessionId" element={<CheckoutPage />} />
      <Route path="/checkout/:sessionId/success" element={<SuccessPage />} />
    </Routes>
  )
}

import { useState, useEffect } from 'react'

export default function CountdownTimer({ expiresAt, inline }) {
  const [remaining, setRemaining] = useState('')

  useEffect(() => {
    function tick() {
      const diff = new Date(expiresAt) - new Date()
      if (diff <= 0) {
        setRemaining('00:00')
        return
      }
      const m = Math.floor(diff / 60000)
      const s = Math.floor((diff % 60000) / 1000)
      setRemaining(`${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`)
    }
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [expiresAt])

  const diff = new Date(expiresAt) - new Date()
  const urgent = diff < 5 * 60 * 1000
  const expired = diff <= 0

  return (
    <span style={{
      fontSize: 12,
      fontWeight: urgent && !expired ? 600 : 400,
      color: expired ? '#d0d0d0' : urgent ? '#de481b' : '#6b6b6b',
    }}>
      {expired ? '已过期' : inline ? remaining : `订单有效期: ${remaining}`}
    </span>
  )
}

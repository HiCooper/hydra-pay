(function () {
  'use strict'

  var BASE_URL = document.currentScript
    ? new URL(document.currentScript.src).origin
    : window.location.origin

  function resolveContainer(container) {
    if (typeof container === 'string') {
      return document.querySelector(container)
    }
    return container
  }

  function createIframe(src) {
    var iframe = document.createElement('iframe')
    iframe.src = src
    iframe.style.border = 'none'
    iframe.style.width = '100%'
    iframe.style.minHeight = '500px'
    iframe.allow = 'payment'
    return iframe
  }

  function HydraPayCheckout(options) {
    if (!options || !options.sessionId) {
      throw new Error('HydraPay: sessionId is required')
    }

    var container = resolveContainer(options.container)
    if (!container) {
      throw new Error('HydraPay: container element not found')
    }

    var baseUrl = options.baseUrl || BASE_URL
    var origin = encodeURIComponent(window.location.origin)
    var src = baseUrl + '/pay/checkout/' + options.sessionId + '?embed=true&origin=' + origin

    var iframe = createIframe(src)
    var resolved = false

    function cleanup() {
      if (resolved) return
      resolved = true
      if (iframe.parentNode) {
        iframe.parentNode.removeChild(iframe)
      }
      window.removeEventListener('message', handler)
    }

    function handler(event) {
      // Only accept messages from the payment service origin
      if (event.origin !== new URL(baseUrl).origin) return

      var data = event.data
      if (!data || !data.type) return

      switch (data.type) {
        case 'hydra-pay:ready':
          if (options.onReady) options.onReady(data)
          break
        case 'hydra-pay:success':
          cleanup()
          if (options.onSuccess) options.onSuccess(data)
          break
        case 'hydra-pay:cancel':
          cleanup()
          if (options.onCancel) options.onCancel(data)
          break
        case 'hydra-pay:error':
          cleanup()
          if (options.onError) options.onError(data)
          break
        case 'hydra-pay:expired':
          cleanup()
          if (options.onExpired) options.onExpired(data)
          break
        case 'hydra-pay:completed':
          cleanup()
          if (options.onCompleted) options.onCompleted(data)
          break
        case 'hydra-pay:redirect':
          cleanup()
          window.location.href = data.url
          break
      }
    }

    window.addEventListener('message', handler)
    container.innerHTML = ''
    container.appendChild(iframe)
  }

  window.HydraPay = {
    checkout: {
      create: HydraPayCheckout,
    },
  }
})()

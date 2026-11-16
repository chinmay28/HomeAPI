/* HomeAPI for Android — what the app adds to HomeAPI's page.

   Runs before the page's own scripts, on the server's origin only. A web view
   has no Downloads and no share sheet of its own, and nothing outside the
   page can read a blob: address, so this file turns the two ways a page
   hands a file over into messages to the app (see PageBridge.kt and
   core/Bridge.kt for the other side):

   - saving: an <a download> clicked on a blob: or data: address — Settings →
     Export is one — is read here and sent across in chunks; one on the
     server's own address is fetched by the app.
   - navigator.share(), which a web view lacks: the files go across the same
     way, then the app opens the phone's share sheet.

   It also says the page is inside the app, as window.homeapiApp, so the page
   can offer the app's own settings. Nothing here runs on any other site. */
(function () {
  'use strict'
  if (window.homeapiApp) return
  var port = window.HomeAPIAndroid
  if (!port || typeof port.postMessage !== 'function') return

  // Base64 grows a chunk by a third; 768 KB of file is a 1 MB message.
  var CHUNK = 768 * 1024
  var nextRe = 1
  var waiting = {}

  port.addEventListener('message', function (event) {
    var reply
    try { reply = JSON.parse(event.data) } catch (e) { return }
    var w = waiting[reply.re]
    if (!w) return
    delete waiting[reply.re]
    if (reply.ok) {
      w.resolve(reply)
    } else {
      var err = new Error(reply.error || 'The app could not do that.')
      if (reply.cancelled) err.name = 'AbortError'
      w.reject(err)
    }
  })

  function call(msg) {
    return new Promise(function (resolve, reject) {
      var re = nextRe++
      waiting[re] = { resolve: resolve, reject: reject }
      msg.re = re
      port.postMessage(JSON.stringify(msg))
    })
  }

  function base64Of(blob) {
    return new Promise(function (resolve, reject) {
      var reader = new FileReader()
      reader.onload = function () {
        var s = String(reader.result)
        resolve(s.slice(s.indexOf(',') + 1))
      }
      reader.onerror = function () { reject(reader.error) }
      reader.readAsDataURL(blob)
    })
  }

  // Each chunk waits for the app's answer before the next is read, so a big
  // file is never more than one chunk ahead of the disk.
  async function send(blob, name, action) {
    var begun = await call({ t: 'begin', name: name || '', mime: blob.type || '', size: blob.size, action: action })
    var id = begun.id
    try {
      for (var seq = 0, off = 0; off < blob.size; seq++, off += CHUNK) {
        var data = await base64Of(blob.slice(off, off + CHUNK))
        await call({ t: 'chunk', id: id, seq: seq, data: data })
      }
      return await call({ t: 'end', id: id })
    } catch (e) {
      port.postMessage(JSON.stringify({ t: 'abort', id: id, re: 0 }))
      throw e
    }
  }

  function warn(e) { console.warn('HomeAPI app:', e && e.message ? e.message : e) }

  /* An <a download>: true when the app took it. */
  function takeAnchor(a) {
    if (!a || typeof a.hasAttribute !== 'function' || !a.hasAttribute('download')) return false
    var href = a.href || ''
    var name = a.getAttribute('download') || ''
    if (/^(blob|data):/i.test(href)) {
      fetch(href)
        .then(function (r) { return r.blob() })
        .then(function (b) { return send(b, name, 'save') })
        .catch(warn)
      return true
    }
    if (href.indexOf(location.origin + '/') === 0) {
      call({ t: 'fetch', url: href, name: name }).catch(warn)
      return true
    }
    return false
  }

  // The page saves by creating an anchor and calling click() on it.
  var click = HTMLAnchorElement.prototype.click
  HTMLAnchorElement.prototype.click = function () {
    if (takeAnchor(this)) return
    return click.apply(this, arguments)
  }
  // And a person may tap one.
  document.addEventListener('click', function (e) {
    if (e.defaultPrevented) return
    var a = e.target && e.target.closest ? e.target.closest('a[download]') : null
    if (a && takeAnchor(a)) e.preventDefault()
  }, true)

  function canShare(data) {
    if (!data) return false
    if (data.files && data.files.length) return true
    return Boolean(data.url || data.text || data.title)
  }

  async function share(data) {
    if (!canShare(data)) throw new TypeError('Nothing to share.')
    var ids = []
    var files = Array.prototype.slice.call(data.files || [])
    for (var i = 0; i < files.length; i++) {
      ids.push((await send(files[i], files[i].name, 'stage')).id)
    }
    await call({ t: 'share', ids: ids, title: data.title || '', text: data.text || '', url: data.url || '' })
  }

  try {
    Object.defineProperty(Navigator.prototype, 'share', { value: share, configurable: true, writable: true })
    Object.defineProperty(Navigator.prototype, 'canShare', { value: canShare, configurable: true, writable: true })
  } catch (e) { warn(e) }

  window.homeapiApp = Object.freeze({
    platform: 'android',
    // "remote": HomeAPI is on another machine; "on_phone": it is this phone.
    mode: "__HOMEAPI_MODE__",
    openSettings: function () { return call({ t: 'settings' }) },
  })
})()

var view = { page: 0, filter: '', backup: {} }

$('#infinite-table-0').on('click', 'a[data-filter]', function (event) {
  const username = event.target.innerText
  setFilter(username)
})

$('#infinite-table-1').on('click', 'a[data-filter]', function (event) {
  setFilter('')
})

$('#filter-icon').on('click', function (event) {
  setFilter('')
})

var infScroll = new InfiniteScroll('#infinite-scroll-container', {
  path: function () {
    return '/get_runs/?page=' + view.page + view.filter
  },
  responseType: 'text',
  history: false,
  checkLastPage: false,
  scrollThreshold: 400
})

infScroll.on('load', function (response) {
  view.page++
  const id = (view.filter ? '1' : '0')
  const r = response.split('!-- End of machines --')
  if (r.length > 1) { document.getElementById('accordion').innerHTML = r[0] }
  const rows = r[r.length - 1]
  document.getElementById('infinite-table-' + id).insertAdjacentHTML('beforeend', rows)
  const eof = document.getElementById('end-of-rows')
  setEOF(eof !== null)
  if (eof !== null) { eof.parentNode.removeChild(eof) }
  const tmp = document.getElementById('page-signature-data')
  if (tmp !== null) {
    const elem = document.getElementById('page-signature')
    elem.dataset.signature = tmp.dataset.signature
    tmp.parentNode.removeChild(tmp) // Remove once has been saved in 'page-signature'
  }
})

function startWebSocket () {
  const parser = document.createElement('a')
  parser.href = window.location.href
  const wsuri = 'wss://' + parser.hostname + '/runs_ws/'

  const sock = new WebSocket(wsuri)

  sock.onopen = function () {
    console.log('connected to ' + wsuri)
    document.getElementById('ws-connected-icon').classList.remove('text-secondary')
  }
  sock.onclose = function (e) {
    console.log('connection closed (' + e.code + ')')
    document.getElementById('ws-connected-icon').classList.add('text-secondary')
  }
  sock.onmessage = function (e) {
    console.log('message received: ' + e.data)
    sock.send('pong')
    const elem = document.getElementById('page-signature')
    if (elem === null) { return }
    const update = JSON.parse(e.data)
    const sign = elem.dataset.signature
    if (sign !== update.SignOld) {
      console.log('Wrong signature, closing')
      sock.close()
      return
    }
    elem.dataset.signature = update.SignNew
    // Update data...
  }
}

function setEOF (eof) {
  view.eof = eof
  infScroll.options.loadOnScroll = !view.eof
  if (view.eof) { $('#waiting-icon').hide() } else { $('#waiting-icon').show() }
}

function setFilter (username) {
  if (!view.filter && username) {
    view.backup.page = view.page
    view.backup.scroll = $(window).scrollTop()
    view.backup.eof = view.eof
    view.backup.machines = document.getElementById('accordion').innerHTML
    view.filter = '&username=' + username
    view.page = 0
    setEOF(false)
    infScroll.loadNextPage()
    document.getElementById('filter-icon').classList.remove('text-secondary')
    document.getElementById('infinite-table-0').style.display = 'none'
    document.getElementById('infinite-table-1').style.display = 'block'
  } else if (view.filter) {
    view.filter = ''
    view.page = view.backup.page
    setEOF(view.backup.eof)
    document.getElementById('filter-icon').classList.add('text-secondary')
    document.getElementById('infinite-table-1').style.display = 'none'
    document.getElementById('infinite-table-0').style.display = 'block'
    document.getElementById('infinite-table-1').innerHTML = ''
    document.getElementById('accordion').innerHTML = view.backup.machines
    $(window).scrollTop(view.backup.scroll)
  }
}

infScroll.loadNextPage() // Will start first page loading
startWebSocket()

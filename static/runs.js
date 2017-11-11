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
  var sock = {}

  var icon = document.getElementById('ws-connected-icon')
  icon.onclick = function (e) {
    if (!sock || sock.readyState === sock.CLOSED) {
      icon.style.visibility = 'hidden'

      // Reset main view and reload first page
      if (view.filter) { setFilter('') }
      document.getElementById('infinite-table-0').innerHTML = ''
      document.getElementById('infinite-table-1').innerHTML = ''
      const elem = document.getElementById('page-signature').dataset.signature = ''
      view.page = 0
      setEOF(false)
      infScroll.loadNextPage()

      // (re)create a new socket (we can't reconnect)
      sock = new WebSocket(wsuri)

      sock.onopen = function () {
        console.log('Socket: connected to ' + wsuri)
        icon.classList.remove('text-secondary')
        icon.style.visibility = 'visible'
      }
      sock.onclose = function (e) {
        console.log('Socket: connection closed (' + e.code + ')')
        icon.classList.add('text-secondary')
        icon.style.visibility = 'visible'
      }
      sock.onmessage = function (e) {
        console.log('Socket: received ' + e.data.length + ' bytes')
        sock.send('pong')
        const list = JSON.parse(e.data)
        const elem = document.getElementById('page-signature')
        const sign = elem.dataset.signature
        if (sign && sign !== list.SignOld) {
          console.log('Wrong signature, closing')
          sock.close()
          return
        }
        elem.dataset.signature = list.SignNew
        updateRows(list.Diff)
      }
    } else if (sock.readyState === sock.OPEN) {
      icon.style.visibility = 'hidden'
      sock.close()
    }
  }

  icon.click() // Load first page
}

function updateRows (diff) {
  for (let i = 0; i < diff.length; i++) {
    let rows = document.getElementsByClassName('row' + diff[i].Id)
    let item = diff[i].Item
    for (let k = 0; k < rows.length; k++) {
      switch (item.Field) {
        case 'LedColor':
          var target = rows[k].getElementsByTagName('small')[0]
          target.style['color'] = item.Value
          break
        case 'Workers':
          var target = rows[k].getElementsByTagName('a')[0]
          target.innerHTML = item.Value
          break
        case 'BoxColor':
          var target = rows[k].getElementsByClassName('card')[0]
          target.style['background-color'] = item.Value
          break
        case 'Border':
          var target = rows[k].getElementsByClassName('card')[0]
          target.style['border-color'] = item.Value
          break
        case 'Info':
          var target = rows[k].getElementsByClassName('card-subtitle')[0]
          target.innerHTML = item.Value
          break
        case 'Games':
          break
        default:
          console.log('Unknown field ' + item.Field)
      }
    }
    // For each active task save the queue of the last games count ordered
    // by most recent to oldest. It will be used to compute games per minute.
    if (item.Field === 'Games') {
      let games = document.getElementById('coll' + diff[i].Id.substring(0, 7))
      games = games.getElementsByClassName('machines-table')[0]
      let s = games.dataset.games
      if (s) { s = s.split(' ') } else { s = [] }
      s.unshift(item.Value)
      games.dataset.games = s.slice(0, 20).join(' ')
    }
  }
  if ($('#machines').is(':visible')) { $('#machines').trigger('shown.bs.modal') }
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

startWebSocket() // Will load first page

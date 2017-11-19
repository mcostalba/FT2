$('#infinite-table-0').on('click', 'a[data-collapseid]', function (event) {
  var collapseId = event.target.dataset.collapseid
  $('#' + collapseId).collapse('show')
  $('#machines').modal()
})

$('#infinite-table-1').on('click', 'a[data-collapseid]', function (event) {
  var collapseId = event.target.dataset.collapseid
  $('#' + collapseId).collapse('show')
  $('#machines').modal()
})

$('.modal-dialog').draggable({
  handle: '.modal-header'
})

google.charts.load('current', { 'packages': ['gauge'] })
google.charts.setOnLoadCallback(function () {
  var label = function (label) {
    return google.visualization.arrayToDataTable([
      ['Label', 'Value'],
      [label, 0]
    ])
  }
  var opt = function (max) {
    return {
      width: 120,
      height: 120,
      redFrom: 90 * max / 100,
      redTo: 100 * max / 100,
      yellowFrom: 75 * max / 100,
      yellowTo: 90 * max / 100,
      minorTicks: 5,
      max
    }
  }
  var chart = {
    gpm: label('GPM'),
    machines: label('Machines'),
    cores: label('Cores'),
    nps: label('MNps'),
    flags: label('Flags'),
    optGpm: opt(2000),
    optMachines: opt(150),
    optCores: opt(600),
    optNps: opt(1000),
    optFlags: opt(50),
    gauges: {
      gpm: new google.visualization.Gauge(document.getElementById('ch-gpm')),
      machines: new google.visualization.Gauge(document.getElementById('ch-machines')),
      cores: new google.visualization.Gauge(document.getElementById('ch-cores')),
      nps: new google.visualization.Gauge(document.getElementById('ch-mnps')),
      flags: new google.visualization.Gauge(document.getElementById('ch-flags'))
    },
    setValues: function (v) {
      const osInfo = `Linux: ${v.os.linux}\nWindows: ${v.os.windows}\nMac: ${v.os.mac}\nOther: ${v.os.other}`
      document.getElementById('ch-os').innerHTML = osInfo

      this.gpm.setValue(0, 1, v.gpm)
      this.machines.setValue(0, 1, v.machines)
      this.cores.setValue(0, 1, v.cores)
      this.nps.setValue(0, 1, Math.round(v.nps))
      this.flags.setValue(0, 1, v.flagSet.size)

      this.gauges.gpm.draw(this.gpm, this.optGpm)
      this.gauges.machines.draw(this.machines, this.optMachines)
      this.gauges.cores.draw(this.cores, this.optCores)
      this.gauges.nps.draw(this.nps, this.optNps)
      this.gauges.flags.draw(this.flags, this.optFlags)
    }
  }
  $('#machines').on('shown.bs.modal', function () {
    let os = { linux: 0, windows: 0, mac: 0, other: 0 }
    let v = {gpm: 0, cores: 0, nps: 0, machines: 0, flagSet: new Set(), os}
    const tables = document.getElementsByClassName('machines-table')

    for (let i = 0; i < tables.length; i++) {
      // Compute games-per-minute starting from the queue of game counters
      // sent by websocket updates every 'tick' seconds.
      let games = tables[i].dataset.games
      if (games) {
        let s = games.split(' ', 13) // Assume 5 seconds tick, use samples 0 and 12
        if (s.length > 1) {
          let value = parseInt(s[0]) - parseInt(s[s.length - 1])
          value = Math.floor(value * 12 / (s.length - 1))
          if (value > 0) { v.gpm += value }
        }
      }
      let machines = tables[i].rows.length, cores = 0, nps = 0.0
      for (let j = 0, row; row = tables[i].rows[j]; j++) {
        v.flagSet.add(row.cells[1].innerHTML)
        cores += parseInt(row.cells[3].innerHTML, 10)
        nps += parseFloat(row.cells[4].innerHTML, 10)
        v.machines++
        let uname = row.cells[2].innerHTML
        if (uname.includes('Linux')) {
          v.os.linux++
        } else if (uname.includes('Windows')) {
          v.os.windows++
        } else if (uname.includes('Darwin')) {
          v.os.mac++
        } else {
          v.os.other++
        }
      }
      v.cores += cores
      v.nps += nps
      let sum = machines + ' for ' + cores + ' cores ' + nps.toFixed(1) + ' Mnps'
      let card = tables[i]
      while ((card = card.parentNode) && !card.classList.contains('card')) {}
      card.getElementsByClassName('summary')[0].innerHTML = sum
    }
    chart.setValues(v)
  })
  $('#machines').on('hidden.bs.modal', function () {
    $('.collapse').collapse('hide')
    let os = { linux: 0, windows: 0, mac: 0, other: 0 }
    let v = {gpm: 0, cores: 0, nps: 0, machines: 0, flagSet: new Set(), os}
    chart.setValues(v)
  }).trigger('hidden.bs.modal')
})

function updateMachines (diff) {
  for (let i = 0; i < diff.length; i++) {
    let collapse = document.getElementById('coll' + diff[i].Id.substring(0, 7))
    let item = diff[i].Item
    let mkey = diff[i].Mkey
    switch (item.Field) {
      case 'Games':
        // For each active task save the queue of the last games count ordered
        // by most recent to oldest. It will be used to compute games per minute.
        let games = collapse.getElementsByClassName('machines-table')[0]
        let s = games.dataset.games
        if (s) { s = s.split(' ') } else { s = [] }
        s.unshift(item.Value)
        games.dataset.games = s.slice(0, 20).join(' ')
        break
      case 'Idle':
        let muted = 'text-muted'
        let light = 'font-weight-light'
        let cl = document.getElementById(mkey).classList
        if (item.Value === 'true') { cl.add(muted, light) } else { cl.remove(muted, light) }
        break
      case 'Mnps':
        document.getElementById(mkey).children[4].innerText = item.Value
        break
      case 'Remove':
        let elem = document.getElementById(mkey)
        elem.parentNode.removeChild(elem)
        break
      case 'Add':
        let tbody = collapse.getElementsByTagName('tbody')[0]
        tbody.insertAdjacentHTML('beforeend', item.Value)
        break
      default:
    }
  }
  if ($('#machines').is(':visible')) { $('#machines').trigger('shown.bs.modal') }
}

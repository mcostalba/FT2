$('#infinite-table').on('click', 'a[data-collapseid]', function (event) {
  var collapseId = event.target.dataset.collapseid
  $('#' + collapseId).collapse('show')
  $('#machines').modal()
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
    machines: label('Machines'),
    cores: label('Cores'),
    nps: label('MNps'),
    flags: label('Flags'),
    optMachines: opt(150),
    optCores: opt(600),
    optNps: opt(1000),
    optFlags: opt(50),
    gauges: {
      machines: new google.visualization.Gauge(document.getElementById('ch-machines')),
      cores: new google.visualization.Gauge(document.getElementById('ch-cores')),
      nps: new google.visualization.Gauge(document.getElementById('ch-mnps')),
      flags: new google.visualization.Gauge(document.getElementById('ch-flags'))
    },
    setValues: function (v) {
      const osInfo = `Linux: ${v.os.linux}\nWindows: ${v.os.windows}\nMac: ${v.os.mac}\nOther: ${v.os.other}`
      document.getElementById('ch-os').innerHTML = osInfo

      this.machines.setValue(0, 1, v.machines)
      this.cores.setValue(0, 1, v.cores)
      this.nps.setValue(0, 1, Math.round(v.nps))
      this.flags.setValue(0, 1, v.flagSet.size)

      this.gauges.machines.draw(this.machines, this.optMachines)
      this.gauges.cores.draw(this.cores, this.optCores)
      this.gauges.nps.draw(this.nps, this.optNps)
      this.gauges.flags.draw(this.flags, this.optFlags)
    }
  }
  $('#machines').on('shown.bs.modal', function () {
    let os = { linux: 0, windows: 0, mac: 0, other: 0 }
    let v = {cores: 0, nps: 0, machines: 0, flagSet: new Set(), os}
    const tables = document.getElementsByClassName('machines-table')

    for (let i = 0; i < tables.length; i++) {
      for (let j = 0, row; row = tables[i].rows[j]; j++) {
        v.flagSet.add(row.cells[1].innerHTML)
        v.cores += parseInt(row.cells[3].innerHTML, 10)
        v.nps += parseFloat(row.cells[4].innerHTML, 10)
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
    }
    chart.setValues(v)
  })
  $('#machines').on('hidden.bs.modal', function () {
    $('.collapse').collapse('hide')
    let os = { linux: 0, windows: 0, mac: 0, other: 0 }
    let v = {cores: 0, nps: 0, machines: 0, flagSet: new Set(), os}
    chart.setValues(v)
  }).trigger('hidden.bs.modal')
})

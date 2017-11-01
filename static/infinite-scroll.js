var chart = {};
var view = { page: 0, filter: "", eof: false, backup: {} };

var infScroll = new InfiniteScroll('#infinitetable', {
  path: function () {
    return '/get_runs/?page=' + view.page + view.filter;
  },
  responseType: 'text',
  history: 'false',
  checkLastPage: 'false',
  scrollThreshold: 400,
});

infScroll.on('load', function (response) {
  view.page++;
  const r = response.split("!-- End of machines --");
  const rows = r[r.length - 1];
  if (r.length > 1) {
    machinesContainer.innerHTML = r[0];
    google.charts.load('current', { 'packages': ['gauge'] });
    google.charts.setOnLoadCallback(setupGauges);
  }
  infinitetable.insertAdjacentHTML('beforeend', rows);
  setEOF(end_of_rows !== null);
});

function setEOF(eof) {

  view.eof = eof;
  infScroll.options.loadOnScroll = !view.eof;
  if (view.eof)
    $("#waitingicon").hide();
  else
    $("#waitingicon").show();
}

function setFilter(username) {

  if (!view.filter && username) {
    view.backup.page = view.page;
    view.backup.scroll = $(window).scrollTop();
    view.backup.eof = view.eof;
    view.filter = '&username=' + username;
    view.page = 0;
    setEOF(false);
    infScroll.loadNextPage();
    filtericon.classList.remove('text-secondary');
    view.backup.machines = machinesContainer.cloneNode(true);
    view.backup.table = infinitetable.innerHTML;
    infinitetable.innerHTML = "";
  } else if (view.filter) {
    view.filter = "";
    view.page = view.backup.page;
    setEOF(view.backup.eof);
    filtericon.classList.add('text-secondary');
    infinitetable.innerHTML = view.backup.table;
    machinesContainer.parentNode.replaceChild(view.backup.machines, machinesContainer);
    $(window).scrollTop(view.backup.scroll);
    view.backup.table = null;
    view.backup.machines = null;
  }
}

function showMachines(collapseId) {
  $('#' + collapseId).collapse('show');
  $('#machines').modal();
}

function updateGauges() {

  let cores = 0, nps = 0, machines = 0, flagSet = new Set();
  let os = { linux: 0, windows: 0, mac: 0, other: 0 };
  const tables = document.getElementsByClassName('machines-table');

  for (let i = 0; i < tables.length; i++) {
    for (let j = 0, row; row = tables[i].rows[j]; j++) {
      flagSet.add(row.cells[1].innerHTML);
      cores += parseInt(row.cells[3].innerHTML, 10);
      nps += parseFloat(row.cells[4].innerHTML, 10);
      machines++;
      let uname = row.cells[2].innerHTML;
      if (uname.includes("Linux")) {
        os.linux++;
      } else if (uname.includes("Windows")) {
        os.windows++;
      } else if (uname.includes("Darwin")) {
        os.mac++;
      } else {
        os.other++;
      }
    }
  }

  const osInfo = `Linux: ${os.linux}\nWindows: ${os.windows}\nMac: ${os.mac}\nOther: ${os.other}`;
  document.getElementById('ch-os').innerHTML = osInfo;

  chart.machines.setValue(0, 1, machines);
  chart.cores.setValue(0, 1, cores);
  chart.nps.setValue(0, 1, Math.round(nps));
  chart.flags.setValue(0, 1, flagSet.size);

  chart.gauges.machines.draw(chart.machines, chart.optMachines);
  chart.gauges.cores.draw(chart.cores, chart.optCores);
  chart.gauges.nps.draw(chart.nps, chart.optNps);
  chart.gauges.flags.draw(chart.flags, chart.optFlags);
}

function resetGauges() {

  const osInfo = "Linux: 0\nWindows: 0\nMac: 0\nOther: 0";
  document.getElementById('ch-os').innerHTML = osInfo;

  chart.machines.setValue(0, 1, 0);
  chart.cores.setValue(0, 1, 0);
  chart.nps.setValue(0, 1, 0);
  chart.flags.setValue(0, 1, 0);

  chart.gauges.machines.draw(chart.machines, chart.optMachines);
  chart.gauges.cores.draw(chart.cores, chart.optCores);
  chart.gauges.nps.draw(chart.nps, chart.optNps);
  chart.gauges.flags.draw(chart.flags, chart.optFlags);

  $('.collapse').collapse('hide');
}

function setupGauges() {

  chart = {
    machines: google.visualization.arrayToDataTable([
      ['Label', 'Value'],
      ['Machines', 0],
    ]),
    optMachines: {
      width: 120, height: 120,
      redFrom: 90, redTo: 100,
      yellowFrom: 75, yellowTo: 90,
      minorTicks: 5,
      max: 100
    },
    cores: google.visualization.arrayToDataTable([
      ['Label', 'Value'],
      ['Cores', 0],
    ]),
    optCores: {
      width: 120, height: 120,
      redFrom: 450, redTo: 500,
      yellowFrom: 375, yellowTo: 450,
      minorTicks: 5,
      max: 500
    },
    nps: google.visualization.arrayToDataTable([
      ['Label', 'Value'],
      ['MNps', 0]
    ]),
    optNps: {
      width: 120, height: 120,
      redFrom: 900, redTo: 1000,
      yellowFrom: 750, yellowTo: 900,
      minorTicks: 5,
      max: 1000
    },
    flags: google.visualization.arrayToDataTable([
      ['Label', 'Value'],
      ['Flags', 0]
    ]),
    optFlags: {
      width: 120, height: 120,
      redFrom: 45, redTo: 50,
      yellowFrom: 37, yellowTo: 45,
      minorTicks: 5,
      max: 50
    },
    gauges: {
      machines: new google.visualization.Gauge(document.getElementById('ch-machines')),
      cores: new google.visualization.Gauge(document.getElementById('ch-cores')),
      nps: new google.visualization.Gauge(document.getElementById('ch-mnps')),
      flags: new google.visualization.Gauge(document.getElementById('ch-flags')),
    },
  };

  resetGauges();

  $('#machines').on('shown.bs.modal', function () { updateGauges(); });
  $('#machines').on('hidden.bs.modal', function () { resetGauges(); });
}

infScroll.loadNextPage(); // load initial page

// Small script to implement infinite scrolling of 'runs' view
// using infinite-scroll package

var infScroll = new InfiniteScroll( '#infinitetable', {
  path: function() {
    return '/get_runs/?page=' + this.loadCount;
  },
  responseType: 'text',
  history: 'false',
});

infScroll.on( 'load', function( response ) {
  const r = response.split("!-- End of machines --");
  infinitetable.insertAdjacentHTML('beforeend', r[r.length-1]);
  if (r.length > 1)
    machinesContainer.innerHTML = r[0];
});

function showMachines(run) {
  $(".collapse").collapse('hide');
  $("#"+run).collapse('show');
  $("#machines").modal();
}

infScroll.loadNextPage(); // load initial page

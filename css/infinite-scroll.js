// Small script to implement infinite scrolling of 'runs' view
// using infinite-scroll package

var infScroll = new InfiniteScroll( '#infinitetable', {
  path: function() {
    return '/get_runs/?page=' + this.loadCount;
  },
  responseType: 'text',
  history: 'false',
})

infScroll.on( 'load', function( response ) {
  infinitetable.insertAdjacentHTML('beforeend', response);
});

infScroll.loadNextPage(); // load initial page

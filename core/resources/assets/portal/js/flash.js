$(document).ready(function() {
  var f = $('#flash-message');
  if (f.length) {
    var t = f.data('flash-type');
    var msg = f.data('flash-message');
    window.$flare.notify[t](msg);
  }
});


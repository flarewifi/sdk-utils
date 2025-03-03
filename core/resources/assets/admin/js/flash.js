window.addEventListener('DOMContentLoaded', function() {
  var f = document.getElementById('flash-message');
  if (f) {
    var t = f.getAttribute('data-flash-type');
    var msg = f.getAttribute('data-flash-message');
    window.$flare.notify[t](msg);
  }
})

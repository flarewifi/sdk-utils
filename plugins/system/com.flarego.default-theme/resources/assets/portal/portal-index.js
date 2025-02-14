var $ = require('@flarehotspot/lib/vendor/jquery-v1.12.4')

$(document).ready(function(){
  var scriptEl = $('#portal-data');
  var data = scriptEl.text();
  // var url = data.session_sync_url;
  console.log(data);
});

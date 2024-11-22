var jQuery = require('../vendor/jquery-1.12.4/jquery-1.12.4.min.js')
window.$ = jQuery;
window.jQuery = jQuery;

require('../vendor/bootstrap-3.4.1/js/bootstrap.js')

// window.DomReady(function () {
//   var events = window.PortalEvents;

//   events.addEventListener("client:connected", function (res) {
//     var data = JSON.parse(res.data);
//     console.log(data);
//   });

//   events.addEventListener("client:disconnected", function (res) {
//     var data = JSON.parse(res.data);
//     console.log(data);
//   });

//   events.onerror = function (e) {
//     console.log("Socket error:", e);
//   };
// });

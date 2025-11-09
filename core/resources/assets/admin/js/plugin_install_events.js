window.addEventListener('DOMContentLoaded', function() {
  console.log("Listening for install:progress...");

  window.$flare.events.on("install:progress", function(res) {
    var payload = typeof res.data === "string" ? JSON.parse(res.data) : res.data;

    window.loadNotifications();
    if (payload.status === 0) {
      window.$flare.notify.success(payload.message);
    } else {
      window.$flare.notify.error(payload.message);
    }
  });
});

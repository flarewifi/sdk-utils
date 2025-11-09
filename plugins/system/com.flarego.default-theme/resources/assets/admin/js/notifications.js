window.$flare.events.ready(function () {
  window.$flare.events.on("flare_notification", function (event) {
    var data = JSON.parse(event.data);

    if (data.type === "info") {
      window.$flare.notify.info(data.subject);
    } else if (data.type === "success") {
      window.$flare.notify.success(data.subject);
    } else if (data.type === "error") {
      window.$flare.notify.error(data.subject);
    } else if (data.type === "warn") {
      window.$flare.notify.warn(data.subject);
    }
  });
});

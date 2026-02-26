(function () {
  "use strict";

  // Revenue chart initialization using $flare.ui.line_chart
  function initRevenueChart() {
    var el = document.getElementById("revenueChart");
    if (!el) return;

    var dataAttr = el.getAttribute("data-chart");
    var chartData = [];
    try {
      chartData = JSON.parse(dataAttr || "[]");
    } catch (e) {
      chartData = [];
    }

    $flare.ui.line_chart.create(el, {
      data: chartData,
      series: [
        { key: "coinslot", color: "#3b82f6", label: "Coinslot" },
        { key: "voucher",  color: "#a855f7", label: "Voucher" }
      ],
      yAxis: { min: 0, max: 600, stepSize: 150 },
      tension: 0.4,
      fillOpacity: [0.35, 0.05],
      padding: { right: 30 },
      tooltipFormat: function (label, value) {
        return label + ": \u20B1" + value.toFixed(2);
      }
    });
  }

  // Copy device ID to clipboard
  function initCopyButtons() {
    var btns = document.querySelectorAll("[data-fw-copy]");
    for (var i = 0; i < btns.length; i++) {
      (function (btn) {
        btn.addEventListener("click", function () {
          var text = btn.getAttribute("data-fw-copy");
          var iconDefault = btn.querySelector(".fw-copy-icon-default");
          var iconCheck = btn.querySelector(".fw-copy-icon-check");

          if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(function () {
              showCopied(btn, iconDefault, iconCheck);
            }, function () {
              fallbackCopy(text, btn, iconDefault, iconCheck);
            });
          } else {
            fallbackCopy(text, btn, iconDefault, iconCheck);
          }
        });
      })(btns[i]);
    }
  }

  function fallbackCopy(text, btn, iconDefault, iconCheck) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
      showCopied(btn, iconDefault, iconCheck);
    } catch (e) {
      // silent fail
    }
    document.body.removeChild(ta);
  }

  function showCopied(btn, iconDefault, iconCheck) {
    btn.classList.add("fw-copied");
    if (iconDefault) iconDefault.style.display = "none";
    if (iconCheck) iconCheck.style.display = "inline";
    setTimeout(function () {
      btn.classList.remove("fw-copied");
      if (iconDefault) iconDefault.style.display = "inline";
      if (iconCheck) iconCheck.style.display = "none";
    }, 2000);
  }

  // Last updated timestamp
  function updateLastUpdated() {
    var el = document.getElementById("fw-last-updated-time");
    if (!el) return;
    var now = new Date();
    var months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
    var month = months[now.getMonth()];
    var day = now.getDate();
    var year = now.getFullYear();
    var hours = now.getHours();
    var minutes = now.getMinutes();
    var ampm = hours >= 12 ? "PM" : "AM";
    hours = hours % 12 || 12;
    var mm = minutes < 10 ? "0" + minutes : "" + minutes;
    var seconds = now.getSeconds();
    var ss = seconds < 10 ? "0" + seconds : "" + seconds;
    var label = el.getAttribute("data-label") || "Last updated";
    el.textContent = label + ": " + month + " " + day + ", " + year + " " + hours + ":" + mm + ":" + ss + " " + ampm;
  }

  // Re-initialize chart after htmx settles new content into the DOM.
  // htmx:afterSettle fires after the swap is complete and new elements are live.
  document.addEventListener("htmx:afterSettle", function () {
    var el = document.getElementById("revenueChart");
    if (el && !el.querySelector("svg")) {
      initRevenueChart();
    }
    updateLastUpdated();
  });

  // Initialize on DOM ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      initRevenueChart();
      initCopyButtons();
      updateLastUpdated();
    });
  } else {
    initRevenueChart();
    initCopyButtons();
    updateLastUpdated();
  }
})();

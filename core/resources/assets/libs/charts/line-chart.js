/**
 * $flare.ui.line_chart — Lightweight SVG stacked area chart utility (~4KB)
 * ES5, no dependencies. Exposes $flare.ui.line_chart.create(container, options).
 */
(function () {
  "use strict";

  var ns = "http://www.w3.org/2000/svg";
  var instanceCount = 0;

  function create(container, opts) {
    if (!container) return;

    var id = ++instanceCount;
    var data = opts.data || [];
    var series = opts.series || [];
    var tension = opts.tension != null ? opts.tension : 0.4;
    var fillOpacity = opts.fillOpacity || [0.8, 0.1];
    var tooltipFormat = opts.tooltipFormat || null;
    var pad = merge({ top: 10, right: 10, bottom: 30, left: 50 }, opts.padding || {});

    // Y-axis config
    var yAxis = opts.yAxis || {};
    var yMin = yAxis.min != null ? yAxis.min : 0;
    var yMax = yAxis.max != null ? yAxis.max : autoMax(data, series);
    var yStep = yAxis.stepSize || niceStep(yMax - yMin);

    // Ensure container fills its parent
    container.style.width = "100%";
    container.style.height = "100%";

    // Build SVG
    var svg = createEl("svg", { "class": "fw-chart-svg" });
    var defs = createEl("defs");
    svg.appendChild(defs);

    // Tooltip element
    var tooltip = document.createElement("div");
    tooltip.className = "fw-chart-tooltip";
    container.style.position = "relative";
    container.innerHTML = "";
    container.appendChild(svg);
    container.appendChild(tooltip);

    // Clip path to constrain curves to plot area (prevents Catmull-Rom overshoot)
    var clipId = "fwclip-" + id;
    var clipPath = createEl("clipPath", { id: clipId });
    var clipRect = createEl("rect");
    clipPath.appendChild(clipRect);
    defs.appendChild(clipPath);

    // Groups
    var gGrid = createEl("g", { "class": "fw-chart-grid" });
    var gAreas = createEl("g", { "clip-path": "url(#" + clipId + ")" });
    var gLines = createEl("g", { "clip-path": "url(#" + clipId + ")" });
    var gYLabels = createEl("g", { "class": "fw-chart-y-labels" });
    var gXLabels = createEl("g", { "class": "fw-chart-x-labels" });
    var indicator = createEl("line", { "class": "fw-chart-indicator" });
    var gHover = createEl("g", { style: "cursor: crosshair" });

    // Hover dots — small circles at each series intersection
    var hoverDots = [];
    for (var hd = 0; hd < series.length; hd++) {
      var dot = createEl("circle", {
        r: 5,
        fill: "#fff",
        stroke: series[hd].color,
        "stroke-width": 2,
        "class": "fw-chart-hover-dot"
      });
      hoverDots.push(dot);
    }

    // Render-time state for hover positioning
    var _stacked = [];
    var _yMin = yMin, _yMax = yMax, _plotB = 0, _plotH = 0;

    svg.appendChild(gGrid);
    svg.appendChild(gAreas);
    svg.appendChild(gLines);
    svg.appendChild(gYLabels);
    svg.appendChild(gXLabels);
    svg.appendChild(indicator);
    for (var hdi = 0; hdi < hoverDots.length; hdi++) {
      svg.appendChild(hoverDots[hdi]);
    }
    svg.appendChild(gHover);

    // Gradient defs
    for (var s = 0; s < series.length; s++) {
      var grad = createEl("linearGradient", {
        id: "fwcg-" + id + "-" + s,
        x1: "0", y1: "0", x2: "0", y2: "1"
      });
      var stop1 = createEl("stop", {
        offset: "0%",
        "stop-color": series[s].color,
        "stop-opacity": fillOpacity[0]
      });
      var stop2 = createEl("stop", {
        offset: "100%",
        "stop-color": series[s].color,
        "stop-opacity": fillOpacity[1]
      });
      grad.appendChild(stop1);
      grad.appendChild(stop2);
      defs.appendChild(grad);
    }

    function render() {
      var w = container.clientWidth;
      var h = container.clientHeight;
      if (w === 0 || h === 0) return;

      svg.setAttribute("viewBox", "0 0 " + w + " " + h);
      svg.setAttribute("width", w);
      svg.setAttribute("height", h);

      var plotL = pad.left;
      var plotR = w - pad.right;
      var plotT = pad.top;
      var plotB = h - pad.bottom;
      var plotW = plotR - plotL;
      var plotH = plotB - plotT;
      var n = data.length;
      if (n === 0) {
        gGrid.innerHTML = "";
        gAreas.innerHTML = "";
        gLines.innerHTML = "";
        gYLabels.innerHTML = "";
        gXLabels.innerHTML = "";
        gHover.innerHTML = "";
        svg.innerHTML = "";
        var noDataText = createEl("text", {
          x: w / 2,
          y: h / 2,
          "text-anchor": "middle",
          "dominant-baseline": "middle",
          "class": "fw-chart-no-data"
        });
        noDataText.textContent = opts.noDataText || "No data available";
        svg.appendChild(noDataText);
        return;
      }

      // Update clip rect to match plot area
      clipRect.setAttribute("x", plotL);
      clipRect.setAttribute("y", plotT);
      clipRect.setAttribute("width", plotW);
      clipRect.setAttribute("height", plotH);

      // Clear
      gGrid.innerHTML = "";
      gAreas.innerHTML = "";
      gLines.innerHTML = "";
      gYLabels.innerHTML = "";
      gXLabels.innerHTML = "";
      gHover.innerHTML = "";

      // Y grid lines + labels
      var steps = Math.round((yMax - yMin) / yStep);
      for (var yi = 0; yi <= steps; yi++) {
        var val = yMin + yi * yStep;
        var yy = plotB - (val - yMin) / (yMax - yMin) * plotH;
        var gridLine = createEl("line", {
          x1: plotL, y1: yy, x2: plotR, y2: yy
        });
        gGrid.appendChild(gridLine);
        var label = createEl("text", {
          x: plotL - 8, y: yy + 4, "text-anchor": "end"
        });
        label.textContent = val;
        gYLabels.appendChild(label);
      }

      // Compute x positions
      var xSpacing = n > 1 ? plotW / (n - 1) : 0;
      var xPositions = [];
      for (var xi = 0; xi < n; xi++) {
        xPositions.push(plotL + xi * xSpacing);
      }

      // X labels + vertical grid lines (sample to avoid overlap)
      var maxLabels = Math.max(2, Math.floor(plotW / 60));
      var labelStep = Math.ceil(n / maxLabels);
      for (var xl = 0; xl < n; xl += labelStep) {
        var vLine = createEl("line", {
          x1: xPositions[xl], y1: plotT, x2: xPositions[xl], y2: plotB,
          "stroke-dasharray": "3 3"
        });
        gGrid.appendChild(vLine);

        var xLabel = createEl("text", {
          x: xPositions[xl], y: plotB + 20, "text-anchor": "middle"
        });
        xLabel.textContent = data[xl].label || "";
        gXLabels.appendChild(xLabel);
      }

      // Compute stacked values
      var stacked = [];
      for (var di = 0; di < n; di++) {
        var row = [];
        var cumulative = 0;
        for (var si = 0; si < series.length; si++) {
          var v = (data[di].values && data[di].values[series[si].key]) || 0;
          cumulative += v;
          row.push(cumulative);
        }
        stacked.push(row);
      }

      // Store for hover dot positioning
      _stacked = stacked;
      _plotB = plotB;
      _plotH = plotH;

      // Draw areas + lines (bottom series first = drawn first = behind)
      for (var si2 = 0; si2 < series.length; si2++) {
        var pts = [];
        for (var pi = 0; pi < n; pi++) {
          pts.push({
            x: xPositions[pi],
            y: plotB - (stacked[pi][si2] - yMin) / (yMax - yMin) * plotH
          });
        }

        var basePts = [];
        if (si2 === 0) {
          basePts = [{ x: plotR, y: plotB }, { x: plotL, y: plotB }];
        } else {
          for (var bp = n - 1; bp >= 0; bp--) {
            basePts.push({
              x: xPositions[bp],
              y: plotB - (stacked[bp][si2 - 1] - yMin) / (yMax - yMin) * plotH
            });
          }
        }

        // Smooth line path
        var linePath = catmullRom(pts, tension);

        // Area: line path + base (reversed) + close
        var areaD;
        if (si2 === 0) {
          // Bottom series: close straight to baseline (plotB = y=0)
          areaD = linePath + " L" + plotR + "," + plotB +
            " L" + plotL + "," + plotB + " Z";
        } else {
          // Stacked series: close along the previous series curve
          var basePath = catmullRom(basePts, tension);
          areaD = linePath + " L" + basePts[0].x + "," + basePts[0].y +
            basePath.replace(/^M[^C]*/, "") + " Z";
        }

        var area = createEl("path", {
          d: areaD,
          fill: "url(#fwcg-" + id + "-" + si2 + ")"
        });
        gAreas.appendChild(area);

        var line = createEl("path", {
          d: linePath,
          fill: "none",
          stroke: series[si2].color,
          "stroke-width": 2
        });
        gLines.appendChild(line);
      }

      // Hover columns
      var colW = n > 1 ? plotW / (n - 1) : plotW;
      for (var hi = 0; hi < n; hi++) {
        (function (idx) {
          var cx = xPositions[idx];
          var rx = n > 1 ? cx - colW / 2 : plotL;
          var rw = n > 1 ? colW : plotW;
          var rect = createEl("rect", {
            x: Math.max(plotL, rx),
            y: plotT,
            width: Math.min(rw, plotR - Math.max(plotL, rx)),
            height: plotH,
            fill: "transparent"
          });
          rect.addEventListener("mouseenter", function () {
            showTooltip(idx, cx, plotT, plotB, w);
          });
          rect.addEventListener("mousemove", function () {
            showTooltip(idx, cx, plotT, plotB, w);
          });
          rect.addEventListener("mouseleave", function () {
            hideTooltip();
          });
          gHover.appendChild(rect);
        })(hi);
      }

      // Update indicator line attrs (position set on hover)
      indicator.setAttribute("y1", plotT);
      indicator.setAttribute("y2", plotB);
    }

    function showTooltip(idx, cx, plotT, plotB, containerW) {
      indicator.setAttribute("x1", cx);
      indicator.setAttribute("x2", cx);
      indicator.style.opacity = "1";

      // Position hover dots at stacked y-values
      for (var di = 0; di < hoverDots.length; di++) {
        var sy = _plotB - (_stacked[idx][di] - _yMin) / (_yMax - _yMin) * _plotH;
        hoverDots[di].setAttribute("cx", cx);
        hoverDots[di].setAttribute("cy", sy);
        hoverDots[di].style.opacity = "1";
      }

      var html = '<div class="fw-chart-tooltip-title">' + escHtml(data[idx].label || "") + '</div>';
      for (var ti = 0; ti < series.length; ti++) {
        var raw = (data[idx].values && data[idx].values[series[ti].key]) || 0;
        var formatted = tooltipFormat ? tooltipFormat(series[ti].label, raw) : (series[ti].label + ": " + raw);
        html += '<div class="fw-chart-tooltip-row">' +
          '<span class="fw-chart-tooltip-dot" style="background:' + series[ti].color + '"></span>' +
          escHtml(formatted) + '</div>';
      }
      tooltip.innerHTML = html;
      tooltip.style.opacity = "1";

      // Position tooltip
      var tw = tooltip.offsetWidth;
      var th = tooltip.offsetHeight;
      var left = cx + 12;
      if (left + tw > containerW - 4) {
        left = cx - tw - 12;
      }
      var top = plotT;
      if (top + th > container.clientHeight) {
        top = container.clientHeight - th - 4;
      }
      tooltip.style.left = left + "px";
      tooltip.style.top = top + "px";
    }

    function hideTooltip() {
      indicator.style.opacity = "0";
      tooltip.style.opacity = "0";
      for (var di = 0; di < hoverDots.length; di++) {
        hoverDots[di].style.opacity = "0";
      }
    }

    // Initial render
    render();

    // Responsive
    if (typeof ResizeObserver !== "undefined") {
      var ro = new ResizeObserver(function () { render(); });
      ro.observe(container);
    } else {
      var resizeTimer;
      window.addEventListener("resize", function () {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(render, 150);
      });
    }

    return { render: render };
  }

  // --- Helpers ---

  function createEl(tag, attrs) {
    var el = document.createElementNS(ns, tag);
    if (attrs) {
      for (var k in attrs) {
        if (attrs.hasOwnProperty(k)) {
          if (k === "style") {
            el.style.cssText = attrs[k];
          } else {
            el.setAttribute(k, attrs[k]);
          }
        }
      }
    }
    return el;
  }

  function merge(defaults, overrides) {
    var out = {};
    for (var k in defaults) {
      if (defaults.hasOwnProperty(k)) out[k] = defaults[k];
    }
    for (var k2 in overrides) {
      if (overrides.hasOwnProperty(k2)) out[k2] = overrides[k2];
    }
    return out;
  }

  function autoMax(data, series) {
    var max = 0;
    for (var i = 0; i < data.length; i++) {
      var sum = 0;
      for (var s = 0; s < series.length; s++) {
        sum += (data[i].values && data[i].values[series[s].key]) || 0;
      }
      if (sum > max) max = sum;
    }
    return Math.ceil(max / 100) * 100 || 100;
  }

  function niceStep(range) {
    var rough = range / 4;
    var mag = Math.pow(10, Math.floor(Math.log(rough) / Math.LN10));
    var norm = rough / mag;
    if (norm <= 1.5) return mag;
    if (norm <= 3) return 2 * mag;
    if (norm <= 7) return 5 * mag;
    return 10 * mag;
  }

  /**
   * Catmull-Rom to cubic bezier SVG path.
   * tension: 0 = straight lines, 1 = maximum smoothing
   */
  function catmullRom(pts, tension) {
    if (pts.length < 2) return "";
    if (pts.length === 2) {
      return "M" + pts[0].x + "," + pts[0].y + " L" + pts[1].x + "," + pts[1].y;
    }

    var alpha = tension != null ? tension : 0.4;
    var path = "M" + pts[0].x + "," + pts[0].y;

    for (var i = 0; i < pts.length - 1; i++) {
      var p0 = pts[i === 0 ? 0 : i - 1];
      var p1 = pts[i];
      var p2 = pts[i + 1];
      var p3 = pts[i + 2 < pts.length ? i + 2 : pts.length - 1];

      var cp1x = p1.x + (p2.x - p0.x) * alpha / 3;
      var cp1y = p1.y + (p2.y - p0.y) * alpha / 3;
      var cp2x = p2.x - (p3.x - p1.x) * alpha / 3;
      var cp2y = p2.y - (p3.y - p1.y) * alpha / 3;

      path += " C" + cp1x + "," + cp1y + " " + cp2x + "," + cp2y + " " + p2.x + "," + p2.y;
    }

    return path;
  }

  function escHtml(str) {
    var d = document.createElement("div");
    d.appendChild(document.createTextNode(str));
    return d.innerHTML;
  }

  // Expose under $flare.ui namespace
  window.$flare = window.$flare || {};
  window.$flare.ui = window.$flare.ui || {};
  window.$flare.ui.line_chart = { create: create };
})();

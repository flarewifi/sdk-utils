$(document).ready(function () {
  // Initialize search for each .search-bar form (desktop and mobile)
  $(".search-bar").each(function () {
    var $form = $(this);
    var $searchInput = $form.find("input");
    var $resultsDropdown = $form.find(".dropdown-menu");
    var allResults = JSON.parse($form.attr("data-results") || "[]");

    $searchInput
      .on("input", function () {
        var query = $(this).val().trim().toLowerCase();
        $resultsDropdown.empty();

        if (!query) {
          $resultsDropdown.removeClass("show");
          return;
        }

        var matches = [];
        allResults.forEach(function(item) {
          if (!Array.isArray(item.Keywords)) return;
          var bestScore = 0;
          var bestKeyword = null;
          item.Keywords.forEach(function(k) {
            var lowerK = k.toLowerCase();
            if (lowerK === query) {
              bestScore = 3;
              bestKeyword = k;
            } else if (lowerK.indexOf(query) === 0) {
              if (bestScore < 2) { bestScore = 2; bestKeyword = k; }
            } else if (lowerK.indexOf(query) !== -1) {
              if (bestScore < 1) { bestScore = 1; bestKeyword = k; }
            }
          });
          if (bestScore > 0) {
            matches.push({item: item, score: bestScore, keyword: bestKeyword});
          }
        });
        matches.sort(function(a, b) { return b.score - a.score; });

        if (matches.length === 0) {
          $resultsDropdown.html(
            '<span class="dropdown-item text-muted">No results found</span>'
          );
        } else {
          var seen = {};
          matches.forEach(function (match, index) {
            var item = match.item;
            var matchedKeyword = match.keyword || item.Label;

            // Deduplicate by RouteUrl + matchedKeyword combination
            var key = item.RouteUrl + '|' + matchedKeyword.toLowerCase();
            if (seen[key]) return;
            seen[key] = true;

            $resultsDropdown.append(
              '<a href="' + item.RouteUrl + '" class="dropdown-item d-flex flex-column ' + (index == 0 ? 'active' : '') + ' ">' +
              '<span class="fw-medium">' + item.Label + '</span>' +
              '<small class="text-muted">' + matchedKeyword + '</small>' +
              '</a>'
            );
          });
        }

        $resultsDropdown.addClass("show");
      })
      // Navigate on 'Enter'
      .on("keydown", function (e) {
        if (e.which === 13 || e.code === 'Enter') {
          e.preventDefault();

          window.location.href = $(".dropdown-item.active").attr("href")
        }
      })
      .on("keydown", function (e) {
        // Select from results using arrow keys
        if (e.which === 38) {
          e.preventDefault();
          var active = $(".dropdown-item.active")
          if (active.prev().is(".dropdown-item")) {
            active.toggleClass("active");
            active.prev().toggleClass("active")
          }
        } else if (e.which === 40) {
          e.preventDefault();
          var active = $(".dropdown-item.active")
          if (active.next().is(".dropdown-item")) {
            active.toggleClass("active");
            active.next().toggleClass("active")
          }
        }
      });


  });

  // Close dropdown when clicking outside any search bar
  $(document).on("click", function (e) {
    $(".search-bar").each(function () {
      var $form = $(this);
      var $resultsDropdown = $form.find(".dropdown-menu");
      var $searchInput = $form.find("input");

      if (!$resultsDropdown.is(e.target) && !$searchInput.is(e.target) && !$(e.target).closest($form).length) {
        $resultsDropdown.removeClass("show");
      }
    });
  });

  // Mobile sidebar behaviors
  var $mobileSidebar = $(".sidebar");
  var $mobileSearchInput = $(".mobile-search-input");
  var $mobileSearchResults = $(".mobile-search-results");

  // Close sidebar when clicking a search result in mobile
  $(document).on("click", ".mobile-search-results .dropdown-item", function () {
    // Use Alpine.js to close the sidebar
    var layoutEl = document.querySelector(".layout");
    if (layoutEl && layoutEl.__x) {
      layoutEl.__x.$data.sidebarOpen = false;
    }
  });

  // Clear search when sidebar closes
  // Watch for sidebar class changes using MutationObserver
  if ($mobileSidebar.length) {
    var observer = new MutationObserver(function (mutations) {
      mutations.forEach(function (mutation) {
        if (mutation.attributeName === "class") {
          var isOpen = $mobileSidebar.hasClass("open");
          if (!isOpen) {
            // Sidebar just closed - clear search
            $mobileSearchInput.val("");
            $mobileSearchResults.empty().removeClass("show");
          }
        }
      });
    });

    observer.observe($mobileSidebar[0], { attributes: true });
  }
});

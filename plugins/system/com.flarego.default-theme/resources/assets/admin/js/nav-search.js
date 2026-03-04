$(document).ready(function () {
  // Initialize search for each .search-bar form (desktop and mobile)
  $(".search-bar").each(function() {
    var $form = $(this);
    var $searchInput = $form.find("input");
    var $resultsDropdown = $form.find(".dropdown-menu");
    var allResults = JSON.parse($form.attr("data-results") || "[]");

    $searchInput.on("input", function () {
      var query = $(this).val().trim().toLowerCase();
      $resultsDropdown.empty();

      if (!query) {
        $resultsDropdown.removeClass("show");
        return;
      }

      var matches = allResults.filter(function(item) {
        if (!Array.isArray(item.Keywords)) return false;
        return item.Keywords.some(function(k) {
          return k.toLowerCase().indexOf(query) !== -1;
        });
      });

      if (matches.length === 0) {
        $resultsDropdown.html(
          '<span class="dropdown-item text-muted">No results found</span>'
        );
      } else {
        var seen = {};
        matches.forEach(function(item) {
          // Find the first keyword that matched the query
          var matchedKeyword = item.Keywords.find(function(k) {
            return k.toLowerCase().indexOf(query) !== -1;
          }) || item.Label;

          // Deduplicate by RouteUrl + matchedKeyword combination
          var key = item.RouteUrl + '|' + matchedKeyword.toLowerCase();
          if (seen[key]) return;
          seen[key] = true;

          $resultsDropdown.append(
            '<a href="' + item.RouteUrl + '" class="dropdown-item d-flex flex-column">' +
            '<span class="fw-medium">' + item.Label + '</span>' +
            '<small class="text-muted">' + matchedKeyword + '</small>' +
            '</a>'
          );
        });
      }

      $resultsDropdown.addClass("show");
    });
  });

  // Close dropdown when clicking outside any search bar
  $(document).on("click", function (e) {
    $(".search-bar").each(function() {
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
  $(document).on("click", ".mobile-search-results .dropdown-item", function() {
    // Use Alpine.js to close the sidebar
    var layoutEl = document.querySelector(".layout");
    if (layoutEl && layoutEl.__x) {
      layoutEl.__x.$data.sidebarOpen = false;
    }
  });

  // Clear search when sidebar closes
  // Watch for sidebar class changes using MutationObserver
  if ($mobileSidebar.length) {
    var observer = new MutationObserver(function(mutations) {
      mutations.forEach(function(mutation) {
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

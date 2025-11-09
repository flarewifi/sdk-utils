$(document).ready(function () {
  var $form = $("#navbarSearch");
  var $searchInput = $("#searchInput");
  var $resultsDropdown = $("#searchResults");

  var allResults = JSON.parse($form.attr("data-results") || "[]");

  $searchInput.on("input", function () {
    var query = $(this).val().trim().toLowerCase();
    $resultsDropdown.empty();

    if (!query) {
      $resultsDropdown.removeClass("show");
      return;
    }

    var matches = allResults.filter((item) => {
      if (!Array.isArray(item.Keywords)) return false;
      return item.Keywords.some((k) => k.toLowerCase().includes(query));
    });

    if (matches.length === 0) {
      $resultsDropdown.html(
        '<span class="dropdown-item text-muted">No results found</span>'
      );
    } else {
      matches.forEach((item) => {
        $resultsDropdown.append(`
                <a href="${item.RouteUrl}" class="dropdown-item">
                ${item.Label}
                </a>
            `);
      });
    }

    $resultsDropdown.addClass("show");
  });

  $(document).on("click", function (e) {
    if (!$resultsDropdown.is(e.target) && !$searchInput.is(e.target)) {
      $resultsDropdown.removeClass("show");
    }
  });
});

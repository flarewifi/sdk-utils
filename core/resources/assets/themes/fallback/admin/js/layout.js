var $ = window.jQuery || window.$;
var $window = $(window);
var navSearch = $("#navbarSearch");
var mdClasses = "md-classes align-items-center d-none d-md-flex";
var smClasses = "sm-classes w-100 collapse d-md-none";
var currentClasses = "";

function responsive() {
  if ($window.width() > 768) {
    if (currentClasses !== mdClasses) {
      navSearch.removeClass(smClasses);
      navSearch.addClass(mdClasses);
      currentClasses = mdClasses;
    }
  } else {
    if (currentClasses !== smClasses) {
      navSearch.removeClass(mdClasses);
      navSearch.addClass(smClasses);
      currentClasses = smClasses;
    }
  }
}

$(document).ready(function () {
  $window.on("resize", responsive);
  responsive();

  $("#logoutBtn").on("click", function (e) {
    e.preventDefault();
    var confirmLogout = confirm("Are you sure you want to logout?");
    if (confirmLogout) {
      var logoutUrl = $(this).data("logout-url");
      var form = document.createElement("form");
      form.method = "POST";
      form.action = logoutUrl;
      document.body.appendChild(form);
      form.submit();
    }
  });
});

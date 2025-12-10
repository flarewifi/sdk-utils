/**
 * Portal Success Redirect
 * Redirects to a custom URL after 3 seconds
 * ES5 syntax for maximum browser compatibility
 */
(function() {
  var container = document.querySelector('.redirect-container');
  if (container) {
    var redirectURL = container.getAttribute('data-redirect-url');
    if (redirectURL) {
      setTimeout(function() {
        window.location.href = redirectURL;
      }, 3000);
    }
  }
})();

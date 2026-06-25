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
        // replace(), not href=: this fires automatically (no user gesture), so
        // href= would push a history entry and trip Chrome's history-manipulation
        // intervention. replace() also keeps the back button off this success page.
        window.location.replace(redirectURL);
      }, 1500);
    }
  }
})();

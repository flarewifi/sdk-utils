// ES5 - Portal error page
(function() {
  'use strict';
  
  // Log for debugging
  var errorMsg = document.querySelector('.error-message');
  if (errorMsg) {
    console.error('Portal Error:', errorMsg.textContent);
  }
})();

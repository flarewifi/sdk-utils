(function ($) {
  $(function () {
    $('.payment-option-card').on('click', function (e) {
      e.preventDefault(); // Prevent immediate navigation

      var $clickedCard = $(this);
      var targetUrl = $clickedCard.attr('href');

      // Prevent double clicks - check if already loading
      if (
        $clickedCard.hasClass('loading') ||
        $clickedCard.hasClass('disabled')
      ) {
        return false;
      }

      // Add loading state to clicked card immediately
      $clickedCard.addClass('loading disabled');
      $clickedCard.attr('aria-busy', 'true');

      // Disable all other payment cards
      $('.payment-option-card').not($clickedCard).addClass('disabled');

      // Add small delay to ensure visual feedback renders, then navigate
      setTimeout(function () {
        window.location.href = targetUrl;
      }, 150);

      return false;
    });
  });
})(window.$);

/**
 * Handle cancel purchase form submission
 * Shows confirmation dialog and loading state
 * @param {HTMLFormElement} form - The cancel form element
 * @returns {boolean} - Whether to proceed with submission
 */
function handleCancelSubmit(form) {
  // Get confirmation message from data attribute
  var confirmMessage = form.getAttribute('data-confirm');

  // Show confirmation dialog
  var confirmed = window.confirm(confirmMessage);

  if (!confirmed) {
    return false; // User cancelled, don't submit
  }

  // User confirmed, show loading state
  var submitButton = form.querySelector('button[type="submit"]');

  if (submitButton) {
    // Add loading class
    submitButton.classList.add('loading');
    submitButton.setAttribute('disabled', 'disabled');
    submitButton.setAttribute('aria-busy', 'true');

    // Show spinner, hide label and icon
    var spinner = submitButton.querySelector('.cancel-spinner');
    if (spinner) {
      spinner.style.display = 'inline-block';
    }
  }

  // Disable payment option cards to prevent interaction
  var paymentCards = document.querySelectorAll('.payment-option-card');
  for (var i = 0; i < paymentCards.length; i++) {
    paymentCards[i].classList.add('disabled');
  }

  return true; // Proceed with form submission
}

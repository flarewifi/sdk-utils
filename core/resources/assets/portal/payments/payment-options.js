(function ($) {
  $(function () {
    var paymentCards = document.querySelectorAll('.payment-option-card');

    for (var i = 0; i < paymentCards.length; i++) {
      paymentCards[i].addEventListener('click', function (e) {
        var clickedCard = e.currentTarget;

        // Prevent double clicks - check if already loading
        if (
          clickedCard.classList.contains('loading') ||
          clickedCard.classList.contains('disabled')
        ) {
          e.preventDefault();
          return false;
        }

        // Add loading state to clicked card
        clickedCard.classList.add('loading', 'disabled');
        clickedCard.setAttribute('aria-busy', 'true');

        // Disable all other payment cards
        var allCards = document.querySelectorAll('.payment-option-card');
        for (var j = 0; j < allCards.length; j++) {
          if (allCards[j] !== clickedCard) {
            allCards[j].classList.add('disabled');
          }
        }

        // Allow the default navigation to proceed
      });
    }
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

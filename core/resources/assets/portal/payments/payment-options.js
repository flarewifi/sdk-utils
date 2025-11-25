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

'use strict';

// Client-side UX for the core forgot-password (OTP) flow. Every behavior is keyed
// on data-* attributes, so each block is a no-op on a page that doesn't use it.
// jQuery is a core global; the portal bundle targets ES2017.

jQuery(function () {
  const pad = (n) => (n < 10 ? '0' + n : '' + n);
  const fmt = (sec) => Math.floor(sec / 60) + ':' + pad(sec % 60);

  // Cooldown: count `data-cooldown` seconds down to 0. While active, disable the
  // optional [data-cooldown-btn] and show [data-cooldown-hint] (its
  // [data-cooldown-label] shows m:ss); hide [data-cooldown-idle] until it ends.
  jQuery('[data-cooldown]').each(function () {
    const $wrap = jQuery(this);
    let remaining = parseInt($wrap.attr('data-cooldown'), 10) || 0;
    const $btn = $wrap.find('[data-cooldown-btn]');
    const $hint = $wrap.find('[data-cooldown-hint]');
    const $label = $wrap.find('[data-cooldown-label]');
    const $idle = $wrap.find('[data-cooldown-idle]');

    const render = () => {
      const active = remaining > 0;
      if ($btn.length) { $btn.prop('disabled', active); }
      if ($hint.length) { $hint.toggle(active); }
      if ($label.length) { $label.text(fmt(remaining)); }
      if ($idle.length) { $idle.toggle(!active); }
    };
    render();
    if (remaining > 0) {
      const iv = setInterval(() => {
        remaining -= 1;
        if (remaining <= 0) { remaining = 0; clearInterval(iv); }
        render();
      }, 1000);
    }
  });

  // Loading: on submit, disable the submit button and swap [data-btn-idle] for the
  // spinner label [data-btn-loading]. Guard against re-submitting.
  jQuery('[data-loading-form]').on('submit', function () {
    const $btn = jQuery(this).find('button[type="submit"]');
    if ($btn.prop('disabled')) { return false; }
    $btn.prop('disabled', true);
    jQuery(this).find('[data-btn-idle]').hide();
    jQuery(this).find('[data-btn-loading]').css('display', 'inline-flex');
  });

  // Password match: disable submit and show [data-match-error] while the two
  // password fields differ, so the mismatch is caught before the round-trip.
  jQuery('[data-match-form]').each(function () {
    const $form = jQuery(this);
    const $pw = $form.find('[data-match-pw]');
    const $pw2 = $form.find('[data-match-pw2]');
    const $err = $form.find('[data-match-error]');
    const $btn = $form.find('button[type="submit"]');
    const check = () => {
      const mismatch = $pw2.val().length > 0 && $pw.val() !== $pw2.val();
      $err.toggle(mismatch);
      $btn.prop('disabled', mismatch);
    };
    $pw.on('input', check);
    $pw2.on('input', check);
  });
});

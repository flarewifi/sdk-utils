jQuery(document).ready(function () {
    var form = jQuery('#login-form');
    var btn = form.find('button[type="submit"]');
    var btnText = btn.find('.fw-btn-text');
    var btnSpinner = btn.find('.fw-btn-spinner');

    form.on('submit', function () {
        btn.prop('disabled', true);
        btnText.text('Signing in...');
        btnSpinner.show();
    });
});

jQuery(document).ready(function () {
    var form = jQuery('#login-form');
    var btn = form.find('button[type="submit"]');
    var btnText = btn.find('.fw-btn-text');
    var btnSpinner = btn.find('.fw-btn-spinner');

    // Submit form when Enter is pressed in any input field
    form.find('input').on('keydown', function (e) {
        if (e.key === 'Enter' || e.keyCode === 13) {
            e.preventDefault();
            form.submit();
        }
    });

    form.on('submit', function () {
        btn.prop('disabled', true);
        btnText.text('Signing in...');
        btnSpinner.show();
    });
});

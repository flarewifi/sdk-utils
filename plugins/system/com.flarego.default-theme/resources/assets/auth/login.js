jQuery(document).ready(function() {
    function centerForm() {
        var windowHeight = jQuery(window).height();
        var formRow = jQuery('.container .row');
        var formColumn = formRow.find('[class*="col-"]');
        
        formColumn.css('margin-top', '0');
        
        var formHeight = formRow.outerHeight();
        var topMargin = (windowHeight / 2) - (formHeight / 2);
        
        if (topMargin > 0) {
            formRow.css('margin-top', topMargin + 'px');
        }
    }
    
    centerForm();
    
    jQuery(window).resize(function() {
        centerForm();
    });
});

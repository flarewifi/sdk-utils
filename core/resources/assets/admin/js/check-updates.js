(function() {
    /**
     * Format file size to human-readable format (B, KB, MB)
     * @param {number} bytes - File size in bytes
     * @returns {string} Formatted file size
     */
    function formatFileSize(bytes) {
        if (bytes < 1024) {
            return bytes + ' B';
        } else if (bytes < 1048576) {
            return (bytes / 1024).toFixed(1) + ' KB';
        } else {
            return (bytes / 1048576).toFixed(1) + ' MB';
        }
    }

    /**
     * Handle firmware file selection with confirmation dialog
     * @param {HTMLInputElement} input - The file input element
     */
    function handleFirmwareFileSelect(input) {
        if (!input.files || !input.files[0]) {
            return;
        }

        var file = input.files[0];
        var size = formatFileSize(file.size);
        var confirmMessage = 'Upload firmware?\n\nFile: ' + file.name + '\nSize: ' + size;

        if (confirm(confirmMessage)) {
            var filenameDisplay = document.getElementById('selected-filename');
            if (filenameDisplay) {
                filenameDisplay.textContent = file.name;
            }
            input.form.requestSubmit();
        } else {
            input.value = '';
        }
    }

    /**
     * Initialize event listeners when DOM is ready
     */
    function init() {
        var fileInput = document.getElementById('sysupgrade_file_inline');
        if (fileInput) {
            fileInput.addEventListener('change', function() {
                handleFirmwareFileSelect(this);
            });
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();

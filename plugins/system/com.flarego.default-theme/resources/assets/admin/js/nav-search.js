window.addEventListener('DOMContentLoaded', function () {
    var searchInput = document.getElementById('searchInput');
    var resultsDropdown = document.getElementById('searchResults');
    var apiUrl = searchInput.getAttribute('data-url');
    var debounceTimeout = null;

    searchInput.addEventListener('input', function () {
        var query = this.value.replace(/^\s+|\s+$/g, '');
        clearTimeout(debounceTimeout);

        if (!query) {
            resultsDropdown.classList.remove('show');
            resultsDropdown.innerHTML = '';
            return;
        }

        debounceTimeout = setTimeout(function () {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', apiUrl + '?q=' + encodeURIComponent(query), true);
            xhr.withCredentials = true;
            xhr.onreadystatechange = function () {
                if (xhr.readyState === 4) {
                if (xhr.status === 200) {
                    try {
                        var data = JSON.parse(xhr.responseText);
                        var results = data.result || [];
                        var html = '';

                        if (results.length === 0) {
                            html = '<span class="dropdown-item text-muted">No results found</span>';
                        } else {
                            for (var i = 0; i < results.length; i++) {
                            var item = results[i];
                            html += '<a href="' + item.RouteUrl + '" class="dropdown-item' +
                                (item.IsCurrent ? ' active' : '') + '">' +
                                item.Label + '</a>';
                            }
                        }

                        resultsDropdown.innerHTML = html;
                        resultsDropdown.classList.add('show');
                    } catch (e) {
                        console.error('Invalid JSON response:', xhr.responseText);
                    }
                } else {
                    console.error('Search request failed:', xhr.status);
                }
                }
            };
            xhr.send();
            }, 300);
        });

        document.addEventListener('click', function (e) {
            if (!resultsDropdown.contains(e.target) && e.target !== searchInput) {
                resultsDropdown.classList.remove('show');
            }
        });
    });

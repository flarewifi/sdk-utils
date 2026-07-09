(function() {
    // RebootCtrl sleeps ~3s before actually executing the reboot/sysupgrade command,
    // so the app server is still the OLD (pre-reboot) process for a while after this
    // page renders — a sysupgrade flash in particular can keep that old process alive
    // for a long, unpredictable time before the device actually goes down. Guessing a
    // fixed delay long enough to clear every case isn't reliable, so instead of timing
    // this, DETECT it: only redirect after observing a genuine down-then-up transition
    // (at least one failed check, followed by a success). A run of early successes
    // against the still-alive old process just keeps polling, never redirecting.
    var INITIAL_DELAY_MS = 3000;
    var POLL_INTERVAL_MS = 2000;
    // Per-attempt cap so a request that hangs (e.g. the device's network interfaces
    // bouncing mid-reboot) doesn't stall the whole polling loop until it times out.
    var FETCH_TIMEOUT_MS = 2000;

    /**
     * Poll statusUrl until it goes from unreachable to reachable again (any HTTP
     * status — this only checks that the server is accepting connections), then
     * navigate to redirectUrl. A success before any observed failure is assumed to be
     * the still-running pre-reboot process and does not trigger a redirect.
     * @param {string} statusUrl
     * @param {string} redirectUrl
     */
    function pollUntilReady(statusUrl, redirectUrl) {
        var sawFailure = false;

        function attempt() {
            var controller = new AbortController();
            var timeoutId = setTimeout(function() { controller.abort(); }, FETCH_TIMEOUT_MS);

            fetch(statusUrl, { cache: 'no-store', signal: controller.signal })
                .then(function() {
                    clearTimeout(timeoutId);
                    if (sawFailure) {
                        window.location.href = redirectUrl;
                        return;
                    }
                    setTimeout(attempt, POLL_INTERVAL_MS);
                })
                .catch(function() {
                    clearTimeout(timeoutId);
                    sawFailure = true;
                    setTimeout(attempt, POLL_INTERVAL_MS);
                });
        }

        setTimeout(attempt, INITIAL_DELAY_MS);
    }

    /**
     * Start polling for the reboot marker element within root, if present and not
     * already started. Safe to call repeatedly — each swapped-in element only ever
     * starts its own poll loop once.
     * @param {ParentNode} root
     */
    function init(root) {
        var el = root.querySelector('[data-reboot-status-url]');
        if (!el || el.dataset.rebootPollStarted) {
            return;
        }
        el.dataset.rebootPollStarted = 'true';
        pollUntilReady(el.dataset.rebootStatusUrl, el.dataset.rebootRedirectUrl);
    }

    // The reboot marker is always swapped in via htmx (the reboot button's response),
    // never present on first page load, so htmx:afterSettle is the only trigger that
    // matters — this still runs the initial scan for safety (e.g. a page reload while
    // a poll was in flight, which itself indicates the server is already back up).
    document.body.addEventListener('htmx:afterSettle', function(evt) {
        init(evt.target);
    });

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function() { init(document); });
    } else {
        init(document);
    }
})();

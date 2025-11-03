function loadNotifications(notifUrl) {
    $.getJSON(notifUrl, function(data) {
        const notifications = data.notifications || [];
        const $list = $("#notifItems");
        const count = notifications.length;

        $("#notifBellCount").text(count).toggle(count > 0);
        $("#notifCountBadge").text(count);

        $list.empty();

        if (count === 0) {
            $list.append(`<p class="dropdown-item text-muted">No new notifications</p>`);
            return;
        }

        notifications.forEach(n => {
            $list.append(`
                <li>
                    <a class="dropdown-item" href="#">
                        🔔 ${n.subject}<br>
                        <small class="text-muted">${new Date(n.created_at).toLocaleString()}</small>
                    </a>
                </li>
            `);
        });
    }).fail(() => console.error("Failed to load notifications"));
}

$(document).ready(function () {
    const $dropdown = $("#notifDropdown");
    const notifUrl = $dropdown.data("notif-url");

    if (!notifUrl) {
        console.error("Notification URL not found (missing data-notif-url)");
        return;
    }

    loadNotifications(notifUrl);
    $dropdown.on("click", function () {
        loadNotifications(notifUrl);
    });
});

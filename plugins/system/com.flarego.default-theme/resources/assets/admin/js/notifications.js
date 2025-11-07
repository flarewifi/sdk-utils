window.loadNotifications = function()  {
    const $dropdown = $("#notifDropdown");
    const getNotifsUrl = $dropdown.data("notif-url");

    $.getJSON(getNotifsUrl, function(data) {
        const notifications = data.notifications || [];
        const $list = $("#notifDropdownMenu");
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
                <li class="d-flex flex-column p-2 border-bottom">
                    <a
                        class="dropdown-item text-wrap notif-item text-center"
                        href="#"
                        data-id="${n.id}"
                        data-subject="${n.subject}"
                        data-content="${n.content}"
                        data-date="${n.created_at}"
                        data-bs-toggle="modal"
                        data-bs-target="#notifModal"
                    >
                        🔔 ${n.subject}<br>
                        <small class="text-muted text-end mb-2">${new Date(n.created_at).toLocaleString()}</small>
                    </a>

                    <button 
                        class="btn btn-sm btn-link text-decoration-none text-primary mark-read-btn align-self-end"
                        data-id="${n.id}"
                    >
                        Mark Read
                    </button>
                </li>
            `);
        });
    }).fail(() => console.error("Failed to load notifications"));
}

$(document).ready(function () {
    const $dropdown = $("#notifDropdown");

    loadNotifications();
    $dropdown.on("click", function () {
        loadNotifications()
    });
});

$(document).on("click", ".mark-read-btn", function (e) {
    e.preventDefault();

    const id = $(this).data("id");
    const $dropdown = $("#notifDropdown");
    const updateNotifURL = $dropdown.data("notif-update-url"); 
    console.log("marking as read...")

    $.ajax({
        url: updateNotifURL,
        method: "POST",
        contentType: "application/json",
        data: JSON.stringify({ id: id, status: 1 }), 
        success: function() {
            loadNotifications();
        },
        error: function() {
            console.error("Failed to mark notification as read");
        }
    });
});


$(document).on("click", ".notif-item", function (e) {
    const subject = $(this).data("subject");
    const content = $(this).data("content");
    const date = new Date($(this).data("date")).toLocaleString();

    const id = $(this).data("id");
    const $dropdown = $("#notifDropdown");
    const updateNotifURL = $dropdown.data("notif-update-url"); 

    // Mark as read
    $.ajax({
        url: updateNotifURL,
        method: "POST",
        contentType: "application/json",
        data: JSON.stringify({ id: id, status: 1 }),
        success: function() {
            loadNotifications();
        },
        error: function() {
            console.error("Failed to mark notification as read");
        }
    });

    $("#notifModalTitle").text(subject);
    $("#notifModalContent").html(content.replace(/\n/g, "<br>"));
    $("#notifModalDate").text(date);
});

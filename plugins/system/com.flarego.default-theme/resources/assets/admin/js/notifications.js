window.loadNotifications = function() {
    var $dropdown = $("#notifDropdown");
    var getNotifsUrl = $dropdown.data("notif-url");

    $.getJSON(getNotifsUrl, function(data) {
        var notifications = data.notifications || [];
        var $list = $("#notifDropdownMenu");
        var count = notifications.length;

        $("#notifBellCount").text(count).toggle(count > 0);
        $("#notifCountBadge").text(count);

        $list.empty();

        if (count === 0) {
            $list.append('<p class="dropdown-item text-muted">No new notifications</p>');
            return;
        }

        notifications.forEach(function(n) {
            var notifHTML = [
                '<li class="d-flex flex-column p-2 border-bottom">',
                    '<a class="dropdown-item text-wrap notif-item text-center" href="#" ',
                        'data-id="' + n.id + '" ',
                        'data-subject="' + n.subject + '" ',
                        'data-content="' + n.content + '" ',
                        'data-date="' + n.created_at + '" ',
                        'data-bs-toggle="modal" data-bs-target="#notifModal">',
                        '🔔 ' + n.subject + '<br>',
                        '<small class="text-muted text-end mb-2">' + new Date(n.created_at).toLocaleString() + '</small>',
                    '</a>',
                    '<button class="btn btn-sm btn-link text-decoration-none text-primary mark-read-btn align-self-end" ',
                        'data-id="' + n.id + '">',
                        'Mark Read',
                    '</button>',
                '</li>'
            ].join('');

            $list.append(notifHTML);
        });
    }).fail(function() { 
        console.error("Failed to load notifications"); 
    });
};

$(document).ready(function () {
    var $dropdown = $("#notifDropdown");

    loadNotifications();
    $dropdown.on("click", function () {
        loadNotifications()
    });
});

$(document).on("click", ".mark-read-btn", function (e) {
    e.preventDefault();

    var id = $(this).data("id");
    var $dropdown = $("#notifDropdown");
    var updateNotifURL = $dropdown.data("notif-update-url"); 
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
    var subject = $(this).data("subject");
    var content = $(this).data("content");
    var date = new Date($(this).data("date")).toLocaleString();

    var id = $(this).data("id");
    var $dropdown = $("#notifDropdown");
    var updateNotifURL = $dropdown.data("notif-update-url"); 

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

-- name: GetRevenueChartLast7Days :many
SELECT
    days.day,
    COALESCE(c.coinslot_revenue, 0) AS coinslot_revenue,
    COALESCE(vr.voucher_revenue, 0) AS voucher_revenue
FROM (
    SELECT DATE('now', '-6 days') AS day UNION ALL
    SELECT DATE('now', '-5 days') UNION ALL
    SELECT DATE('now', '-4 days') UNION ALL
    SELECT DATE('now', '-3 days') UNION ALL
    SELECT DATE('now', '-2 days') UNION ALL
    SELECT DATE('now', '-1 days') UNION ALL
    SELECT DATE('now')
) AS days
LEFT JOIN (
    SELECT
        DATE(p.confirmed_at) AS day,
        SUM(CASE WHEN py.provider = 'com.flarego.wireless-coinslot' OR py.provider = 'com.flarego.wired-coinslot' THEN py.amount ELSE 0 END) AS coinslot_revenue
    FROM purchases p
    LEFT JOIN payments py ON py.purchase_id = p.id
    WHERE p.confirmed_at IS NOT NULL
      AND p.confirmed_at >= DATE('now', '-6 days', 'start of day')
      AND p.confirmed_at < DATE('now', '+1 day', 'start of day')
    GROUP BY DATE(p.confirmed_at)
) AS c ON c.day = days.day
LEFT JOIN (
    SELECT
        DATE(v.created_at) AS day,
        SUM(vb.amount) AS voucher_revenue
    FROM vouchers v
    LEFT JOIN voucher_batches vb ON vb.uuid = v.batch_uuid
    WHERE v.created_at IS NOT NULL
      AND v.created_at >= DATE('now', '-6 days', 'start of day')
      AND v.created_at < DATE('now', '+1 day', 'start of day')
    GROUP BY DATE(v.created_at)
) AS vr ON vr.day = days.day
ORDER BY days.day ASC;

-- name: GetDashboardSalesSummary :one
SELECT
    COALESCE(SUM(py.amount), 0) + COALESCE((
        SELECT SUM(vb.amount)
        FROM vouchers v
        LEFT JOIN voucher_batches vb ON vb.uuid = v.batch_uuid
        WHERE v.created_at IS NOT NULL
          AND v.created_at >= DATE('now', 'start of day')
          AND v.created_at < DATE('now', '+1 day', 'start of day')
    ), 0) AS total_revenue,
    COALESCE(SUM(CASE WHEN py.provider = 'com.flarego.wireless-coinslot' OR py.provider = 'com.flarego.wired-coinslot' THEN py.amount ELSE 0 END), 0) AS coinslot_revenue,
    COALESCE((
        SELECT SUM(vb.amount)
        FROM vouchers v
        LEFT JOIN voucher_batches vb ON vb.uuid = v.batch_uuid
        WHERE v.created_at IS NOT NULL
          AND v.created_at >= DATE('now', 'start of day')
          AND v.created_at < DATE('now', '+1 day', 'start of day')
    ), 0) AS voucher_revenue
FROM purchases p
LEFT JOIN payments py ON py.purchase_id = p.id
WHERE p.confirmed_at IS NOT NULL
  AND p.confirmed_at >= DATE('now', 'start of day')
  AND p.confirmed_at < DATE('now', '+1 day', 'start of day');

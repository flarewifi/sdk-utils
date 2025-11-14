-- name: UpsertQuickAccessNav :exec
INSERT INTO quick_access_navs (
  plugin_pkg, route_name, route_params, visit_count, updated_at
)
VALUES
  (
    @plugin_pkg,
    @route_name,
    @route_params,
    1,
    CURRENT_TIMESTAMP
  )
ON CONFLICT(plugin_pkg, route_name, route_params)
DO UPDATE SET
  visit_count = quick_access_navs.visit_count + 1,
  updated_at = CURRENT_TIMESTAMP;


-- name: GetTop3QuickAccessNavs :many
SELECT
  id,
  plugin_pkg,
  route_name,
  route_params,
  visit_count,
  created_at,
  updated_at
FROM
  quick_access_navs
ORDER BY
  visit_count DESC
LIMIT
  3;


-- name: FindQuickAccessNav :one
SELECT
  id,
  plugin_pkg,
  route_name,
  route_params,
  visit_count,
  created_at,
  updated_at
FROM
  quick_access_navs
WHERE
  plugin_pkg = @plugin_pkg
  AND route_name = @route_name
  AND route_params = @route_params
LIMIT
  1;

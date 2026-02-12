CREATE TABLE IF NOT EXISTS quick_access_navs (
    id INTEGER PRIMARY KEY,
    plugin_pkg VARCHAR(255) NOT NULL,
    route_name VARCHAR(255) NOT NULL,
    route_params TEXT NOT NULL DEFAULT '',
    visit_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(plugin_pkg, route_name, route_params)
);

CREATE INDEX IF NOT EXISTS index_visit_count ON quick_access_navs(visit_count DESC);
CREATE INDEX IF NOT EXISTS index_plugin_pkg ON quick_access_navs(plugin_pkg);

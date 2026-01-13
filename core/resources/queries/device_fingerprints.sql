-- name: CreateDeviceFingerprint :one
INSERT INTO device_fingerprints (
    device_id,
    fingerprint_hash,
    user_agent,
    browser_name,
    os_family,
    screen_resolution,
    language,
    timezone,
    is_cna
) VALUES (
    @device_id,
    @fingerprint_hash,
    @user_agent,
    @browser_name,
    @os_family,
    @screen_resolution,
    @language,
    @timezone,
    @is_cna
) RETURNING id;

-- name: FindFingerprintsByDeviceID :many
SELECT
    id,
    device_id,
    fingerprint_hash,
    user_agent,
    browser_name,
    os_family,
    screen_resolution,
    language,
    timezone,
    is_cna,
    created_at,
    last_seen_at
FROM device_fingerprints
WHERE device_id = @device_id
    AND created_at >= datetime('now', '-6 months')
ORDER BY last_seen_at DESC;

-- name: CheckFingerprintExactMatch :one
SELECT
    id,
    device_id,
    fingerprint_hash,
    user_agent,
    browser_name,
    os_family,
    screen_resolution,
    language,
    timezone,
    is_cna,
    created_at,
    last_seen_at
FROM device_fingerprints
WHERE device_id = @device_id
    AND fingerprint_hash = @fingerprint_hash
    AND created_at >= datetime('now', '-6 months')
LIMIT 1;

-- name: UpdateFingerprintLastSeen :exec
UPDATE device_fingerprints
SET last_seen_at = CURRENT_TIMESTAMP
WHERE id = @id;

-- name: DeleteOldFingerprints :exec
DELETE FROM device_fingerprints
WHERE created_at < datetime('now', '-6 months');

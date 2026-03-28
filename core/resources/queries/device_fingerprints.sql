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
LIMIT 1;

-- name: FindDeviceByFingerprintHash :one
-- Finds a device ID by exact fingerprint hash match (across all devices).
-- Used to identify a returning device when cookie and MAC lookup both fail.
-- Returns the most recently seen match.
SELECT device_id
FROM device_fingerprints
WHERE fingerprint_hash = @fingerprint_hash AND is_cna = FALSE
ORDER BY last_seen_at DESC
LIMIT 1;

-- name: UpdateFingerprintLastSeen :exec
UPDATE device_fingerprints
SET last_seen_at = datetime('now')
WHERE id = @id;

-- name: DeleteOldFingerprints :exec
-- DEPRECATED: Use DeleteExcessFingerprintsForDevice instead
DELETE FROM device_fingerprints
WHERE created_at < datetime('now', '-6 months');

-- name: GetDevicesWithExcessFingerprints :many
-- Returns device IDs that have more than 10 fingerprints (max allowed per device)
SELECT device_id, COUNT(*) as fingerprint_count
FROM device_fingerprints
GROUP BY device_id
HAVING COUNT(*) > 10;

-- name: DeleteExcessFingerprintsForDevice :exec
-- Keeps only the 10 most recent fingerprints for a device (by last_seen_at)
-- Deletes older fingerprints beyond the limit
DELETE FROM device_fingerprints WHERE id IN (
    SELECT df_old.id FROM device_fingerprints df_old
    WHERE df_old.device_id = @device_id
      AND df_old.id NOT IN (
        SELECT df_keep.id FROM device_fingerprints df_keep
        WHERE df_keep.device_id = @device_id
        ORDER BY df_keep.last_seen_at DESC
        LIMIT 10
      )
);

-- name: DeleteAllFingerprints :exec
-- Deletes all device fingerprint records.
DELETE FROM device_fingerprints;

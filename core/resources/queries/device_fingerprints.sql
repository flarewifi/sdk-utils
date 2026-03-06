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

-- name: FindSharedFingerprintHashes :many
-- Finds non-CNA fingerprint hashes that appear on multiple distinct devices since the given UTC timestamp.
-- Used by the merge job to detect duplicate device records for the same physical device.
-- Caller should pass time.Now().UTC().AddDate(0, 0, -30) for 30-day lookback.
SELECT fingerprint_hash
FROM device_fingerprints
WHERE last_seen_at >= @since_utc
  AND fingerprint_hash != ''
  AND is_cna = FALSE
GROUP BY fingerprint_hash
HAVING COUNT(DISTINCT device_id) > 1;

-- name: FindDeviceIDsByFingerprintHash :many
-- Returns all device IDs that have a given non-CNA fingerprint hash since the given UTC timestamp,
-- ordered by most recently seen. Used by the merge job.
-- Caller should pass time.Now().UTC().AddDate(0, 0, -30) for 30-day lookback.
SELECT DISTINCT df.device_id
FROM device_fingerprints df
WHERE df.fingerprint_hash = @fingerprint_hash
  AND df.is_cna = FALSE
  AND df.last_seen_at >= @since_utc
ORDER BY df.last_seen_at DESC;

-- name: FindDeviceByFingerprintHash :one
-- Finds the most recently seen device with a given non-CNA fingerprint hash.
-- Used as a MAC-address fallback in portal registration when ARP lookup fails.
SELECT df.device_id
FROM device_fingerprints df
WHERE df.fingerprint_hash = @fingerprint_hash
  AND df.is_cna = FALSE
ORDER BY df.last_seen_at DESC
LIMIT 1;

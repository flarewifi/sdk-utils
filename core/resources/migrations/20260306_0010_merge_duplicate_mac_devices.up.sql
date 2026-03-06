-- Migration: Merge duplicate devices with the same current MAC address
-- This migration identifies devices that share the same MAC address (where is_current = TRUE)
-- and merges them, keeping the oldest device (earliest created_at) as the primary.

-- =============================================================================
-- STEP 1: Create temp table to identify duplicates and determine which device to keep
-- =============================================================================

CREATE TEMP TABLE IF NOT EXISTS duplicate_mac_devices AS
WITH duplicate_macs AS (
    -- Find MAC addresses that have multiple devices claiming them as current
    SELECT mac_address
    FROM device_macs
    WHERE is_current = TRUE
    GROUP BY mac_address
    HAVING COUNT(DISTINCT device_id) > 1
),
devices_with_dup_macs AS (
    -- Get all devices that have a duplicate MAC, with their created_at
    SELECT 
        dm.device_id,
        dm.mac_address,
        d.created_at,
        ROW_NUMBER() OVER (PARTITION BY dm.mac_address ORDER BY d.created_at ASC) as rn
    FROM device_macs dm
    INNER JOIN devices d ON dm.device_id = d.id
    INNER JOIN duplicate_macs dup ON dm.mac_address = dup.mac_address
    WHERE dm.is_current = TRUE
)
SELECT 
    mac_address,
    device_id,
    created_at,
    rn,
    -- Device with rn=1 is the oldest (keeper), others will be merged into it
    CASE WHEN rn = 1 THEN device_id ELSE NULL END as keeper_device_id,
    CASE WHEN rn > 1 THEN device_id ELSE NULL END as source_device_id
FROM devices_with_dup_macs;

-- =============================================================================
-- STEP 2: For each duplicate group, get the target device ID (oldest device to keep)
-- =============================================================================

CREATE TEMP TABLE IF NOT EXISTS merge_targets AS
SELECT 
    src.mac_address,
    src.source_device_id,
    keeper.device_id as target_device_id
FROM duplicate_mac_devices src
INNER JOIN duplicate_mac_devices keeper 
    ON src.mac_address = keeper.mac_address 
    AND keeper.rn = 1
WHERE src.source_device_id IS NOT NULL;

-- =============================================================================
-- STEP 3: Transfer sessions from source devices to target devices
-- =============================================================================

UPDATE sessions
SET device_id = (
    SELECT target_device_id 
    FROM merge_targets 
    WHERE merge_targets.source_device_id = sessions.device_id
)
WHERE device_id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 4: Transfer purchases from source devices to target devices
-- Note: device_id can be NULL for admin purchases, so we only update non-NULL values
-- =============================================================================

UPDATE purchases
SET device_id = (
    SELECT target_device_id 
    FROM merge_targets 
    WHERE merge_targets.source_device_id = purchases.device_id
)
WHERE device_id IS NOT NULL 
  AND device_id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 5: Transfer fingerprints from source devices to target devices
-- =============================================================================

UPDATE device_fingerprints
SET device_id = (
    SELECT target_device_id 
    FROM merge_targets 
    WHERE merge_targets.source_device_id = device_fingerprints.device_id
)
WHERE device_id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 6: Transfer MAC history from source devices to target devices
-- =============================================================================

-- First, update device_id for MACs that don't conflict (unique MACs from source)
UPDATE device_macs
SET device_id = (
    SELECT target_device_id 
    FROM merge_targets 
    WHERE merge_targets.source_device_id = device_macs.device_id
),
is_current = FALSE  -- Mark as non-current since target already has a current MAC
WHERE device_id IN (SELECT source_device_id FROM merge_targets)
AND NOT EXISTS (
    -- Check if target device already has this exact MAC address
    SELECT 1 FROM device_macs dm2 
    WHERE dm2.device_id = (
        SELECT target_device_id 
        FROM merge_targets 
        WHERE merge_targets.source_device_id = device_macs.device_id
    )
    AND dm2.mac_address = device_macs.mac_address
);

-- Delete remaining MAC records from source devices (duplicates that couldn't be transferred)
DELETE FROM device_macs
WHERE device_id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 7: Transfer wallet balance and transactions
-- =============================================================================

-- First, transfer wallet transactions to target wallet
UPDATE wallet_transactions
SET wallet_id = (
    SELECT w_target.id 
    FROM wallets w_source
    INNER JOIN merge_targets mt ON w_source.device_id = mt.source_device_id
    INNER JOIN wallets w_target ON w_target.device_id = mt.target_device_id
    WHERE w_source.id = wallet_transactions.wallet_id
)
WHERE wallet_id IN (
    SELECT w.id FROM wallets w 
    WHERE w.device_id IN (SELECT source_device_id FROM merge_targets)
);

-- Add source wallet balances to target wallets
UPDATE wallets
SET balance = balance + COALESCE((
    SELECT SUM(w_source.balance)
    FROM wallets w_source
    INNER JOIN merge_targets mt ON w_source.device_id = mt.source_device_id
    WHERE mt.target_device_id = wallets.device_id
), 0)
WHERE device_id IN (SELECT target_device_id FROM merge_targets);

-- Delete source wallets (transactions already transferred)
DELETE FROM wallets
WHERE device_id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 8: Delete source devices (all related data has been transferred)
-- =============================================================================

DELETE FROM devices
WHERE id IN (SELECT source_device_id FROM merge_targets);

-- =============================================================================
-- STEP 9: Cleanup temp tables
-- =============================================================================

DROP TABLE IF EXISTS merge_targets;
DROP TABLE IF EXISTS duplicate_mac_devices;

-- Note: VACUUM cannot run inside a transaction. 
-- Disk space reclamation will happen naturally over time or via manual VACUUM.

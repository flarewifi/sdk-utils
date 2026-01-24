#!/usr/bin/env python3
"""
Fix snake_case translation keys to Title Case
"""

import os
import re
import sys

# Map of snake_case to Title Case
REPLACEMENTS = {
    "add_new_rates_success": "Add new rates success",
    "already_connected_to_internet": "Already connected to internet",
    "delete_all_vouchers_success": "Delete all vouchers success",
    "delete_rate_success": "Delete rate success",
    "delete_voucher_success": "Delete voucher success",
    "generate_vouchers_success": "Generate vouchers success",
    "index_out_of_bounds": "Index out of bounds",
    "invalid_amount_value": "Invalid amount value",
    "invalid_count_value": "Invalid count value",
    "invalid_form_values": "Invalid form values",
    "invalid_index": "Invalid index",
    "invalid_rate_index": "Invalid rate index",
    "not_connected_to_internet": "Not connected to internet",
    "now_connected_to_internet": "Now connected to internet",
    "now_disconnected_from_internet": "Now disconnected from internet",
    "pause_setting_add_success": "Pause setting add success",
    "pause_setting_delete_success": "Pause setting delete success",
    "pause_setting_save_success": "Pause setting save success",
    "success_free_trial": "Success free trial",
    "unable_to_delete_vouchers": "Unable to delete vouchers",
    "unable_to_find_rate_to_delete": "Unable to find rate to delete",
    "unable_to_find_rate_to_update": "Unable to find rate to update",
    "unable_to_get_device": "Unable to get device",
    "unable_to_get_rate_settings": "Unable to get rate settings",
    "unable_to_get_speed_value": "Unable to get speed value",
    "unable_to_get_voucher_count_value": "Unable to get voucher count value",
    "unable_to_get_voucher_validity_unit_value": "Unable to get voucher validity unit value",
    "unable_to_get_voucher_validity_value": "Unable to get voucher validity value",
    "unable_to_save_pause_settings": "Unable to save pause settings",
    "unable_to_update_wifi_rate_settings": "Unable to update wifi rate settings",
    "update_rates_success": "Update rates success",
    "update_voucher_success": "Update voucher success",
}


def fix_file(filepath):
    """Fix snake_case keys in a single file"""
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    original_content = content
    changes = []

    for snake_case, title_case in REPLACEMENTS.items():
        pattern = rf'(api\.Translate\([^,]+,\s*")({snake_case})(")'
        if re.search(pattern, content):
            content = re.sub(pattern, rf"\1{title_case}\3", content)
            changes.append(f"  {snake_case} → {title_case}")

    if content != original_content:
        with open(filepath, "w", encoding="utf-8") as f:
            f.write(content)
        print(f"✓ Fixed {filepath}")
        for change in changes:
            print(change)
        return True
    return False


def main():
    base_dir = "data/plugins/local/com.flarego.wifi-hotspot/app/controllers"

    if not os.path.exists(base_dir):
        print(f"Error: Directory not found: {base_dir}")
        sys.exit(1)

    fixed_count = 0

    for root, dirs, files in os.walk(base_dir):
        for file in files:
            if file.endswith(".go"):
                filepath = os.path.join(root, file)
                if fix_file(filepath):
                    fixed_count += 1

    print(f"\n✅ Fixed {fixed_count} files")


if __name__ == "__main__":
    main()

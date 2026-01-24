#!/usr/bin/env python3
"""
FlareHotspot Translation Batch Update Tool

Safely update translation files across all languages in one command.
Designed for AI coding agents and bulk translation workflows.

Usage:
  ./scripts/update-translations.py <type> <key> <translations_json>
  ./scripts/update-translations.py --file translations.json

Examples:
  # Direct command line
  ./scripts/update-translations.py warning "The purchase has been cancelled" '{"en":"text","es":"texto"}'

  # Using a JSON file
  ./scripts/update-translations.py --file translations.json
  
  # Plugin translations
  ./scripts/update-translations.py --base-dir data/plugins/local/my-plugin/resources/translations \\
    label "Settings" '{"en":"Settings","es":"Configuración"}'
  
  # Dry run (preview only)
  ./scripts/update-translations.py --dry-run error "Failed" '{"en":"Failed","es":"Falló"}'

Features:
  - Batch update all languages at once
  - UTF-8 encoding validation
  - Backup creation
  - Dry-run mode for preview
  - Works from any directory
  - AI agent friendly (no file read requirements)
"""

import argparse
import json
import os
import shutil
import sys
from pathlib import Path


def validate_content(text, lang):
    """Validate translation content for common issues."""
    warnings = []

    # Check for null bytes
    if "\0" in text:
        raise ValueError(f"Content for {lang} contains null bytes (invalid UTF-8)")

    # Check for empty content
    if not text or not text.strip():
        warnings.append(f"⚠️  {lang}: Empty or whitespace-only content")

    # Check for leading/trailing whitespace
    if text != text.strip():
        warnings.append(
            f"ℹ️  {lang}: Has leading/trailing whitespace (will be preserved)"
        )

    # Check for double spaces
    if "  " in text:
        warnings.append(f"ℹ️  {lang}: Contains double spaces")

    return warnings


def create_backup(file_path):
    """Create a backup of existing file."""
    if not file_path.exists():
        return None

    backup_path = file_path.with_suffix(".txt.bak")
    try:
        shutil.copy2(file_path, backup_path)
        return str(backup_path)
    except Exception as e:
        print(f"⚠️  Warning: Could not create backup: {e}", file=sys.stderr)
        return None


def update_translations(
    trans_type,
    key,
    translations,
    base_dir="core/resources/translations",
    dry_run=False,
    create_backups=False,
    verbose=False,
):
    """Update translation files for all languages."""
    created = []
    updated = []
    errors = []
    all_warnings = []
    backups = []

    for lang, text in translations.items():
        try:
            # Validate content
            content_warnings = validate_content(text, lang)
            all_warnings.extend(content_warnings)

            # Create directory if it doesn't exist
            dir_path = Path(base_dir) / lang / trans_type

            if not dry_run:
                dir_path.mkdir(parents=True, exist_ok=True)

            # File path
            file_path = dir_path / f"{key}.txt"

            # Check if file exists
            file_exists = file_path.exists()

            if dry_run:
                # Just report what would happen
                if file_exists:
                    old_content = file_path.read_text(encoding="utf-8")
                    if verbose:
                        updated.append(
                            f"{file_path}\n    Old: {repr(old_content)}\n    New: {repr(text)}"
                        )
                    else:
                        updated.append(str(file_path))
                else:
                    if verbose:
                        created.append(f"{file_path}\n    Content: {repr(text)}")
                    else:
                        created.append(str(file_path))
            else:
                # Create backup if requested
                if create_backups and file_exists:
                    backup = create_backup(file_path)
                    if backup:
                        backups.append(backup)

                # Write the translation (with single newline at end)
                with open(file_path, "w", encoding="utf-8") as f:
                    f.write(text)
                    if not text.endswith("\n"):
                        f.write("\n")

                if file_exists:
                    updated.append(str(file_path))
                else:
                    created.append(str(file_path))

        except Exception as e:
            errors.append(f"❌ {lang}: {e}")

    return created, updated, errors, all_warnings, backups


def main():
    parser = argparse.ArgumentParser(
        description="FlareHotspot Translation Batch Update Tool",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Using command line arguments
  %(prog)s warning "The purchase has been cancelled" '{"en":"cancelled","es":"cancelada"}'

  # Using a JSON file
  %(prog)s --file translations.json

  # Dry run to preview changes
  %(prog)s --dry-run label "Settings" '{"en":"Settings","es":"Configuración"}'

  # Create backups before updating
  %(prog)s --backup label "Dashboard" '{"en":"Dashboard","es":"Panel"}'

  # JSON file format:
  {
    "type": "warning",
    "key": "The purchase has been cancelled",
    "translations": {
      "en": "The purchase has been cancelled",
      "es": "La compra ha sido cancelada"
    }
  }

Features:
  - Batch update all languages at once
  - UTF-8 validation and encoding
  - Optional backup creation
  - Dry-run mode for preview
  - Verbose output for debugging
  - Works from any directory
        """,
    )

    parser.add_argument(
        "type",
        nargs="?",
        help="Translation type (error, warning, success, label, etc.)",
    )
    parser.add_argument(
        "key", nargs="?", help="Translation key (filename without .txt)"
    )
    parser.add_argument(
        "translations_json",
        nargs="?",
        help="JSON string with language:translation pairs",
    )
    parser.add_argument("--file", "-f", help="JSON file containing translation data")
    parser.add_argument(
        "--base-dir",
        default="core/resources/translations",
        help="Base translations directory (default: core/resources/translations)",
    )
    parser.add_argument(
        "--dry-run",
        "-n",
        action="store_true",
        help="Preview changes without writing files",
    )
    parser.add_argument(
        "--backup",
        "-b",
        action="store_true",
        help="Create .bak backup files before updating",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Show detailed output including file contents",
    )

    args = parser.parse_args()

    # Parse input
    if args.file:
        # Load from file
        try:
            with open(args.file, "r", encoding="utf-8") as f:
                data = json.load(f)
            trans_type = data["type"]
            key = data["key"]
            translations = data["translations"]
        except FileNotFoundError:
            print(f"❌ Error: File '{args.file}' not found", file=sys.stderr)
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"❌ Error: Invalid JSON in file: {e}", file=sys.stderr)
            sys.exit(1)
        except KeyError as e:
            print(f"❌ Error: Missing required field in JSON: {e}", file=sys.stderr)
            print("Required fields: type, key, translations", file=sys.stderr)
            sys.exit(1)
    else:
        # Use command line arguments
        if not all([args.type, args.key, args.translations_json]):
            parser.print_help()
            sys.exit(1)

        trans_type = args.type
        key = args.key
        try:
            translations = json.loads(args.translations_json)
        except json.JSONDecodeError as e:
            print(
                f"❌ Error: Invalid JSON in translations argument: {e}", file=sys.stderr
            )
            sys.exit(1)

    # Show what we're doing
    mode = "DRY RUN" if args.dry_run else "UPDATE"
    print(f"\n{'=' * 60}")
    print(f"{mode}: {trans_type}/{key}.txt")
    print(f"{'=' * 60}\n")
    print(f"Base directory: {args.base_dir}")
    print(f"Languages: {', '.join(translations.keys())}\n")

    # Update translations
    created, updated, errors, warnings, backups = update_translations(
        trans_type,
        key,
        translations,
        args.base_dir,
        dry_run=args.dry_run,
        create_backups=args.backup,
        verbose=args.verbose,
    )

    # Print results
    if created:
        print(
            f"✅ {'Would create' if args.dry_run else 'Created'} {len(created)} file(s):"
        )
        for path in created:
            print(f"  {path}")
        print()

    if updated:
        print(
            f"✅ {'Would update' if args.dry_run else 'Updated'} {len(updated)} file(s):"
        )
        for path in updated:
            print(f"  {path}")
        print()

    if backups:
        print(f"💾 Created {len(backups)} backup(s):")
        for path in backups:
            print(f"  {path}")
        print()

    if warnings:
        print(f"⚠️  {len(warnings)} warning(s):")
        for warning in warnings:
            print(f"  {warning}")
        print()

    if errors:
        print(f"❌ {len(errors)} error(s):")
        for error in errors:
            print(f"  {error}")
        print()
        sys.exit(1)

    total = len(created) + len(updated)
    if args.dry_run:
        print(f"💡 Dry run complete. {total} file(s) would be modified.")
        print(f"   Run without --dry-run to apply changes.")
    else:
        print(f"✅ Successfully processed {total} translation file(s)!")

    print()


if __name__ == "__main__":
    main()

# PluginInfo

It contains information about the plugin as defined in [plugin.json](./plugin.json.md). The following are the information available in `sdkpkg.PluginInfo`:

## Name

The name of the package.

## Description

The descripotion of the plugin.

## Version

The version of the plugin.

## Package

The package name of the plugin.

## SystemPackages

This is a list of system packages required by the plugin.

## PreInstall

Optional path to a shell script run during installation, after `SystemPackages`
are installed but before the plugin files are copied into place. See
[plugin.json](./plugin.json.md#preinstall).

## PostInstall

Optional path to a shell script run after the plugin is installed. See
[plugin.json](./plugin.json.md#postinstall).

## SDK

This is the minimum SDK version that the plugin supports.

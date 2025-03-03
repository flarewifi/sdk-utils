# The `plugin.json` File

## Fields

The `plugin.json` file contains the metadata of the plugin. It is a JSON file located in the root directory of a plugin. It contains the following fields:

### name
The name of the plugin.

### package
The package name of the plugin. It should be unique and in reverse domain format. Example: `com.mydomain.myplugin`.

### description
The description of the plugin.

### version

The version of the plugin. It should follow the [Semantic Versioning](https://semver.org/) format. Example: `1.0.0`

### system_packages

List of system packages required by the plugin.

### sdk

The minimum sdk version that the plugin supports. The supported SDK versions are available in [flarehotspot/devkit](https://github.com/flarehotspot/devkit/releases) repository.

## Example

Below is an example of a `plugin.json` file:

```json
{
    "name": "My Plugin",
    "package": "com.mydomain.myplugin",
    "description": "This is my plugin description",
    "version": "0.0.1",
    "system_packages": [],
    "sdk": "1.0.0"
}
```

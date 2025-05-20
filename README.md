# CommonRepo

> [!NOTE]
> This project is a working prototype but will not see further development, as
> commonrepo v2 will be developed under <https://github.com/common-repo>.
>
> When that project becomes public, this repository will be archived.
>
> *January 2024*

[![License](https://img.shields.io/github/license/shakefu/commonrepo)](LICENSE)

CommonRepo is a powerful tool for managing multiple inheritance in repository templates. It allows you to create and maintain template repositories that can be inherited and customized by other repositories, similar to how classes work in object-oriented programming.

## Features

- **In memory compositing**: Virutal in-memory filesystem compositing for
  ultra-fast build times and caching efficiency.
- **Multiple Inheritance**: Inherit from multiple template repositories
- **File Filtering**: Include/exclude specific files and directories
- **File Renaming**: Transform file paths during inheritance
- **Template Support**: Process template files with variables
- **Version Control**: Support for specific git refs (tags, branches, commits)
- **Deep Cloning**: Configurable depth for upstream repository inheritance

## Installation

```bash
go install github.com/shakefu/commonrepo@latest
```

## Quick Start

1. Create a `.commonrepo.yml` file in your repository:

```yaml
# Source repository configuration
include:
  - "**/*"  # Include all files
  - ".*"    # Include hidden files
  - ".*/**/*"  # Include hidden directories

exclude:
  - ".git/**/*"  # Exclude git directory
  - "**/*.md"    # Exclude markdown files

template:
  - "templates/**"  # Process files in templates directory

rename:
  - "templates/(.*)": "%[1]s"  # Move templates to root

# Upstream repositories to inherit from
upstream:
  - url: https://github.com/example/template
    ref: v1.0.0
    include: [".*"]
    exclude: [".gitignore"]
    rename: [{".*\\.md": "docs/%[1]s"}]

# Template variables
template-vars:
  project: ${PROJECT_NAME:-myprojectname}
```

2. Run CommonRepo in your repository:

```bash
commonrepo
```

## Configuration

### Source Repository Configuration

The source repository defines which files should be imported into child repositories:

- `include`: List of glob patterns for files to include
- `exclude`: List of glob patterns for files to exclude
- `template`: List of glob patterns for template files
- `rename`: List of rename rules for file paths
- `install`: List of tool installation specifications
- `install-from`: Optional override for installation path
- `install-with`: Optional override for preferred install manager order

### Consumer Repository Configuration

Consumer repositories can define which sources they want to inherit from:

- `upstream`: List of source repositories to inherit from
  - `url`: Repository URL
  - `ref`: Git reference (tag, branch, or commit)
  - `overwrite`: Whether to overwrite existing files
  - `include`: Additional include patterns
  - `exclude`: Additional exclude patterns
  - `rename`: Additional rename rules
- `template-vars`: Template variables for all upstreams

## Examples

See `testdata/fixtures/schema.yml` for a complete example of the configuration schema.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

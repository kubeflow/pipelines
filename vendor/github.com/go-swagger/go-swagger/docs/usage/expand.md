# Expand a swagger spec

The toolkit has a command to expand a swagger specification.

Expanding a specification resolve all `$ref` (remote or local) and replace them by their expanded
content in the main spec document.

### Usage

To expand a specification:

```
Usage:
  swagger [OPTIONS] expand [expand-OPTIONS]

expands the $refs in a swagger document to inline schemas

Application Options:
  -q, --quiet                     silence logs

Help Options:
  -h, --help                      Show this help message

[expand command options]
          --compact               applies to JSON formated specs. When present, doesn't prettify the json
      -o, --output=               the file to write to
          --format=[yaml|json]    the format for the spec document (default: json)
```

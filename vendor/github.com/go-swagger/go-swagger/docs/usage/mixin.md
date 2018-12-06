# Mixin several swagger specs

The toolkit has a command to mixin swagger specification.

Mixin merges several specs into the first (primary) spec given, and issues warnings when conflicts are detected.

### Usage

To mixin several specifications:

```
Usage:
  swagger [OPTIONS] mixin [mixin-OPTIONS]

merge additional specs into first/primary spec by copying their paths and definitions

Application Options:
  -q, --quiet                     silence logs

Help Options:
  -h, --help                      Show this help message

[mixin command options]
      -c=                         expected # of rejected mixin paths, defs, etc due to existing key. Non-zero exit if does not match actual.
          --compact               applies to JSON formated specs. When present, doesn't prettify the json
      -o, --output=               the file to write to
          --format=[yaml|json]    the format for the spec document (default: json)
```

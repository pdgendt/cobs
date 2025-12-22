# Python interopability

## Prerequisites

Make sure `uv` is installed.

## Commands

Example usage:

```sh
echo -n "\x55" | uv run main.py encode | uv run main.py decode | xxd -p
```

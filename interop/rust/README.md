# Python interopability

## Prerequisites

Make sure `cargo` is installed.

## Commands

Example usage:

```sh
echo -n "\x55" | cargo run encode | cargo run decode | xxd -p
```

# /// script
# requires-python = ">=3.13"
# dependencies = [
#   "cobs==1.2.2",
# ]
# ///

import argparse
import sys

from cobs import cobs, cobsr


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["decode", "encode"])
    parser.add_argument("-r", "--reduced", action="store_true")

    args = parser.parse_args()

    if args.command == "decode":
        func = cobsr.decode if args.reduced else cobs.decode
    else:
        func = cobsr.encode if args.reduced else cobs.encode

    sys.stdout.buffer.write(func(sys.stdin.buffer.read()))


if __name__ == "__main__":
    main()

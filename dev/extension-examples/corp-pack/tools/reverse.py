#!/usr/bin/env python3
"""Example command-tool that reverses input text."""

import json
import sys

args = json.load(sys.stdin)
message = args.get("message", "")
print(message[::-1])

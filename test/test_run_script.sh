#!/bin/bash
echo "=== TEST: /run-script ==="
curl -N -X POST http://localhost:8001/run-script \
  -F "script=sh hello.sh"

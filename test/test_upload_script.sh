#!/bin/bash
echo "=== TEST: /upload-script ==="
curl -N -X POST http://localhost:8001/upload-script \
  -F "file=@test.zip" \
  -F "target=./tmp" \
  -F "script=sh hello.sh"

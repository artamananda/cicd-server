#!/bin/bash
echo "=== TEST: /upload-only ==="
curl -N -X POST http://localhost:8001/upload-only \
  -F "file=@test.zip" \
  -F "target=./tmp"

#!/bin/bash

echo "Select target OS:"
select os in linux windows darwin; do
  if [[ -n "$os" ]]; then
    break
  fi
done

echo "Select target architecture:"
select arch in amd64 arm64 arm386; do
  if [[ -n "$arch" ]]; then
    break
  fi
done

# output_name="http-remote-access-${os}-${arch}"
output_name="http-remote-access"
if [ "$os" == "windows" ]; then
  output_name="${output_name}.exe"
fi

if [ -d "../build" ]; then
  echo "Removing existing build directory..."
  rm -rf ../build
else
  echo "Creating build directory..."
  mkdir -p ../build
fi

echo "Building for GOOS=$os and GOARCH=$arch..."
GOOS=$os GOARCH=$arch go build -o "../build/${output_name}" ../main.go

cat > ../build/package.json <<EOF
{
  "apps": [
    {
      "name": "${output_name}",
      "script": "./${output_name}"
    }
  ]
}
EOF

echo "package.json has been generated."

if [ $? -eq 0 ]; then
  echo "Build successful: ../build/${output_name}"
else
  echo "Build failed."
fi
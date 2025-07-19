# 🚀 Alternate Method for Linux Deployment Without SSH Access

This app allows you to deploy your application using GitHub Actions over http without requiring SSH access. The process involves sending your binary file to the server, where the app will receive it and execute the commands you specify.

# How to Build, Install and Run

### Build

```bash
cd cmd
./build_script.sh
cd ..
```

## Run App

### Run Using PM2

```bash
cd build
pm2 start package.json
./
```

### Run Binary File Directly

```bash
cd build
./http-remote-access
./
```

For Example:

```bash
./http-remote-access
```

If you're on Windows:

```bash
http-remote-access.exe
```

# 🌐 API Access

This server exposes three HTTP POST endpoints for uploading and running scripts with streaming logs.

---

## 1. POST `/upload-only`

Upload a ZIP file which will be extracted on the server.

- **Content-Type:** `multipart/form-data`
- **Form fields:**
  - `file` (required): ZIP file to upload and extract
  - `target` (required): target directory to extract the ZIP contents into

### Example using curl:

```bash
curl -X POST http://<your-domain>/upload-only \
  -F "file=@./your-archive.zip" \
  -F "target=/path/to/extract"
```

## 2. POST `/upload-script`

Upload a ZIP file, extract it, then run a shell script.

- **Content-Type:** `multipart/form-data`
- **Form fields:**
  - `file` (required): ZIP file to upload and extract
  - `target` (required): target directory to extract the ZIP contents into
  - `script` (required): shell command/script to run inside the target directory after extraction

### Example using curl:

```bash
curl -X POST http://<your-domain>/upload-only \
  -F "file=@./your-archive.zip" \
  -F "target=/path/to/extract" \
  -F "script=./deploy.sh"
```

## 3. POST `/run-script`

Run a shell script on the server without uploading anything.

- **Content-Type:** `multipart/form-data`
- **Form fields:**
  - `script` (required): shell command/script to execute

### Example using curl:

```bash
curl -X POST http://<your-domain>/upload-only \
  -d "script=./deploy.sh" \
```

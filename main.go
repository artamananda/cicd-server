package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	http.HandleFunc("/upload-only", uploadOnlyHandler)
	http.HandleFunc("/upload-script", uploadWithScriptHandler)
	http.HandleFunc("/run-script", runScriptOnlyHandler)

	log.Println("Server started on :8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// --- Upload ZIP only with streaming log ---
func uploadOnlyHandler(w http.ResponseWriter, r *http.Request) {
	enableStreaming(w)
	flusher := w.(http.Flusher)

	log.Println("[INFO] Upload only handler started")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(500 << 20)
	if err != nil {
		http.Error(w, "[ERR] Error parsing form", http.StatusBadRequest)
		log.Printf("[ERR] Error parsing form: %v", err)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "[ERR] File is required", http.StatusBadRequest)
		log.Printf("[ERR] File is required: %v", err)
		return
	}
	defer file.Close()

	targetDir := r.FormValue("target")
	if targetDir == "" {
		http.Error(w, "[ERR] Target directory is required", http.StatusBadRequest)
		log.Println("[ERR] Target directory is required")
		return
	}

	// Now safe to write response and flush because input is validated
	fmt.Fprintln(w, "[INFO] Starting upload process")
	flusher.Flush()

	fmt.Fprintf(w, "[INFO] Uploading %s to %s\n", fileHeader.Filename, targetDir)
	flusher.Flush()

	err = saveAndExtractZipStream(w, flusher, file, fileHeader.Filename, targetDir)
	if err != nil {
		fmt.Fprintf(w, "[ERR] %v\n", err)
		log.Printf("[ERR] saveAndExtractZipStream error: %v", err)
		return
	}

	fmt.Fprintln(w, "[DONE] Upload and extract complete.")
	flusher.Flush()
}

// --- Upload ZIP + Run Script ---
func uploadWithScriptHandler(w http.ResponseWriter, r *http.Request) {
	enableStreaming(w)
	flusher := w.(http.Flusher)

	log.Println("[INFO] Upload with script handler started")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(500 << 20)
	if err != nil {
		log.Printf("[ERR] Error parsing form: %v", err)
		http.Error(w, "[ERR] Error parsing form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "[ERR] File is required", http.StatusBadRequest)
		log.Printf("[ERR] File is required: %v", err)
		return
	}
	defer file.Close()

	targetDir := r.FormValue("target")
	if targetDir == "" {
		http.Error(w, "[ERR] Target directory is required", http.StatusBadRequest)
		log.Println("[ERR] Target directory is required")
		return
	}

	script := r.FormValue("script")
	if script == "" {
		http.Error(w, "[ERR] Script is required", http.StatusBadRequest)
		log.Println("[ERR] Script is required")
		return
	}

	// Now safe to write response and flush because input is validated
	fmt.Fprintln(w, "[INFO] Starting upload + script execution")
	flusher.Flush()

	fmt.Fprintf(w, "[INFO] Uploading %s to %s\n", fileHeader.Filename, targetDir)
	flusher.Flush()

	err = saveAndExtractZipStream(w, flusher, file, fileHeader.Filename, targetDir)
	if err != nil {
		fmt.Fprintf(w, "[ERR] %v\n", err)
		log.Printf("[ERR] saveAndExtractZipStream error: %v", err)
		return
	}

	fmt.Fprintf(w, "[INFO] Running script: %s\n", script)
	flusher.Flush()

	err = runScriptStreaming(w, script, targetDir)
	if err != nil {
		fmt.Fprintf(w, "[ERR] Script execution failed: %v\n", err)
		log.Printf("[ERR] Script execution failed: %v", err)
	}
}

// --- Run Script Only ---
func runScriptOnlyHandler(w http.ResponseWriter, r *http.Request) {
	enableStreaming(w)
	flusher := w.(http.Flusher)

	log.Println("[INFO] Run script handler started")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	script := r.FormValue("script")
	if script == "" {
		http.Error(w, "[ERR] Script is required", http.StatusBadRequest)
		log.Println("[ERR] Script is required")
		return
	}

	targetDir := r.FormValue("target")
	if targetDir == "" {
		targetDir = "./tmp"
	}

	fmt.Fprintf(w, "[INFO] Executing script in %s: %s\n", targetDir, script)
	flusher.Flush()

	err := runScriptStreaming(w, script, targetDir)
	if err != nil {
		fmt.Fprintf(w, "[ERR] Script execution failed: %v\n", err)
		log.Printf("[ERR] Script execution failed: %v", err)
	}
}

// --- Helper: Streaming ZIP Upload + Extract ---
func saveAndExtractZipStream(w http.ResponseWriter, flusher http.Flusher, file io.Reader, filename, targetDir string) error {
	log.Println("[INFO] Saving and extracting ZIP stream...")

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create target dir: %v", err)
	}

	zipPath := filepath.Join(targetDir, filename)
	outFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}

	defer outFile.Close()

	fmt.Fprintf(w, "[INFO] Saving file to %s\n", zipPath)
	flusher.Flush()

	_, err = io.Copy(outFile, file)
	if err != nil {
		return fmt.Errorf("could not save file: %v", err)
	}

	fmt.Fprintln(w, "[INFO] Extracting ZIP...")
	flusher.Flush()

	err = extractZipStream(w, flusher, zipPath, targetDir)
	if err != nil {
		return fmt.Errorf("extract failed: %v", err)
	}

	// Delete the zip file after successful extraction
	err = os.Remove(zipPath)
	if err != nil {
		// Just log error but don’t fail
		log.Printf("[WARN] Failed to delete zip file %s: %v", zipPath, err)
	} else {
		fmt.Fprintf(w, "[INFO] Deleted zip file %s\n", zipPath)
		flusher.Flush()
	}

	return nil
}

// --- Extract ZIP with log ---
func extractZipStream(w http.ResponseWriter, flusher http.Flusher, zipFilePath, targetDir string) error {
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("could not open zip: %v", err)
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		if strings.HasPrefix(f.Name, "__MACOSX") || strings.HasPrefix(f.Name, "._") {
			continue
		}

		path := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return fmt.Errorf("create dir error: %v", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			return fmt.Errorf("create dir error: %v", err)
		}

		inFile, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry error: %v", err)
		}

		outFile, err := os.Create(path)
		if err != nil {
			inFile.Close()
			return fmt.Errorf("create file error: %v", err)
		}

		_, err = io.Copy(outFile, inFile)

		// Close files ASAP (tidak defer di loop)
		inFile.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("extract file error: %v", err)
		}

		fmt.Fprintf(w, "[INFO] Extracted: %s\n", path)
		flusher.Flush()
	}
	return nil
}

// --- Run Script with Streaming ---
func runScriptStreaming(w http.ResponseWriter, script, targetDir string) error {
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = targetDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout error: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start error: %v", err)
	}

	flusher := w.(http.Flusher)

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintf(w, "[OUT] %s\n", scanner.Text())
			flusher.Flush()
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(w, "[ERR] %s\n", scanner.Text())
			flusher.Flush()
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("script exited with error: %v", err)
	}

	fmt.Fprintln(w, "[DONE] Script executed successfully.")
	flusher.Flush()
	return nil
}

// --- Enable streaming ---
func enableStreaming(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Transfer-Encoding", "chunked")
}

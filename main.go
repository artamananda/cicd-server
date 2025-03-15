package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Check if method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method")
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(500 << 20) // 500 MB limit
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		log.Println("Error parsing form:", err)
		return
	}

	// Retrieve the uploaded file and its header
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		log.Println("Error retrieving the file:", err)
		return
	}
	defer file.Close()

	// Validate that the uploaded file is a ZIP file
	if !strings.HasSuffix(fileHeader.Filename, ".zip") {
		http.Error(w, "Only ZIP files are allowed", http.StatusBadRequest)
		log.Println("Only ZIP files are allowed")
		return
	}

	// Retrieve the target directory from form value
	targetDir := r.FormValue("target")
	if targetDir == "" {
		http.Error(w, "Target directory not provided", http.StatusBadRequest)
		log.Println("Target directory not provided")
		return
	}

	// Ensure the target directory exists
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		log.Println("Error creating upload directory:", err)
		return
	}

	// Prepare the target file path
	targetFilePath := filepath.Join(targetDir, fileHeader.Filename)

	// Create the file on the server
	outFile, err := os.Create(targetFilePath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		log.Println("Error creating file:", err)
		return
	}
	defer outFile.Close()

	// Save the uploaded file to disk
	_, err = io.Copy(outFile, file)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		log.Println("Error saving the file:", err)
		return
	}

	// Extract the ZIP file to the target directory
	err = extractZip(targetFilePath, targetDir)
	if err != nil {
		http.Error(w, "Error extracting ZIP file", http.StatusInternalServerError)
		log.Println("Error extracting ZIP file:", err)
		return
	}

	// Retrieve the script to execute
	script := r.FormValue("script")
	if script != "" {
		// Run the script after extraction
		err := runScript(script, targetDir)
		if err != nil {
			http.Error(w, "Error running the script", http.StatusInternalServerError)
			log.Println("Error running the script:", err)
			return
		}
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded, extracted, and script executed successfully in %s", targetDir)
}

// Function to extract ZIP file to a directory
func extractZip(zipFilePath, targetDir string) error {
	// Open the ZIP file
	zipFile, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("could not open zip file: %v", err)
	}
	defer zipFile.Close()

	// Loop through each file in the ZIP archive and extract them
	for _, zipFileEntry := range zipFile.File {
		// Skip files or directories that start with "__MACOSX" or "._"
		if strings.HasPrefix(zipFileEntry.Name, "__MACOSX") || strings.HasPrefix(zipFileEntry.Name, "._") {
			log.Printf("Skipping unwanted file: %s\n", zipFileEntry.Name)
			continue
		}

		// Prepare the output file path
		extractedFilePath := filepath.Join(targetDir, zipFileEntry.Name)

		// Ensure the directory for this file exists
		if zipFileEntry.FileInfo().IsDir() {
			err = os.MkdirAll(extractedFilePath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("could not create directory %s: %v", extractedFilePath, err)
			}
			continue
		}

		// Create the file on disk
		outFile, err := os.Create(extractedFilePath)
		if err != nil {
			return fmt.Errorf("could not create file %s: %v", extractedFilePath, err)
		}
		defer outFile.Close()

		// Open the file inside the ZIP archive
		zipFileEntryReader, err := zipFileEntry.Open()
		if err != nil {
			return fmt.Errorf("could not open file in zip: %v", err)
		}
		defer zipFileEntryReader.Close()

		// Copy the contents of the file to the output file
		_, err = io.Copy(outFile, zipFileEntryReader)
		if err != nil {
			return fmt.Errorf("could not extract file %s: %v", zipFileEntry.Name, err)
		}
	}

	return nil
}

// Function to run the script on the extracted directory
func runScript(script, targetDir string) error {
	// Prepare the script with target directory if needed
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = targetDir

	// Run the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing script: %v\nOutput: %s", err, output)
	}

	log.Printf("Script executed successfully: %s", output)
	return nil
}

func main() {
	// Define the HTTP route
	http.HandleFunc("/upload", uploadFileHandler)

	// Start the server
	log.Println("Server started on :8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Invalid request method")
		return
	}
	r.ParseMultipartForm(10 << 20)
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		log.Println("Error retrieving the file")
		return
	}
	defer file.Close()

	targetDir := "/home/arta"
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		http.Error(w, "Error creating upload directory", http.StatusInternalServerError)
		log.Println("Error creating upload directory")
		return
	}

	targetFilePath := filepath.Join(targetDir, fileHeader.Filename)

	outFile, err := os.Create(targetFilePath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		log.Println("Error creating file")
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		log.Println("Error saving the file")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully and saved to %s", targetFilePath)
}

func main() {
	http.HandleFunc("/upload", uploadFileHandler)
	log.Println("Server started on :8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		log.Fatal(err)
	}
}

package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"clearify/utils"

	"github.com/disintegration/imaging"
)

type ProcessedImage struct {
	Filename string
	Data     []byte
	MimeType string
}

func ServerTemplate(w http.ResponseWriter, r *http.Request) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	tmplPath := filepath.Join(wd, "template", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		response := utils.Response{
			Status:  http.StatusInternalServerError,
			Message: "Failed to server html",
			Error:   err.Error(),
		}
		response.ValidResponse(w)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		response := utils.Response{
			Status:  http.StatusInternalServerError,
			Message: "Failed to server html",
			Error:   err.Error(),
		}
		response.ValidResponse(w)
		return
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println("Method not allowed")
		response := utils.Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "Failed",
			Error:   "Method not allowed",
		}
		response.ValidResponse(w)
		return
	}

	const maxUploadSize = 100 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	scale, sharping := r.FormValue("scale"), r.FormValue("sharping")

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		fmt.Println("File Size more than 100 MB")
		response := utils.Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "File size more than 100 MB",
			Error:   err.Error(),
		}
		response.ValidResponse(w)
		return
	}

	fmt.Println("number of files", len(r.MultipartForm.File["images"]))
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		response := utils.Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "No files uploaded",
			Error:   nil,
		}
		response.ValidResponse(w)
		return
	}

	scaleFloat64, _ := strconv.ParseFloat(scale, 64)
	sharpingFloat64, _ := strconv.ParseFloat(sharping, 64)
	contrast, _ := strconv.ParseFloat(r.FormValue("contrast"), 64)
	quality, _ := strconv.Atoi(r.FormValue("quality"))
	if quality == 0 {
		quality = 85 // Default quality
	}

	imageChan := make(chan ProcessedImage, len(files))
	var wg sync.WaitGroup

	for _, fileHeader := range files {
		wg.Add(1)
		go func(fh *multipart.FileHeader) {
			defer wg.Done()

			file, err := fh.Open()
			if err != nil {
				fmt.Printf("Error opening file: %v\n", err)
				return
			}
			defer file.Close()

			processedImage, err := processImage(file, fh.Filename, scaleFloat64, sharpingFloat64, quality, contrast)
			if err != nil {
				fmt.Printf("Error processing image: %v\n", err)
				return
			}

			imageChan <- processedImage
		}(fileHeader)
	}

	go func() {
		defer close(imageChan)
		wg.Wait()
	}()

	processedImages := make([]ProcessedImage, 0, len(files))
	for img := range imageChan {
		processedImages = append(processedImages, img)
	}

	if len(processedImages) == 0 {
		response := utils.Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "Failed to process any images",
			Error:   "Failed to process any images",
		}
		response.ValidResponse(w)
		return
	}

	if len(processedImages) == 1 {
		sendSingleImage(w, processedImages[0])
	} else {
		sendZipFile(w, processedImages)
	}
}

func processImage(file io.Reader, fileName string, scale float64, sharping float64, quality int, contrast float64) (ProcessedImage, error) {
	src, err := imaging.Decode(file)
	if err != nil {
		return ProcessedImage{}, fmt.Errorf("failed to decode image: %v", err)
	}

	width := src.Bounds().Dx()
	height := src.Bounds().Dy()

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	resized := imaging.Resize(src, newWidth, newHeight, imaging.Lanczos)

	image := resized

	if sharping != 0 {
		sharp := imaging.Sharpen(resized, sharping)
		enhance := imaging.AdjustContrast(sharp, contrast)
		image = enhance
	}

	buf := new(bytes.Buffer)
	var mimeType string

	// Determine the best format based on image characteristics
	format := determineOptimalFormat(image)

	switch format {
	case "jpeg":
		err = jpeg.Encode(buf, image, &jpeg.Options{Quality: quality})
		mimeType = "image/jpeg"
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(buf, image)
		mimeType = "image/png"
	}

	if err != nil {
		return ProcessedImage{}, fmt.Errorf("failed to encode image: %v", err)
	}

	extension := filepath.Ext(fileName)
	newFileName := strings.TrimSuffix(fileName, extension) + "." + format

	return ProcessedImage{
		Filename: "enhanced_" + newFileName,
		Data:     buf.Bytes(),
		MimeType: mimeType,
	}, nil
}

func determineOptimalFormat(img image.Image) string {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	hasTransparency := false

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				hasTransparency = true
				break
			}
		}
		if hasTransparency {
			break
		}
	}

	if hasTransparency {
		return "png"
	} else {
		return "jpeg"
	}
}

func sendSingleImage(w http.ResponseWriter, image ProcessedImage) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", image.Filename))
	w.Header().Set("Content-Type", image.MimeType)
	w.Write(image.Data)
}

func sendZipFile(w http.ResponseWriter, images []ProcessedImage) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, img := range images {
		zipFile, err := zipWriter.Create(img.Filename)
		if err != nil {
			response := utils.Response{
				Status:  http.StatusMethodNotAllowed,
				Message: "Failed to create zip file",
				Error:   err.Error(),
			}
			response.ValidResponse(w)
			return
		}
		_, err = zipFile.Write(img.Data)
		if err != nil {
			response := utils.Response{
				Status:  http.StatusMethodNotAllowed,
				Message: "Failed to create zip file",
				Error:   err.Error(),
			}
			response.ValidResponse(w)
			return
		}
	}

	err := zipWriter.Close()
	if err != nil {
		response := utils.Response{
			Status:  http.StatusMethodNotAllowed,
			Message: "Failed to close zip file",
			Error:   err.Error(),
		}
		response.ValidResponse(w)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=enhanced_images.zip")
	w.Header().Set("Content-Type", "application/zip")
	w.Write(buf.Bytes())
}

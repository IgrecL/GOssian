package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MAX_GOROUTINES = 100

// More concise error handling
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	listen, err := net.Listen("tcp", "localhost:8000")
	check(err)
	defer listen.Close()
	for {
		socket, err := listen.Accept()
		check(err)
		go process(socket)
	}
}

func process(socket net.Conn) {
	
	// Receiving image and processing parameters
	var inputImage image.Image
	radius, sigma, quali := handleRequest(socket, &inputImage)
	t0 := time.Now()

	// Generating the variables given to the goroutines
	mask := generateMask(radius, sigma)
	width, height := inputImage.Bounds().Dx(), inputImage.Bounds().Dy()
	processedImage := image.NewRGBA(image.Rect(0, 0, width, height))

	// Creating and syncing the goroutines 
	wg := new(sync.WaitGroup)
	inputChan := make(chan [2]int, MAX_GOROUTINES)
	outputChan := make(chan [5]int, MAX_GOROUTINES)
	for i := 0; i < MAX_GOROUTINES; i++ {
		wg.Add(1)
		go gaussianBlur(inputImage, mask, inputChan, outputChan, wg)
	}
	
	// Filling inputChan with the coordinates of each pixel and processing the goroutines output as soon as possible
	serve := false
	counter := 1
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			inputChan <- [2]int{x, y}
			if serve {
				o := <-outputChan
				processedImage.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
			}
			if counter == MAX_GOROUTINES {
				serve = true
			} else {
				counter++
			}
		}
	}

	// Processing the last goroutines
	for k := 0; k < MAX_GOROUTINES; k++ {
		o := <-outputChan
		processedImage.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
	}
	close(inputChan)
	wg.Wait()
	close(outputChan)
	fmt.Println(">> Gaussian blur:  ", time.Now().Sub(t0))

	// Sending back the processed image
	sendImage(socket, processedImage, quali)
	fmt.Println("Deconnecting from client", socket.LocalAddr().String())
	fmt.Println()
	socket.Close()
}

// Receiving the image and the processing parameters
func handleRequest(socket net.Conn, inputImage *image.Image) (int, float64, int) {
	fmt.Println("Connecting to client", socket.LocalAddr().String())
	
	// Receiving message type (0: image to process, 1: server shutdown)
	var t int32
	err := binary.Read(socket, binary.LittleEndian, &t)
	check(err)
	if t == 1 {
		fmt.Println("Server shutdown")
		os.Exit(1)
	}
	
	// Receiving processing parameters
	paramBytes := make([]byte, 256)
	_, err = socket.Read(paramBytes)
	check(err)
	t0 := time.Now()
	param := strings.Split(string(paramBytes), ":")
	radius, _ := strconv.Atoi(param[0])
	sigma, _ := strconv.ParseFloat(param[1], 8)
	quali, _ := strconv.Atoi(param[2])
	
	// Decoding the image received as a base64 string
	temp, err := bufio.NewReader(socket).ReadString('\n')
	check(err)
	byteImage, err := base64.StdEncoding.DecodeString(temp)
	check(err)
	*inputImage, err = jpeg.Decode(bytes.NewReader(byteImage))
	check(err)

	fmt.Println(">> Receiving image:", time.Now().Sub(t0))
	return radius, sigma, quali
}

// Sending the processed image back to the client
func sendImage(socket net.Conn, processedImage *image.RGBA, quali int) {
	t0 := time.Now()

	path := time.Now().String() + " [" + socket.LocalAddr().String() + "].jpeg"
	file, err := os.Create(path)
	check(err)
	jpeg.Encode(file, processedImage, &jpeg.Options{Quality: quali})
	file.Close()

	fmt.Fprintf(socket, encodeImage(path)+"\n")
	
	t1 := time.Now()
	fmt.Println(">> Sending image:  ", t1.Sub(t0))
}

// Loading image from file and encoding it as a base64 string
func encodeImage(imgPath string) string {
	f, err := os.Open(imgPath)
	check(err)
	defer f.Close()
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	check(err)
	return base64.StdEncoding.EncodeToString(content)
}

// Returns the value of the normal distribution at (x, y)
func normpdf(x, y, sigma float64) float64 {
	num := -(x*x + y*y)
	denom := 2 * sigma * sigma
	return 1 / (2 * math.Pi * sigma * sigma) * math.Pow(math.E, (num/denom))
}

// Generating the mask used to do the Gaussian blur
func generateMask(radius int, sigma float64) [][]float64 {
	mask := make([][]float64, 2*radius+1)
	for i := range mask {
		mask[i] = make([]float64, 2*radius+1)
	}
	for i := -radius; i < radius+1; i++ {
		for j := -radius; j < radius+1; j++ {
			mask[i+radius][j+radius] = normpdf(float64(i), float64(j), sigma)
		}
	}
	return mask
}

// Applying the Gaussian blur for a pixel of given coordinates (goroutine)
func gaussianBlur(image image.Image, mask [][]float64, inputChan chan [2]int, outputChan chan [5]int, wg *sync.WaitGroup) {
	defer wg.Done()
	for input := range inputChan {

		x, y := input[0], input[1]
		var red, green, blue float64
		var denom float64
		width, height := image.Bounds().Dx(), image.Bounds().Dy()
		radius := (len(mask) - 1) / 2
		
		// 2D convolution of the image and the mask centered on (x, y)
		for i := -radius; i <= radius; i++ {
			for j := -radius; j <= radius; j++ {
				if x+i >= 0 && x+i < width && y+j >= 0 && y+j < height {
					r, g, b, _ := image.At(x+i, y+j).RGBA()
					red += mask[radius+i][radius+j] * (float64(r) / 257)
					green += mask[radius+i][radius+j] * (float64(g) / 257)
					blue += mask[radius+i][radius+j] * (float64(b) / 257)
					denom += mask[radius+i][radius+j]
				}
			}
		}

		// Dividing by denom to get the weigthed mean
		if denom != 0 {
			red /= denom
			green /= denom
			blue /= denom
		}

		output := [5]int{x, y, int(red), int(green), int(blue)}
		outputChan <- output
	}
}


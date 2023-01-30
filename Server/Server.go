package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MAX_GOROUTINES = 100

func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	listen, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	defer listen.Close()
	for {
		socket, err := listen.Accept()
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		go processing(socket)
	}
}

func processing(socket net.Conn) {
	
	// Récupération de l'image et des paramètres de traitement
	imgInput, radius, sigma, quali := handleRequest(socket)
	t0 := time.Now()

	// Génération du masque pour le flou gaussien
	mask := generateMask(radius, sigma)
	t1 := time.Now()
	fmt.Println("Génération du masque :  ", t1.Sub(t0))

	width, height := imgInput.Bounds().Dx(), imgInput.Bounds().Dy()
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))

	inputChan := make(chan [2]int, MAX_GOROUTINES)
	outputChan := make(chan [5]int, MAX_GOROUTINES)
	wg := new(sync.WaitGroup)

	// On ajoute les goroutines au waitgroup et on les exécute
	for i := 0; i < MAX_GOROUTINES; i++ {
		wg.Add(1)
		go gaussianBlur(imgInput, mask, inputChan, outputChan, wg)
	}

	// On remplit inputChan avec les coordonnées de chaque pixel, et on traite l'output des goroutines dès que possible
	serve := false
	counter := 1
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			inputChan <- [2]int{x, y}
			if serve {
				o := <-outputChan
				newImg.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
			}
			if counter == MAX_GOROUTINES {
				serve = true
			} else {
				counter++
			}
		}
	}

	// On traite les goroutines restantes et on ferme les channels
	for k := 0; k < MAX_GOROUTINES; k++ {
		o := <-outputChan
		newImg.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
	}
	close(inputChan)
	wg.Wait()
	close(outputChan)

	t2 := time.Now()
	fmt.Println("Flou gaussien :         ", t2.Sub(t1))

	// On envoie le résultat au client
	sendImage(socket, newImg, quali)

	// Fermeture du socket
	socket.Close()
	fmt.Println("Déconnexion du client :  " + socket.LocalAddr().String())
	fmt.Println()
}

// Récupération de l'image et des paramètres de traitement
func handleRequest(socket net.Conn) (image.Image, int, float64, int) {

	// Connexion au socket
	fmt.Println("Connexion au client :    " + socket.LocalAddr().String())

	// On reçoit le type de message (0 : image à traiter, 1 : fermeture du serveur)
	var t int32
	err := binary.Read(socket, binary.LittleEndian, &t)
	check(err)
	if t == 1 {
		fmt.Println("Fermeture du serveur")
		os.Exit(1)
	}

	// On reçoit les paramètres du traitement
	paramBytes := make([]byte, 256)
	_, err = socket.Read(paramBytes)
	check(err)
	param := strings.Split(string(paramBytes), ":")
	radius, _ := strconv.Atoi(param[0])
	sigma, _ := strconv.ParseFloat(param[1], 8)
	quali, _ := strconv.Atoi(param[2])

	// On reçoit la taille de l'image à traiter (en octets)
	var imageSize int32
	err = binary.Read(socket, binary.LittleEndian, &imageSize)
	check(err)
	fmt.Println("Image reçue :           ", imageSize, "octets")

	// On lit l'output du socket et on le stocke dans un buffer qu'on décode
	imageByte := make([]byte, int(imageSize))
	_, err = socket.Read(imageByte)
	check(err)
	imageReader := bytes.NewReader(imageByte)
	t1 := time.Now()

	// On décode l'image en image.Image
	imgInput, err := jpeg.Decode(imageReader)
	check(err)
	t2 := time.Now()
	fmt.Println("Décodage de l'image :   ", t2.Sub(t1))

	return imgInput, radius, sigma, quali
}

// Envoie l'image traitée au client
func sendImage(socket net.Conn, newImg *image.RGBA, quali int) {
	t0 := time.Now()

	// Impression de l'image traitée
	path := time.Now().String() + " " + socket.LocalAddr().String() + ".jpeg"
	out, err := os.Create(path)
	check(err)
	jpeg.Encode(out, newImg, &jpeg.Options{Quality: quali})
	out.Close()

	// On détermine la taille de l'image (en octets)
	fileInfo, err := os.Stat(path)
	check(err)
	fileSize := fileInfo.Size()

	// On envoie la taille de l'image au client
	sizeBuf := new(bytes.Buffer)
	err = binary.Write(sizeBuf, binary.LittleEndian, int32(fileSize))
	check(err)
	_, err = socket.Write(sizeBuf.Bytes())
	check(err)

	// On envoie le contenu de output.jpeg dans le socket
	sent, err := os.Open(path)
	check(err)
	defer sent.Close()
	_, err = io.Copy(socket, sent)
	check(err)

	t1 := time.Now()
	fmt.Println("Envoi de l'image :      ", t1.Sub(t0))
}

// Renvoie la valeur de loi normale à deux variables (x, y)
func normpdf(x, y, sigma float64) float64 {
	num := -(x*x + y*y)
	denom := 2 * sigma * sigma
	return 1 / (2 * math.Pi * sigma * sigma) * math.Pow(math.E, (num/denom))
}

// Génération du masque utilisé pour réaliser le flou gaussien
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

// Application du flou gaussien pour un pixel de coordonnées donnée
func gaussianBlur(image image.Image, mask [][]float64, inputChan chan [2]int, outputChan chan [5]int, wg *sync.WaitGroup) {
	defer wg.Done()
	for input := range inputChan {

		x, y := input[0], input[1]
		var red, green, blue float64
		var denom float64
		width, height := image.Bounds().Dx(), image.Bounds().Dy()
		radius := (len(mask) - 1) / 2

		// Convolution 2D de l'image et du masque centrée en (x, y)
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

		// On divise les trois couleurs par le dénominateur pour obtenir la moyenne pondérée par la gaussienne
		if denom != 0 {
			red /= denom
			green /= denom
			blue /= denom
		}

		output := [5]int{x, y, int(red), int(green), int(blue)}
		outputChan <- output
	}
}

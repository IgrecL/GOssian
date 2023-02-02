package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"
)

// More concise error handling
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	socket, err := net.Dial("tcp", "localhost:8000")
	check(err)

	// The user has to write the path to the file, or "exit" to close the server
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please write the path to your image:\n>> ")
	path, _ := reader.ReadString('\n')
	path = path[:len(path)-1]
	
	// Sending message type (0: image to process, 1: server shutdown)
	t := 0
	if path == "exit" {
		t = 1
	}
	typeBuf := new(bytes.Buffer)
	err = binary.Write(typeBuf, binary.LittleEndian, int32(t))
	check(err)
	_, err = socket.Write(typeBuf.Bytes())
	check(err)
	
	// If the 'server shutdown' command is sent, stopping the program here
	if t == 1 {
		os.Exit(1)
	}
	
	// The user has to give 3 processing parameters: mask radius, Gaussian blur intensity, image compression
	var radius, intensity, compression string
	fmt.Print("Mask radius? (blur precision) [int]\n>> ")
	for {
		radius, _ = reader.ReadString('\n')
		radius = radius[:len(radius)-1]
		_, err = strconv.Atoi(radius)
		if err == nil {
			break
		} else {
			fmt.Print("Please give an integer!\n>> ")
		}
	}
	fmt.Print("Gaussian blur intensity? (standard deviation for the normal distribution) [float]\n>> ")
	for {
		intensity, _ = reader.ReadString('\n')
		intensity = intensity[:len(intensity)-1]
		_, err = strconv.ParseFloat(intensity, 8)
		if err == nil {
			break
		} else {
			fmt.Print("Please give a float!\n>> ")
		}
	}
	fmt.Print("Jpeg compression? (in percents) [int]\n>> ")
	for {
		compression, _ = reader.ReadString('\n')
		compression = compression[:len(compression)-1]
		_, err = strconv.Atoi(compression)
		if err == nil {
			break
		} else {
			fmt.Print("Please give an integer!\n>> ")
		}
	}

	// Sending the parameters and the image to process
	parameters := radius + ":" + intensity + ":" + compression + ":"
	_, err = socket.Write([]byte(parameters))
	check(err)
	fmt.Fprintf(socket, encodeImage(path)+"\n")

	// Receiving back the image once processed
	t0 := time.Now()
	temp, err := bufio.NewReader(socket).ReadString('\n')
	check(err)
	byteImage, err := base64.StdEncoding.DecodeString(temp)
	check(err)
	imgReceived, err := jpeg.Decode(bytes.NewReader(byteImage))
	check(err)
	fmt.Println("Finished after", time.Now().Sub(t0))
	file, err := os.Create(time.Now().String())
	check(err)
	jpeg.Encode(file, imgReceived, nil)
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

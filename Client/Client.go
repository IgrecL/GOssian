package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image/jpeg"
	_ "image/jpeg"
	"io"
	"net"
	"os"
)

// Gestion des erreurs
func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	// Connexion au serveur
	socket, err := net.Dial("tcp", "localhost:8000")
	check(err)

	// L'utilisateur doit écrire le chemin vers l'image qu'il veut faire traiter, ou "exit" pour fermer le serveur
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Écrivez le chemin vers l'image à traiter : ")
	path, _ := reader.ReadString('\n')
	path = path[:len(path)-1]

	// On envoie 0 si le socket sert à traiter une image, et 1 s'il sert à fermer le serveur
	t := 0
	if path == "exit" {
		t = 1
	}
	typeBuf := new(bytes.Buffer)
	err = binary.Write(typeBuf, binary.LittleEndian, int32(t))
	check(err)
	_, err = socket.Write(typeBuf.Bytes())
	check(err)

	// Si on a envoyé la commande de fermeture du serveur, on s'arrête ici
	if t == 1 {
		os.Exit(1)
	}

	// L'utilisateur doit entrer les paramètres de traitement : rayon du masque, puissance du flou gaussien, compression de l'image
	fmt.Println("Rayon du masque à appliquer ? (précision du flou) [int]")
	radius, _ := reader.ReadString('\n')
	radius = radius[:len(radius)-1]
	fmt.Println("Puissance du flou gaussien ? (écart type de la loi normale) [float]")
	power, _ := reader.ReadString('\n')
	power = power[:len(power)-1]
	fmt.Println("Compression du jpeg ? (en pourcents) [int]")
	compression, _ := reader.ReadString('\n')
	compression = compression[:len(compression)-1]

	// On envoie les paramètres du traitement
	parameters := radius + ":" + power + ":" + compression + ":"
	_, err = socket.Write([]byte(parameters))
	check(err)

	// On calcule la taille de l'image et on l'envoie (en octets)
	fileInfo, err := os.Stat(path)
	check(err)
	fileSize := fileInfo.Size()
	sizeBuf := new(bytes.Buffer)
	err = binary.Write(sizeBuf, binary.LittleEndian, int32(fileSize))
	check(err)
	_, err = socket.Write(sizeBuf.Bytes())
	check(err)

	// On ouvre l'image au chemin indiqué
	file, err := os.Open(path)
	check(err)

	// On envoie l'image
	size, err := io.Copy(socket, file)
	check(err)
	fmt.Println("Image envoyée :", size, "octets")
	file.Close()

	/* Le serveur applique le flou gaussien */

	// On reçoit la taille de l'image traitée (en octets)
	var imageSize int32
	err = binary.Read(socket, binary.LittleEndian, &imageSize)
	check(err)
	fmt.Println("Image reçue   :", imageSize, "octets")

	// On reçoit l'image
	imageByte := make([]byte, imageSize)
	_, err = socket.Read(imageByte)
	check(err)

	// On encode l'image reçue et on crée le fichier d'output
	imageReader := bytes.NewReader(imageByte)
	imgReceived, err := jpeg.Decode(imageReader)
	check(err)
	out, err := os.Create("output.jpeg")
	check(err)
	jpeg.Encode(out, imgReceived, nil)
}

package main

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"os"
)

func main() {
	var img image.Image = importerImage("image.jpg")
	println(img)
	genererMasque(3)
	// flouGaussien(img, 3)

}

func importerImage(chemin string) image.Image {

	// Importation de l'image
	fichier, err := os.Open(chemin)
	if err != nil {
		panic(err)
	}

	// Décodage de l'image
	img, err := jpeg.Decode(fichier)
	if err != nil {
		panic(err)
	}

	return img
}

func genererMasque(rayon int) [][]int {
	masque := make([][]int, 2*rayon+1)
	for i := range masque {
		masque[i] = make([]int, 2*rayon+1)
	}
	for i := range masque {
		for j := range masque[i] {
			fmt.Print(masque[i][j], " ")
		}
		fmt.Println()
	}
	return nil
}

func flouGaussien(img image.Image, rayon int) {
	dim := img.Bounds()
	imageRVB := image.NewRGBA(image.Rect(0, 0, dim.Dx(), dim.Dy()))
	print(imageRVB)

}

// func flouPixel(x int, y int, image image.Image) (res [5]int) { // res[0]=x, res[1]=y, res[2]=red, res[3]=green, res[4]=blue

// 	res[0],res[1] := x, y

// 	return res

// }

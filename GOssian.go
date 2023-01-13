package main

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"math"
	"os"
)

func main() {
	var img image.Image = importerImage("test2.jpg")
	rayon := 2
	println(img)
	genererMasque(rayon, 1.0)
	conversionImage(img, rayon)
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

// Renvoie la valeur de loi normale à deux variables (x, y)
func normpdf(x, y, sigma float64) float64 {
	numerateur := -(x*x + y*y)
	denominateur := 2 * sigma * sigma
	return 1 / (2 * math.Pi * sigma * sigma) * math.Pow(math.E, (numerateur/denominateur))
}

func genererMasque(rayon int, sigma float64) [][]float64 {

	// Initialisation du masque
	masque := make([][]float64, 2*rayon+1)
	for i := range masque {
		masque[i] = make([]float64, 2*rayon+1)
	}

	// Remplissage du masque
	for i := -rayon; i < rayon+1; i++ {
		for j := -rayon; j < rayon+1; j++ {
			masque[i+rayon][j+rayon] = normpdf(float64(i), float64(j), sigma)
		}
	}

	return nil
}

func flouGaussien(img image.RGBA, rayon int) {

}

func conversionImage(img image.Image, rayon int) {
	largeur, hauteur := img.Bounds().Dx(), img.Bounds().Dy()

	// Initialisation du tableau 3D avec un contour
	new := make([][][]float64, largeur+2*rayon)
	for k := range new {
		new[k] = make([][]float64, hauteur+2*rayon)
		for i := 0; i < len(new); i++ {
			for j := 0; j < len(new[i]); j++ {
				new[i][j] = make([]float64, 3)
			}
		}
	}

	// Remplissage du tableau 3D
	for i := 0; i < largeur; i++ {
		for j := 0; j < hauteur; j++ {
			for k := 0; k < 3; k++ {
				r, g, b, _ := img.At(i, j).RGBA()
				new[i+rayon][j+rayon][0] = float64(r)
				new[i+rayon][j+rayon][1] = float64(g)
				new[i+rayon][j+rayon][2] = float64(b)
			}
		}
	}

	// On remplit le contour en appliquant l'image en miroir
	for i := rayon; i < largeur+rayon; i++ {
		for j := 0; j < rayon; j++ {
			// Gauche et droite
			new[i][j] = new[i][2*rayon-j]
			new[i][rayon+hauteur+j] = new[i][rayon+hauteur-2-j]
			// Haut et bas
			new[j][i] = new[2*rayon-j][i]
			new[rayon+largeur+j][i] = new[rayon+largeur-2-j][i]
		}
	}

	for i := range new {
		for j := range new[i] {
			if new[i][j][0] == 0 {
				fmt.Print("  0   ")
			} else {
				fmt.Print(new[i][j][0], " ")
			}
		}
		fmt.Print("\n\n\n")
	}

}

// func flouPixel(x int, y int, image image.Image) (res [5]int) { // res[0]=x, res[1]=y, res[2]=red, res[3]=green, res[4]=blue

// 	res[0],res[1] := x, y

// 	return res

// }

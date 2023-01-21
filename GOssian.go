package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	"math"
	"os"
	"sync"
	"time"
)

const MAX_GOROUTINES = 100

func main() {
	rayon := 10
	imgInput := importerImage("image.jpg")
	masque := genererMasque(rayon, 1.0)
	imgTab := conversionImage(imgInput, rayon)

	largeur, hauteur := imgInput.Bounds().Dx(), imgInput.Bounds().Dy()

	newImg := image.NewRGBA(image.Rect(0, 0, largeur, hauteur))
    
    t0 := time.Now()

	inputChan := make(chan [2]int, 100)
	outputChan := make(chan [5]int, 100)
	wg := new(sync.WaitGroup)

	// On ajoute les goroutines au waitgroup et on les exécute
	for i := 0; i < MAX_GOROUTINES; i++ {
		wg.Add(1)
		go flouGaussien(imgTab, masque, inputChan, outputChan, wg)
	}

	serve := false
	compteur := 1

	for x := rayon; x < largeur+rayon; x++ {
		for y := rayon; y < hauteur+rayon; y++ {
			inputChan <- [2]int{x, y}
			if serve {
				o := <-outputChan
				newImg.Set(int(o[0])-rayon, int(o[1])-rayon, color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
			}
			if compteur == MAX_GOROUTINES {
				serve = true
			} else {
				compteur++
			}
		}
	}
	
	for k := 0; k < MAX_GOROUTINES; k++ {
		o := <-outputChan
		newImg.Set(int(o[0])-rayon, int(o[1])-rayon, color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
	}

	close(inputChan)
	wg.Wait()
	close(outputChan)
    
    t1 := time.Now()
	fmt.Println("Exécuter flou gaussien :", t1.Sub(t0))

	out, err := os.Create("output.jpeg")
	if err != nil {
		panic(err)
	}
	jpeg.Encode(out, newImg, nil)
	out.Close()
}

func importerImage(chemin string) image.Image {

    t0 := time.Now()

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

	t1 := time.Now()
	fmt.Println("Importation de l'image :", t1.Sub(t0))

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

	return masque
}

func vide(tab []float64) bool {
	for i := 0; i < 3; i++ {
		if tab[i] != 0 {
			return false
		}
	}
	return true
}

func conversionImage(img image.Image, rayon int) [][][]float64 {
	largeur, hauteur := img.Bounds().Dx(), img.Bounds().Dy()
	
	t0 := time.Now()

	// Initialisation et remplissage du tableau 3D avec un contour vide
	new := make([][][]float64, largeur+2*rayon)
	for k := range new {
		new[k] = make([][]float64, hauteur+2*rayon)
		for i := 0; i < len(new); i++ {
			for j := 0; j < len(new[i]); j++ {

				// Chaque case du tableau 2D contient un array de 3 pour stocker les couleurs
				new[i][j] = make([]float64, 3)

				// Si on se trouve après le contour, on récupère le RGB des pixels de img
				if i >= rayon && j >= rayon {
					r, g, b, _ := img.At(i-rayon, j-rayon).RGBA()
					new[i][j][0] = float64(r) / 257
					new[i][j][1] = float64(g) / 257
					new[i][j][2] = float64(b) / 257
				}

			}
		}
	}

	t1 := time.Now()
	fmt.Println("Remplissage du tableau :", t1.Sub(t0))

	// On remplit le contour en appliquant l'image en miroir de chaque côté du contour
	for i := rayon; i < largeur+rayon; i++ {
		for j := 0; j < rayon; j++ {
			// Haut et bas
			new[i][j] = new[i][2*rayon-j]
			new[i][rayon+hauteur+j] = new[i][rayon+hauteur-2-j]
		}
	}
	for i := 0; i < rayon; i++ {
		for j := rayon; j < hauteur+rayon; j++ {
			// Gauche et droite
			new[i][j] = new[2*rayon-i][j]
			new[rayon+largeur+i][j] = new[rayon+largeur-2-i][j]
		}
	}

	// Pour remplir les coins on fait des rotations axiales centrées sur les coins en question
	for i := 0; i < rayon; i++ {
		for j := 0; j < rayon; j++ {
			new[i][j] = new[2*rayon-i][2*rayon-j]                             // Coin en haut à gauche
			new[largeur+rayon+i][j] = new[largeur-i][j]                       // Coin en haut à droite
			new[i][hauteur+rayon+j] = new[i][hauteur-j]                       // Coin en bas  à droite
			new[largeur+rayon+i][hauteur+rayon+j] = new[largeur-i][hauteur-j] // Coin en bas  à droite
		}
	}

	t2 := time.Now()
	fmt.Println("Remplissage du contour :", t2.Sub(t1))

	return new
}

func flouGaussien(image [][][]float64, masque [][]float64, inputChan chan [2]int, outputChan chan [5]int, wg  *sync.WaitGroup) {
	defer wg.Done()
	for input := range inputChan {
		x, y := input[0], input[1]

		var somme [3]float64
		var denominateur float64
		rayon := (len(masque) - 1) / 2

		for i := range masque {
			for j := range masque {
				denominateur += masque[i][j]
			}
		}

		for i := -rayon; i < rayon; i++ {
			for j := -rayon; j < rayon; j++ {
				for k := 0; k < 3; k++ {
					somme[k] += masque[rayon+i][rayon+j] * image[x+i][y+j][k]
				}
			}
		}

		for k := 0; k < 3; k++ {
			somme[k] /= denominateur
		}

		output := [5]int{x, y, int(somme[0]), int(somme[1]), int(somme[2])}

		outputChan <- output
	}
}

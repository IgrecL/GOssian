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
	t0 := time.Now()
    
	// Importation de l'image à traiter
	imgInput := importerImage("test.jpg")
    t1 := time.Now()
	fmt.Println("Importation de l'image :", t1.Sub(t0))
	
	// Génération du masque pour le flou gaussien
	masque := genererMasque(rayon, 5.0)
    t2 := time.Now()
	fmt.Println("Génération du masque :  ", t2.Sub(t1))

	largeur, hauteur := imgInput.Bounds().Dx(), imgInput.Bounds().Dy()
	newImg := image.NewRGBA(image.Rect(0, 0, largeur, hauteur))

	inputChan := make(chan [2]int, 100)
	outputChan := make(chan [5]int, 100)
	wg := new(sync.WaitGroup)

	// On ajoute les goroutines au waitgroup et on les exécute
	for i := 0; i < MAX_GOROUTINES; i++ {
		wg.Add(1)
		go flouGaussien(imgInput, masque, inputChan, outputChan, wg)
	}
	
	// On remplit inputChan avec les coordonnées de chaque pixel, et on traite l'output des goroutines dès que possible
	serve := false
	compteur := 1
	for x := 0; x < largeur; x++ {
		for y := 0; y < hauteur; y++ {
			inputChan <- [2]int{x, y}
			if serve {
				o := <-outputChan
				newImg.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
			}
			if compteur == MAX_GOROUTINES {
				serve = true
			} else {
				compteur++
			}
		}
	}
	
	// On traite les goroutines restantes
	for k := 0; k < MAX_GOROUTINES; k++ {
		o := <-outputChan
		newImg.Set(int(o[0]), int(o[1]), color.RGBA{uint8(o[2]), uint8(o[3]), uint8(o[4]), 255})
	}

	close(inputChan)
	wg.Wait()
	close(outputChan)
    
    t3 := time.Now()
	fmt.Println("Flou gaussien :         ", t3.Sub(t2))
	
	// Impression de l'image traitée
	out, err := os.Create("output.jpeg")
	if err != nil {
		panic(err)
	}
	jpeg.Encode(out, newImg, nil)
	out.Close()
}

// Importation et décodage de l'image
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

// Génération du masque utilisé pour réaliser le flou gaussien
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

// Application du flou gaussien pour un pixel de coordonnées donnée
func flouGaussien(image image.Image, masque [][]float64, inputChan chan [2]int, outputChan chan [5]int, wg  *sync.WaitGroup) {
	defer wg.Done()
	for input := range inputChan {

		x, y := input[0], input[1]
		var rouge, vert, bleu float64
		var denominateur float64
		largeur, hauteur := image.Bounds().Dx(), image.Bounds().Dy()
		rayon := (len(masque) - 1) / 2
        
		// Convolution 2D de l'image et du masque centrée en (x, y)
		for i := -rayon; i <= rayon; i++ {
			for j := -rayon; j <= rayon; j++ {
				if x+i >= 0 && x+i < largeur && y+j >= 0 && y+j < hauteur {
					r, g, b, _ := image.At(x+i, y+j).RGBA()
					rouge += masque[rayon+i][rayon+j] * (float64(r) / 257) 
					vert += masque[rayon+i][rayon+j] * (float64(g) / 257)
					bleu += masque[rayon+i][rayon+j] * (float64(b) / 257)
				    denominateur += masque[rayon+i][rayon+j]
				}
		    }
	    }
		
		// On divise les trois couleurs par le dénominateur pour obtenir la moyenne pondérée par la gaussienne
		if denominateur != 0 {
			rouge /= denominateur
			vert /= denominateur
			bleu /= denominateur
		}

		output := [5]int{x, y, int(rouge), int(vert), int(bleu)}
		outputChan <- output
	}
}

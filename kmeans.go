package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	k         = 200
	threshold = .01

	path = "../../data/"
	file = "huiling1"
	ext  = ".jpg"
)

type datum struct {
	red     int
	green   int
	blue    int
	cluster int //clsidx or total cluster number
}

var data [][]datum
var clusters []datum
var clustersSum []datum
var width int
var height int

var mutex = &sync.Mutex{}

func main() {
	doKmeans()
}

func doKmeans() {
	fmt.Printf("k=%d\n", k)
	t0, _ := getTimeStamp(true)
	read()
	initClusters()
	for {
		setClusters()
		if k == setNewClusters() {
			break
		}
	}
	write()
	t1, _ := getTimeStamp(true)
	fmt.Printf("Time consumed=%d", (t1 - t0))
}

func getDistance(i datum, j datum) int {
	dfr, dfg, dfb := i.red-j.red, i.green-j.green, i.blue-j.blue
	return dfr*dfr + dfg*dfg + dfb*dfb
}

func setClusters2() {
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			cluster, dis := -1, math.MaxInt64
			for p := 0; p < k; p++ {
				distance := getDistance(clusters[p], data[i][j])
				if distance < dis {
					cluster, dis = p, distance
					if dis == 0 {
						break
					}
				}
			}
			data[i][j].cluster = cluster
			mutex.Lock()
			clustersSum[cluster].red += data[i][j].red
			clustersSum[cluster].green += data[i][j].green
			clustersSum[cluster].blue += data[i][j].blue
			clustersSum[cluster].cluster++
			mutex.Unlock()
		}
	}
}

func setClusters() {
	var wg sync.WaitGroup
	wg.Add(width)
	for x := 0; x < width; x++ {
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			for j := 0; j < height; j++ {
				cluster, dis := -1, math.MaxInt64
				for p := 0; p < k; p++ {
					distance := getDistance(clusters[p], data[i][j])
					if distance < dis {
						cluster, dis = p, distance
						if dis == 0 {
							break
						}
					}
				}
				data[i][j].cluster = cluster
				mutex.Lock()
				clustersSum[cluster].red += data[i][j].red
				clustersSum[cluster].green += data[i][j].green
				clustersSum[cluster].blue += data[i][j].blue
				clustersSum[cluster].cluster++
				mutex.Unlock()
			}
		}(x, &wg)
	}
	wg.Wait()
}

func setNewClusters() int {
	res := 0
	ch := make(chan int, k)
	for j := 0; j < k; j++ {
		go func(i int, c chan int) {
			ravg, gavg, bavg := float64(clustersSum[i].red/clustersSum[i].cluster), float64(clustersSum[i].green/clustersSum[i].cluster), float64(clustersSum[i].blue/clustersSum[i].cluster)
			rdiff, gdiff, bdiff := math.Abs(float64(clusters[i].red)-ravg), math.Abs(float64(clusters[i].green)-gavg), math.Abs(float64(clusters[i].blue)-bavg)
			if rdiff/float64(clusters[i].red) > threshold || gdiff/float64(clusters[i].green) > threshold || bdiff/float64(clusters[i].blue) > threshold {
				clusters[i].red, clusters[i].green, clusters[i].blue = int(ravg), int(gavg), int(bavg)
				c <- 0
			} else {
				c <- 1
			}
		}(j, ch)
		res += <-ch
	}
	close(ch)
	return res
}

func initClusters() {
	rand.Seed(time.Now().UnixNano())
	clusters, clustersSum = make([]datum, k), make([]datum, k)
	for j := 0; j < k; j++ {
		go func(i int) {
			widx, hidx := rand.Intn(width-1), rand.Intn(height-1)
			clusters[i].red, clusters[i].green, clusters[i].blue, clusters[i].cluster = data[widx][hidx].red, data[widx][hidx].green, data[widx][hidx].blue, data[widx][hidx].cluster
			clustersSum[i].red, clustersSum[i].green, clustersSum[i].blue, clustersSum[i].cluster = data[widx][hidx].red, data[widx][hidx].green, data[widx][hidx].blue, 1
		}(j)
	}
	time.Sleep(time.Millisecond)
}

func read() {
	// Decode the JPEG data. If reading from file, create a reader with
	//
	reader, err := os.Open(path + file + ext)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()
	// reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	m, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	bounds := m.Bounds()

	// Calculate a 16-bin histogram for m's red, green, blue and alpha components.
	//
	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X. Looping over Y first and X second is more
	// likely to result in better memory access patterns than X first and Y second.
	width, height = bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y
	data = make([][]datum, width)
	for x := 0; x < width; x++ {
		data[x] = make([]datum, height)
		for y := 0; y < height; y++ {
			r, g, b, _ := m.At(x, y).RGBA()
			data[x][y].red, data[x][y].green, data[x][y].blue, data[x][y].cluster = int(r), int(g), int(b), -1
		}
	}
}

func dump(y int, img *image.NRGBA) {
	for x := 0; x < width; x++ {
		img.Set(x, y, color.RGBA{
			R: uint8(clusters[data[x][y].cluster].red >> 8),
			G: uint8(clusters[data[x][y].cluster].green >> 8),
			B: uint8(clusters[data[x][y].cluster].blue >> 8),
			A: 255,
		})
	}
}

func dump2(y int, img *image.NRGBA, wg *sync.WaitGroup) {
	defer wg.Done()
	for x := 0; x < width; x++ {
		img.Set(x, y, color.RGBA{
			R: uint8(clusters[data[x][y].cluster].red >> 8),
			G: uint8(clusters[data[x][y].cluster].green >> 8),
			B: uint8(clusters[data[x][y].cluster].blue >> 8),
			A: 255,
		})
	}
}

func write() {
	// Create a colored image of the given width and height.
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	time.Sleep(2 * time.Millisecond)
	//var wg sync.WaitGroup
	//wg.Add(height)
	for y := 0; y < height; y++ {
		//go dump2(y, img, &wg)
		dump(y, img)
	}
	//wg.Wait()
	_, timestamp := getTimeStamp(false)
	f, err := os.Create(path + file + "_" + timestamp + ext)
	if err != nil {
		log.Fatal(err)
	}

	// Specify the quality, between 0-100 Higher is better
	opt := jpeg.Options{
		Quality: 100,
	}

	if err := jpeg.Encode(f, img, &opt); err != nil {
		f.Close()
		log.Fatal(err)
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func getTimeStamp(f bool) (int64, string) {
	t := time.Now()
	if f {
		return t.UnixNano() / int64(time.Millisecond), ""
	}
	return 0, fmt.Sprintf("%d-%02d-%02dT%02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

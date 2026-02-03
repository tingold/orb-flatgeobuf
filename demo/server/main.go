package main

import (
	"bytes"
	"log"
	"net/http"
	"path/filepath"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	flatgeobuf "github.com/tingold/orb-flatgeobuf"
)

type City struct {
	Name       string
	Country    string
	Longitude  float64
	Latitude   float64
	Population int
	Capital    bool
}

var cities = []City{
	{"Tokyo", "Japan", 139.6917, 35.6895, 13960000, true},
	{"New York", "United States", -73.9857, 40.7484, 8336817, false},
	{"London", "United Kingdom", -0.1276, 51.5074, 8982000, true},
	{"Paris", "France", 2.3522, 48.8566, 2161000, true},
	{"Beijing", "China", 116.4074, 39.9042, 21540000, true},
	{"Moscow", "Russia", 37.6173, 55.7558, 12615000, true},
	{"São Paulo", "Brazil", -46.6333, -23.5505, 12300000, false},
	{"Mumbai", "India", 72.8777, 19.0760, 12400000, false},
	{"Los Angeles", "United States", -118.2437, 34.0522, 3971883, false},
	{"Shanghai", "China", 121.4737, 31.2304, 24870000, false},
	{"Istanbul", "Turkey", 28.9784, 41.0082, 15520000, false},
	{"Buenos Aires", "Argentina", -58.3816, -34.6037, 3075646, true},
	{"Cairo", "Egypt", 31.2357, 30.0444, 10230000, true},
	{"Sydney", "Australia", 151.2093, -33.8688, 5312000, false},
	{"Berlin", "Germany", 13.4050, 52.5200, 3669491, true},
}

func main() {
	// Create GeoJSON FeatureCollection
	fc := geojson.NewFeatureCollection()

	for _, city := range cities {
		f := geojson.NewFeature(orb.Point{city.Longitude, city.Latitude})
		f.Properties = geojson.Properties{
			"name":       city.Name,
			"country":    city.Country,
			"population": city.Population,
			"capital":    city.Capital,
		}
		fc.Append(f)
	}

	// Convert to FlatGeobuf
	var buf bytes.Buffer
	opts := &flatgeobuf.Options{
		Name:         "world_cities",
		Description:  "Major world cities",
		IncludeIndex: false,
		CRS:          flatgeobuf.WGS84(),
	}

	err := flatgeobuf.WriteFeatures(&buf, fc, opts)
	if err != nil {
		log.Fatalf("Failed to create FlatGeobuf: %v", err)
	}

	flatgeobufData := buf.Bytes()

	// Get the directory of the client files (one level up from server)
	clientDir := filepath.Join("..", "client")

	// Create a custom handler that checks for data endpoints first
	fs := http.FileServer(http.Dir(clientDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle FlatGeobuf data endpoint
		if r.URL.Path == "/data.fgb" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(flatgeobufData)
			return
		}
		// Serve static files for everything else
		fs.ServeHTTP(w, r)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("Serving client files from:", clientDir)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

# orb-flatgeobuf

FlatGeobuf format support for the [orb](https://github.com/paulmach/orb) geometry library.

This package provides reading and writing of [FlatGeobuf](https://flatgeobuf.org/) files with orb geometry types and GeoJSON features with properties.

[![CI](https://github.com/tingold/orb-flatgeobuf/actions/workflows/ci.yml/badge.svg)](https://github.com/tingold/orb-flatgeobuf/actions/workflows/ci.yml)

## Features

- **Write FlatGeobuf files** from orb geometries or GeoJSON FeatureCollections
- **Read FlatGeobuf files** into orb geometries or GeoJSON FeatureCollections
- **Spatial queries** using the built-in Hilbert R-tree index
- **Property support** with automatic type inference from GeoJSON properties
- **All orb geometry types** supported: Point, MultiPoint, LineString, MultiLineString, Polygon, MultiPolygon, Collection, Ring, Bound

## Performance

FlatGeobuf offers significant performance advantages over GeoJSON, particularly for coordinate-heavy geometries and spatial queries.

### Benchmark Results

#### Payload Size Comparison

| Dataset | GeoJSON | FlatGeobuf | Savings |
|---------|---------|------------|---------|
| 1K Points | 116 KB | 164 KB | -41% (overhead) |
| 1K Points + Properties | 294 KB | 282 KB | **4%** |
| 1K LineStrings (10 vertices each) | 472 KB | 305 KB | **35%** |
| 1K Simple Polygons | 277 KB | 227 KB | **18%** |
| 1K Complex Polygons (32 vertices) | 1.3 MB | 664 KB | **51%** |

#### Serialization Speed

| Operation | GeoJSON | FlatGeobuf | Speedup |
|-----------|---------|------------|---------|
| Serialize 1K Points | 3.3 ms | 1.0 ms | **3x faster** |
| Serialize 10K Points | 32 ms | 11 ms | **3x faster** |

#### Deserialization Speed

| Operation | GeoJSON | FlatGeobuf | Speedup |
|-----------|---------|------------|---------|
| Deserialize 1K Points | 8.3 ms | 0.5 ms | **18x faster** |
| Deserialize 10K Points | 83 ms | 4.8 ms | **17x faster** |

#### Spatial Query Performance

This is where FlatGeobuf truly shines - the built-in R-tree index enables O(log n) spatial queries:

| Operation | GeoJSON (parse + scan) | FlatGeobuf (indexed) | Speedup |
|-----------|------------------------|----------------------|---------|
| Query 10K Points | 85 ms | 0.5 ms | **175x faster** |
| Query 50K Points | 424 ms | 2.3 ms | **186x faster** |

### Key Takeaways

- **Complex geometries**: FlatGeobuf provides 35-51% smaller payloads for coordinate-heavy data
- **Simple points**: GeoJSON may be smaller due to FlatGeobuf's header overhead
- **Read performance**: FlatGeobuf is consistently 17-18x faster to deserialize
- **Spatial queries**: The built-in R-tree index makes FlatGeobuf 175x+ faster for spatial lookups
- **Memory efficiency**: FlatGeobuf uses ~6x less memory and allocations during parsing

Run the benchmarks yourself:

```bash
go test -bench=. -benchmem
go test -v -run "TestPerformanceSummary"
```

## Installation

```bash
go get github.com/tingold/orb-flatgeobuf
```

## Usage

### Writing FlatGeobuf Files

#### Write Geometries

```go
package main

import (
    "os"
    
    "github.com/paulmach/orb"
    flatgeobuf "github.com/tingold/orb-flatgeobuf"
)

func main() {
    geometries := []orb.Geometry{
        orb.Point{-122.4194, 37.7749},
        orb.Point{-73.9857, 40.7484},
        orb.Point{-87.6298, 41.8781},
    }
    
    file, _ := os.Create("points.fgb")
    defer file.Close()
    
    opts := &flatgeobuf.Options{
        Name:         "cities",
        Description:  "Major US cities",
        IncludeIndex: true,
        CRS:          flatgeobuf.WGS84(),
    }
    
    err := flatgeobuf.Write(file, geometries, opts)
    if err != nil {
        panic(err)
    }
}
```

#### Write GeoJSON Features with Properties

```go
package main

import (
    "os"
    
    "github.com/paulmach/orb"
    "github.com/paulmach/orb/geojson"
    flatgeobuf "github.com/tingold/orb-flatgeobuf"
)

func main() {
    fc := geojson.NewFeatureCollection()
    
    f1 := geojson.NewFeature(orb.Point{-122.4194, 37.7749})
    f1.Properties = geojson.Properties{
        "name":       "San Francisco",
        "population": 884363,
        "capital":    false,
    }
    fc.Append(f1)
    
    f2 := geojson.NewFeature(orb.Point{-73.9857, 40.7484})
    f2.Properties = geojson.Properties{
        "name":       "New York",
        "population": 8336817,
        "capital":    false,
    }
    fc.Append(f2)
    
    file, _ := os.Create("cities.fgb")
    defer file.Close()
    
    err := flatgeobuf.WriteFeatures(file, fc, nil)
    if err != nil {
        panic(err)
    }
}
```

### Reading FlatGeobuf Files

#### Read All Features

```go
package main

import (
    "fmt"
    
    flatgeobuf "github.com/tingold/orb-flatgeobuf"
)

func main() {
    reader, err := flatgeobuf.NewReader("cities.fgb")
    if err != nil {
        panic(err)
    }
    defer reader.Close()
    
    // Get file metadata
    header := reader.Header()
    fmt.Printf("Layer: %s\n", header.Name)
    fmt.Printf("Geometry Type: %s\n", header.GeometryType)
    fmt.Printf("Feature Count: %d\n", header.FeaturesCount)
    fmt.Printf("Has Index: %v\n", header.HasIndex)
    
    // Print column schema
    for _, col := range header.Columns {
        fmt.Printf("Column: %s (%s)\n", col.Name, col.Type)
    }
    
    // Read all features
    fc, err := reader.ReadAll()
    if err != nil {
        panic(err)
    }
    
    for _, f := range fc.Features {
        fmt.Printf("Geometry: %v\n", f.Geometry)
        fmt.Printf("Properties: %v\n", f.Properties)
    }
}
```

#### Spatial Query with Index

```go
package main

import (
    "fmt"
    
    "github.com/paulmach/orb"
    flatgeobuf "github.com/tingold/orb-flatgeobuf"
)

func main() {
    reader, err := flatgeobuf.NewReader("cities.fgb")
    if err != nil {
        panic(err)
    }
    defer reader.Close()
    
    // Query features in California bounding box
    bounds := orb.Bound{
        Min: orb.Point{-124.409591, 32.534156},
        Max: orb.Point{-114.131211, 42.009518},
    }
    
    fc, err := reader.Search(bounds)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d features in bounds\n", len(fc.Features))
}
```

#### Read Geometries Only

```go
// Read just the geometries (no properties)
geometries, err := reader.ReadGeometries()

// Or search for geometries
geometries, err := reader.SearchGeometries(bounds)
```

### Reading from Byte Data

```go
data, _ := os.ReadFile("file.fgb")

reader, err := flatgeobuf.NewReaderFromData(data)
if err != nil {
    panic(err)
}
defer reader.Close()
```

## API Reference

### Types

#### Options

```go
type Options struct {
    Name         string  // Layer name
    Description  string  // Layer description
    IncludeIndex bool    // Include spatial index (default: true)
    CRS          *CRS    // Coordinate reference system
}
```

#### CRS

```go
type CRS struct {
    Code        int     // EPSG code (e.g., 4326 for WGS84)
    Name        string  // CRS name
    Description string  // CRS description
    WKT         string  // Well-Known Text representation
}

// Helper function for WGS84
func WGS84() *CRS
```

#### Header

```go
type Header struct {
    Name          string       // Layer name
    Description   string       // Layer description
    GeometryType  string       // "Point", "Polygon", "Unknown", etc.
    FeaturesCount uint64       // Number of features
    Envelope      [4]float64   // Bounding box [minX, minY, maxX, maxY]
    CRS           *CRS         // Coordinate reference system
    HasIndex      bool         // Whether file has spatial index
    Columns       []ColumnInfo // Property schema
}
```

#### ColumnInfo

```go
type ColumnInfo struct {
    Name        string  // Column name
    Type        string  // "Bool", "Int", "Long", "Double", "String", "Json"
    Title       string  // Human-readable title
    Description string  // Column description
    Nullable    bool    // Whether column can be null
}
```

### Writing Functions

```go
// Write geometries to FlatGeobuf format
func Write(w io.Writer, geometries []orb.Geometry, opts *Options) error

// Write a FeatureCollection to FlatGeobuf format
func WriteFeatures(w io.Writer, fc *geojson.FeatureCollection, opts *Options) error

// Write a single feature to FlatGeobuf format
func WriteFeature(w io.Writer, f *geojson.Feature, opts *Options) error
```

### Reader

```go
// Create a reader from a file path (memory-mapped)
func NewReader(path string) (*Reader, error)

// Create a reader from byte data
func NewReaderFromData(data []byte) (*Reader, error)

// Get file metadata
func (r *Reader) Header() *Header

// Read all features as a FeatureCollection
func (r *Reader) ReadAll() (*geojson.FeatureCollection, error)

// Read all geometries without properties
func (r *Reader) ReadGeometries() ([]orb.Geometry, error)

// Spatial query using the built-in index
func (r *Reader) Search(bounds orb.Bound) (*geojson.FeatureCollection, error)

// Spatial query returning only geometries
func (r *Reader) SearchGeometries(bounds orb.Bound) ([]orb.Geometry, error)

// Release resources
func (r *Reader) Close() error
```

## Supported Geometry Types

| orb Type | FlatGeobuf Type |
|----------|-----------------|
| `orb.Point` | Point |
| `orb.MultiPoint` | MultiPoint |
| `orb.LineString` | LineString |
| `orb.MultiLineString` | MultiLineString |
| `orb.Ring` | Polygon |
| `orb.Polygon` | Polygon |
| `orb.MultiPolygon` | MultiPolygon |
| `orb.Collection` | GeometryCollection |
| `orb.Bound` | Polygon (rectangle) |

## Property Type Mapping

| Go Type | FlatGeobuf Column Type |
|---------|----------------------|
| `bool` | Bool |
| `int`, `int32` | Int |
| `int64` | Long |
| `float32` | Float |
| `float64` | Double |
| `string` | String |
| `map[string]interface{}` | Json |
| `[]interface{}` | Json |

## Related Projects

- [orb](https://github.com/paulmach/orb) - Core geometry types
- [orb-predicates](https://github.com/tingold/orb-predicates) - Geometry predicates (Contains, Intersects, etc.)
- [orb-operations](https://github.com/tingold/orb-operations) - Geometry operations (Union, Intersection, etc.)
- [FlatGeobuf](https://flatgeobuf.org/) - FlatGeobuf specification

## License

MIT License - see [LICENSE](LICENSE) for details.

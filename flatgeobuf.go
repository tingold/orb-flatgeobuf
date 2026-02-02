// Package flatgeobuf provides FlatGeobuf format support for the orb geometry library.
// It enables reading and writing FlatGeobuf files with orb.Geometry types and
// geojson.Feature/FeatureCollection with properties.
package flatgeobuf

import (
	"errors"
)

// Common errors returned by this package.
var (
	ErrNilGeometry      = errors.New("flatgeobuf: nil geometry")
	ErrUnsupportedType  = errors.New("flatgeobuf: unsupported geometry type")
	ErrInvalidData      = errors.New("flatgeobuf: invalid data")
	ErrNoIndex          = errors.New("flatgeobuf: file has no spatial index")
	ErrInvalidColumn    = errors.New("flatgeobuf: invalid column type")
	ErrPropertyMismatch = errors.New("flatgeobuf: property type mismatch")
)

// CRS represents a coordinate reference system.
type CRS struct {
	Code        int    // EPSG code (e.g., 4326 for WGS84)
	Name        string // CRS name
	Description string // CRS description
	WKT         string // Well-Known Text representation
}

// WGS84 returns the standard WGS84 CRS (EPSG:4326).
func WGS84() *CRS {
	return &CRS{
		Code: 4326,
		Name: "WGS 84",
	}
}

// Options configures FlatGeobuf writing.
type Options struct {
	Name         string // Layer name
	Description  string // Layer description
	IncludeIndex bool   // Include spatial index (default: true)
	CRS          *CRS   // Coordinate reference system (optional)
}

// DefaultOptions returns default options for writing FlatGeobuf files.
func DefaultOptions() *Options {
	return &Options{
		IncludeIndex: true,
	}
}

// ColumnInfo describes a property column in a FlatGeobuf file.
type ColumnInfo struct {
	Name        string // Column name
	Type        string // Column type ("Bool", "Int", "Long", "Double", "String", "Json", etc.)
	Title       string // Column title (human-readable)
	Description string // Column description
	Nullable    bool   // Whether the column can contain null values
}

// Header contains metadata about a FlatGeobuf file.
type Header struct {
	Name          string       // Layer name
	Description   string       // Layer description
	GeometryType  string       // Geometry type ("Point", "Polygon", "Unknown", etc.)
	FeaturesCount uint64       // Number of features in the file
	Envelope      [4]float64   // Bounding box [minX, minY, maxX, maxY]
	CRS           *CRS         // Coordinate reference system
	HasIndex      bool         // Whether the file has a spatial index
	Columns       []ColumnInfo // Property column schema
}

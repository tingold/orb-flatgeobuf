package flatgeobuf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func TestNewReaderFromData_Invalid(t *testing.T) {
	// Invalid data (not a FlatGeobuf file)
	_, err := NewReaderFromData([]byte("not a flatgeobuf"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestNewReaderFromData_Empty(t *testing.T) {
	_, err := NewReaderFromData([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestRoundTrip_Points(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.fgb")

	// Write some points
	fc := geojson.NewFeatureCollection()
	for i := 0; i < 10; i++ {
		f := geojson.NewFeature(orb.Point{float64(i), float64(i * 2)})
		f.Properties = geojson.Properties{
			"index": i,
			"name":  "point",
		}
		fc.Append(f)
	}

	opts := &Options{
		Name:         "test_points",
		IncludeIndex: true,
	}

	// Write to file
	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, opts)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Read back
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	// Check header
	header := reader.Header()
	if header == nil {
		t.Fatal("expected non-nil header")
	}

	if header.Name != "test_points" {
		t.Errorf("expected name 'test_points', got %q", header.Name)
	}

	if header.GeometryType != "Point" {
		t.Errorf("expected geometry type 'Point', got %q", header.GeometryType)
	}

	if !header.HasIndex {
		t.Error("expected HasIndex to be true")
	}
}

func TestRoundTrip_Polygons(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_polygons.fgb")

	// Write polygons
	fc := geojson.NewFeatureCollection()

	poly1 := orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}}
	f1 := geojson.NewFeature(poly1)
	f1.Properties = geojson.Properties{"name": "square1"}
	fc.Append(f1)

	poly2 := orb.Polygon{{{20, 20}, {30, 20}, {30, 30}, {20, 30}, {20, 20}}}
	f2 := geojson.NewFeature(poly2)
	f2.Properties = geojson.Properties{"name": "square2"}
	fc.Append(f2)

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, nil)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Read back
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	header := reader.Header()
	if header.GeometryType != "Polygon" {
		t.Errorf("expected geometry type 'Polygon', got %q", header.GeometryType)
	}
}

func TestRoundTrip_Search(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_search.fgb")

	// Write points in a grid
	fc := geojson.NewFeatureCollection()
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			f := geojson.NewFeature(orb.Point{float64(x), float64(y)})
			f.Properties = geojson.Properties{
				"x": x,
				"y": y,
			}
			fc.Append(f)
		}
	}

	opts := &Options{
		IncludeIndex: true,
	}

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, opts)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Read and search
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	// Search a small area
	bounds := orb.Bound{
		Min: orb.Point{2, 2},
		Max: orb.Point{4, 4},
	}

	results, err := reader.Search(bounds)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find points in the 2-4 range (inclusive on min, exclusive behavior may vary)
	if len(results.Features) == 0 {
		t.Error("expected some results from search")
	}
}

func TestSearch_NoIndex(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_no_index.fgb")

	// Write without index
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 2}))

	opts := &Options{
		IncludeIndex: false,
	}

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, opts)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Try to search
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	_, err = reader.Search(orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{10, 10}})
	if err != ErrNoIndex {
		t.Errorf("expected ErrNoIndex, got %v", err)
	}
}

func TestReadGeometries(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_geoms.fgb")

	// Write
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 2}))
	fc.Append(geojson.NewFeature(orb.Point{3, 4}))
	fc.Append(geojson.NewFeature(orb.Point{5, 6}))

	opts := &Options{IncludeIndex: true}

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, opts)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Read geometries
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	geoms, err := reader.ReadGeometries()
	if err != nil {
		t.Fatalf("ReadGeometries failed: %v", err)
	}

	if len(geoms) != 3 {
		t.Errorf("expected 3 geometries, got %d", len(geoms))
	}
}

func TestSearchGeometries(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_search_geoms.fgb")

	// Write
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 1}))
	fc.Append(geojson.NewFeature(orb.Point{5, 5}))
	fc.Append(geojson.NewFeature(orb.Point{9, 9}))

	opts := &Options{IncludeIndex: true}

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, opts)
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Search geometries
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	bounds := orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{6, 6}}
	geoms, err := reader.SearchGeometries(bounds)
	if err != nil {
		t.Fatalf("SearchGeometries failed: %v", err)
	}

	// Should find points at (1,1) and (5,5)
	if len(geoms) < 1 {
		t.Errorf("expected at least 1 geometry, got %d", len(geoms))
	}
}

func TestReader_Close(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_close.fgb")

	// Write
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 2}))

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, &Options{IncludeIndex: true})
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Open and close
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestHeader_ColumnInfo(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_columns.fgb")

	// Write with properties
	fc := geojson.NewFeatureCollection()
	f := geojson.NewFeature(orb.Point{1, 2})
	f.Properties = geojson.Properties{
		"name":   "test",
		"value":  42,
		"active": true,
		"score":  3.14,
	}
	fc.Append(f)

	file, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	err = WriteFeatures(file, fc, &Options{IncludeIndex: true})
	_ = file.Close()
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	// Read header
	reader, err := NewReader(tmpFile)
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	header := reader.Header()
	if header == nil {
		t.Fatal("expected non-nil header")
	}

	if len(header.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(header.Columns))
	}

	// Check column types
	columnMap := make(map[string]string)
	for _, col := range header.Columns {
		columnMap[col.Name] = col.Type
	}

	if columnMap["name"] != "String" {
		t.Errorf("expected name column to be String, got %q", columnMap["name"])
	}
}

func TestNewReader_NonExistent(t *testing.T) {
	_, err := NewReader("/nonexistent/path/to/file.fgb")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

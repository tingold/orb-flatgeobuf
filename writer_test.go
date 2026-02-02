package flatgeobuf

import (
	"bytes"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func TestWrite_Points(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Point{1, 2},
		orb.Point{3, 4},
		orb.Point{5, 6},
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check magic bytes
	data := buf.Bytes()
	if len(data) < 8 {
		t.Fatal("output too short")
	}

	expectedMagic := []byte{0x66, 0x67, 0x62, 0x03, 0x66, 0x67, 0x62, 0x00}
	for i, b := range expectedMagic {
		if data[i] != b {
			t.Errorf("magic byte %d: expected 0x%02x, got 0x%02x", i, b, data[i])
		}
	}
}

func TestWrite_LineStrings(t *testing.T) {
	geometries := []orb.Geometry{
		orb.LineString{{0, 0}, {1, 1}, {2, 2}},
		orb.LineString{{5, 5}, {6, 6}},
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWrite_Polygons(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Polygon{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
		orb.Polygon{{{20, 20}, {30, 20}, {30, 30}, {20, 30}, {20, 20}}},
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWrite_MixedGeometries(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Point{1, 2},
		orb.LineString{{0, 0}, {1, 1}},
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, nil)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Mixed geometries should result in Unknown geometry type in header
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWrite_EmptyGeometries(t *testing.T) {
	err := Write(&bytes.Buffer{}, []orb.Geometry{}, nil)
	if err != ErrNilGeometry {
		t.Errorf("expected ErrNilGeometry, got %v", err)
	}
}

func TestWrite_WithOptions(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Point{1, 2},
	}

	opts := &Options{
		Name:         "test_layer",
		Description:  "A test layer",
		IncludeIndex: true,
		CRS:          WGS84(),
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWrite_NoIndex(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Point{1, 2},
		orb.Point{3, 4},
	}

	opts := &Options{
		IncludeIndex: false,
	}

	var buf bytes.Buffer
	err := Write(&buf, geometries, opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWriteFeatures_Simple(t *testing.T) {
	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(orb.Point{1, 2}))
	fc.Append(geojson.NewFeature(orb.Point{3, 4}))

	var buf bytes.Buffer
	err := WriteFeatures(&buf, fc, nil)
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWriteFeatures_WithProperties(t *testing.T) {
	fc := geojson.NewFeatureCollection()

	f1 := geojson.NewFeature(orb.Point{1, 2})
	f1.Properties = geojson.Properties{
		"name":   "Point A",
		"value":  42,
		"active": true,
	}
	fc.Append(f1)

	f2 := geojson.NewFeature(orb.Point{3, 4})
	f2.Properties = geojson.Properties{
		"name":   "Point B",
		"value":  100,
		"active": false,
	}
	fc.Append(f2)

	var buf bytes.Buffer
	err := WriteFeatures(&buf, fc, nil)
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWriteFeatures_NilCollection(t *testing.T) {
	err := WriteFeatures(&bytes.Buffer{}, nil, nil)
	if err != ErrNilGeometry {
		t.Errorf("expected ErrNilGeometry, got %v", err)
	}
}

func TestWriteFeatures_EmptyCollection(t *testing.T) {
	fc := geojson.NewFeatureCollection()
	err := WriteFeatures(&bytes.Buffer{}, fc, nil)
	if err != ErrNilGeometry {
		t.Errorf("expected ErrNilGeometry, got %v", err)
	}
}

func TestWriteFeature_Single(t *testing.T) {
	f := geojson.NewFeature(orb.Point{1, 2})
	f.Properties = geojson.Properties{"name": "test"}

	var buf bytes.Buffer
	err := WriteFeature(&buf, f, nil)
	if err != nil {
		t.Fatalf("WriteFeature failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestWriteFeature_Nil(t *testing.T) {
	err := WriteFeature(&bytes.Buffer{}, nil, nil)
	if err != ErrNilGeometry {
		t.Errorf("expected ErrNilGeometry, got %v", err)
	}
}

func TestWriteFeatures_ComplexGeometries(t *testing.T) {
	fc := geojson.NewFeatureCollection()

	// Polygon with hole
	poly := orb.Polygon{
		{{0, 0}, {100, 0}, {100, 100}, {0, 100}, {0, 0}},
		{{20, 20}, {80, 20}, {80, 80}, {20, 80}, {20, 20}},
	}
	f1 := geojson.NewFeature(poly)
	f1.Properties = geojson.Properties{"type": "polygon_with_hole"}
	fc.Append(f1)

	// MultiPolygon
	mpoly := orb.MultiPolygon{
		{{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}},
		{{{50, 50}, {60, 50}, {60, 60}, {50, 60}, {50, 50}}},
	}
	f2 := geojson.NewFeature(mpoly)
	f2.Properties = geojson.Properties{"type": "multipolygon"}
	fc.Append(f2)

	var buf bytes.Buffer
	err := WriteFeatures(&buf, fc, nil)
	if err != nil {
		t.Fatalf("WriteFeatures failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts == nil {
		t.Fatal("expected non-nil options")
	}

	if !opts.IncludeIndex {
		t.Error("expected IncludeIndex to be true by default")
	}
}

func TestWGS84(t *testing.T) {
	crs := WGS84()

	if crs == nil {
		t.Fatal("expected non-nil CRS")
	}

	if crs.Code != 4326 {
		t.Errorf("expected code 4326, got %d", crs.Code)
	}

	if crs.Name != "WGS 84" {
		t.Errorf("expected name 'WGS 84', got %q", crs.Name)
	}
}

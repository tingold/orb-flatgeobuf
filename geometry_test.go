package flatgeobuf

import (
	"testing"

	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/paulmach/orb"
)

func TestOrbToFGBGeometryType(t *testing.T) {
	tests := []struct {
		name     string
		geom     orb.Geometry
		expected flattypes.GeometryType
	}{
		{"Point", orb.Point{1, 2}, flattypes.GeometryTypePoint},
		{"MultiPoint", orb.MultiPoint{{1, 2}, {3, 4}}, flattypes.GeometryTypeMultiPoint},
		{"LineString", orb.LineString{{0, 0}, {1, 1}}, flattypes.GeometryTypeLineString},
		{"MultiLineString", orb.MultiLineString{{{0, 0}, {1, 1}}}, flattypes.GeometryTypeMultiLineString},
		{"Ring", orb.Ring{{0, 0}, {1, 0}, {1, 1}, {0, 0}}, flattypes.GeometryTypePolygon},
		{"Polygon", orb.Polygon{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}}, flattypes.GeometryTypePolygon},
		{"MultiPolygon", orb.MultiPolygon{{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}}}, flattypes.GeometryTypeMultiPolygon},
		{"Collection", orb.Collection{orb.Point{1, 2}}, flattypes.GeometryTypeGeometryCollection},
		{"Bound", orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{1, 1}}, flattypes.GeometryTypePolygon},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orbToFGBGeometryType(tt.geom)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGeometryToFGB_Point(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	point := orb.Point{1.5, 2.5}

	geom := geometryToFGB(point, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_LineString(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	ls := orb.LineString{{0, 0}, {1, 1}, {2, 2}}

	geom := geometryToFGB(ls, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_Polygon(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	poly := orb.Polygon{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}},
		{{2, 2}, {8, 2}, {8, 8}, {2, 8}, {2, 2}}, // hole
	}

	geom := geometryToFGB(poly, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_MultiPolygon(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	mp := orb.MultiPolygon{
		{{{0, 0}, {5, 0}, {5, 5}, {0, 5}, {0, 0}}},
		{{{10, 10}, {15, 10}, {15, 15}, {10, 15}, {10, 10}}},
	}

	geom := geometryToFGB(mp, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_Collection(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	coll := orb.Collection{
		orb.Point{1, 2},
		orb.LineString{{0, 0}, {1, 1}},
	}

	geom := geometryToFGB(coll, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_Bound(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	bound := orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{10, 10}}

	geom := geometryToFGB(bound, builder)
	if geom == nil {
		t.Fatal("expected non-nil geometry")
	}
}

func TestGeometryToFGB_Nil(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)

	geom := geometryToFGB(nil, builder)
	if geom != nil {
		t.Error("expected nil geometry for nil input")
	}
}

func TestLineStringToXY(t *testing.T) {
	ls := orb.LineString{{1, 2}, {3, 4}, {5, 6}}
	xy := lineStringToXY(ls)

	expected := []float64{1, 2, 3, 4, 5, 6}
	if len(xy) != len(expected) {
		t.Fatalf("expected %d coordinates, got %d", len(expected), len(xy))
	}

	for i, v := range expected {
		if xy[i] != v {
			t.Errorf("at index %d: expected %f, got %f", i, v, xy[i])
		}
	}
}

func TestPolygonToXYEnds(t *testing.T) {
	poly := orb.Polygon{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}}, // 5 points
		{{2, 2}, {8, 2}, {8, 8}, {2, 8}, {2, 2}},     // 5 points
	}

	xy, ends := polygonToXYEnds(poly)

	if len(xy) != 20 { // 10 points * 2 coordinates
		t.Errorf("expected 20 coordinates, got %d", len(xy))
	}

	if len(ends) != 2 {
		t.Fatalf("expected 2 ends, got %d", len(ends))
	}

	if ends[0] != 5 {
		t.Errorf("expected first end to be 5, got %d", ends[0])
	}

	if ends[1] != 10 {
		t.Errorf("expected second end to be 10, got %d", ends[1])
	}
}

func TestBoundToPolygon(t *testing.T) {
	bound := orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{10, 10}}
	poly := boundToPolygon(bound)

	if len(poly) != 1 {
		t.Fatalf("expected 1 ring, got %d", len(poly))
	}

	ring := poly[0]
	if len(ring) != 5 {
		t.Errorf("expected 5 points in ring, got %d", len(ring))
	}

	// Check corners
	expectedCorners := []orb.Point{
		{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0},
	}

	for i, expected := range expectedCorners {
		if ring[i] != expected {
			t.Errorf("corner %d: expected %v, got %v", i, expected, ring[i])
		}
	}
}

func TestComputeBoundingBox(t *testing.T) {
	tests := []struct {
		name     string
		geom     orb.Geometry
		expected [4]float64
	}{
		{
			"Point",
			orb.Point{5, 10},
			[4]float64{5, 10, 5, 10},
		},
		{
			"LineString",
			orb.LineString{{0, 0}, {10, 10}},
			[4]float64{0, 0, 10, 10},
		},
		{
			"Polygon",
			orb.Polygon{{{0, 0}, {20, 0}, {20, 30}, {0, 30}, {0, 0}}},
			[4]float64{0, 0, 20, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bbox := computeBoundingBox(tt.geom)
			if bbox != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, bbox)
			}
		})
	}
}

func TestComputeCollectionBoundingBox(t *testing.T) {
	geometries := []orb.Geometry{
		orb.Point{5, 5},
		orb.Point{15, 20},
		orb.LineString{{0, 0}, {10, 10}},
	}

	bbox := computeCollectionBoundingBox(geometries)
	expected := [4]float64{0, 0, 15, 20}

	if bbox != expected {
		t.Errorf("expected %v, got %v", expected, bbox)
	}
}

func TestComputeCollectionBoundingBox_Empty(t *testing.T) {
	bbox := computeCollectionBoundingBox([]orb.Geometry{})
	expected := [4]float64{0, 0, 0, 0}

	if bbox != expected {
		t.Errorf("expected %v, got %v", expected, bbox)
	}
}

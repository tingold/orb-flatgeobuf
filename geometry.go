package flatgeobuf

import (
	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	"github.com/flatgeobuf/flatgeobuf/src/go/writer"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/paulmach/orb"
)

// orbToFGBGeometryType converts an orb.Geometry to its FlatGeobuf GeometryType.
func orbToFGBGeometryType(geom orb.Geometry) flattypes.GeometryType {
	switch geom.(type) {
	case orb.Point:
		return flattypes.GeometryTypePoint
	case orb.MultiPoint:
		return flattypes.GeometryTypeMultiPoint
	case orb.LineString:
		return flattypes.GeometryTypeLineString
	case orb.MultiLineString:
		return flattypes.GeometryTypeMultiLineString
	case orb.Ring:
		return flattypes.GeometryTypePolygon
	case orb.Polygon:
		return flattypes.GeometryTypePolygon
	case orb.MultiPolygon:
		return flattypes.GeometryTypeMultiPolygon
	case orb.Collection:
		return flattypes.GeometryTypeGeometryCollection
	case orb.Bound:
		return flattypes.GeometryTypePolygon
	default:
		return flattypes.GeometryTypeUnknown
	}
}

// geometryToFGB converts an orb.Geometry to a FlatGeobuf writer.Geometry.
func geometryToFGB(geom orb.Geometry, builder *flatbuffers.Builder) *writer.Geometry {
	if geom == nil {
		return nil
	}

	g := writer.NewGeometry(builder)

	switch v := geom.(type) {
	case orb.Point:
		g.SetType(flattypes.GeometryTypePoint)
		g.SetXY([]float64{v[0], v[1]})

	case orb.MultiPoint:
		g.SetType(flattypes.GeometryTypeMultiPoint)
		xy := make([]float64, 0, len(v)*2)
		for _, p := range v {
			xy = append(xy, p[0], p[1])
		}
		g.SetXY(xy)

	case orb.LineString:
		g.SetType(flattypes.GeometryTypeLineString)
		xy := lineStringToXY(v)
		g.SetXY(xy)

	case orb.MultiLineString:
		g.SetType(flattypes.GeometryTypeMultiLineString)
		xy, ends := multiLineStringToXYEnds(v)
		g.SetXY(xy)
		g.SetEnds(ends)

	case orb.Ring:
		g.SetType(flattypes.GeometryTypePolygon)
		xy := ringToXY(v)
		g.SetXY(xy)
		g.SetEnds([]uint32{uint32(len(v))})

	case orb.Polygon:
		g.SetType(flattypes.GeometryTypePolygon)
		xy, ends := polygonToXYEnds(v)
		g.SetXY(xy)
		g.SetEnds(ends)

	case orb.MultiPolygon:
		g.SetType(flattypes.GeometryTypeMultiPolygon)
		parts := make([]writer.Geometry, 0, len(v))
		for _, poly := range v {
			pg := writer.NewGeometry(builder)
			pg.SetType(flattypes.GeometryTypePolygon)
			xy, ends := polygonToXYEnds(poly)
			pg.SetXY(xy)
			pg.SetEnds(ends)
			parts = append(parts, *pg)
		}
		g.SetParts(parts)

	case orb.Collection:
		g.SetType(flattypes.GeometryTypeGeometryCollection)
		parts := make([]writer.Geometry, 0, len(v))
		for _, child := range v {
			childGeom := geometryToFGB(child, builder)
			if childGeom != nil {
				parts = append(parts, *childGeom)
			}
		}
		g.SetParts(parts)

	case orb.Bound:
		// Convert bound to a polygon (rectangle)
		g.SetType(flattypes.GeometryTypePolygon)
		poly := boundToPolygon(v)
		xy, ends := polygonToXYEnds(poly)
		g.SetXY(xy)
		g.SetEnds(ends)

	default:
		return nil
	}

	return g
}

// geometryFromFGB converts a FlatGeobuf flattypes.Geometry to an orb.Geometry.
func geometryFromFGB(fgbGeom *flattypes.Geometry) orb.Geometry {
	if fgbGeom == nil {
		return nil
	}

	geomType := fgbGeom.Type()

	switch geomType {
	case flattypes.GeometryTypePoint:
		return pointFromXY(fgbGeom)

	case flattypes.GeometryTypeMultiPoint:
		return multiPointFromXY(fgbGeom)

	case flattypes.GeometryTypeLineString:
		return lineStringFromXY(fgbGeom)

	case flattypes.GeometryTypeMultiLineString:
		return multiLineStringFromXYEnds(fgbGeom)

	case flattypes.GeometryTypePolygon:
		return polygonFromXYEnds(fgbGeom)

	case flattypes.GeometryTypeMultiPolygon:
		return multiPolygonFromParts(fgbGeom)

	case flattypes.GeometryTypeGeometryCollection:
		return collectionFromParts(fgbGeom)

	default:
		return nil
	}
}

// Helper functions for writing

func lineStringToXY(ls orb.LineString) []float64 {
	xy := make([]float64, 0, len(ls)*2)
	for _, p := range ls {
		xy = append(xy, p[0], p[1])
	}
	return xy
}

func ringToXY(r orb.Ring) []float64 {
	xy := make([]float64, 0, len(r)*2)
	for _, p := range r {
		xy = append(xy, p[0], p[1])
	}
	return xy
}

func multiLineStringToXYEnds(mls orb.MultiLineString) ([]float64, []uint32) {
	totalPoints := 0
	for _, ls := range mls {
		totalPoints += len(ls)
	}

	xy := make([]float64, 0, totalPoints*2)
	ends := make([]uint32, 0, len(mls))

	cumulative := uint32(0)
	for _, ls := range mls {
		for _, p := range ls {
			xy = append(xy, p[0], p[1])
		}
		cumulative += uint32(len(ls))
		ends = append(ends, cumulative)
	}

	return xy, ends
}

func polygonToXYEnds(poly orb.Polygon) ([]float64, []uint32) {
	totalPoints := 0
	for _, ring := range poly {
		totalPoints += len(ring)
	}

	xy := make([]float64, 0, totalPoints*2)
	ends := make([]uint32, 0, len(poly))

	cumulative := uint32(0)
	for _, ring := range poly {
		for _, p := range ring {
			xy = append(xy, p[0], p[1])
		}
		cumulative += uint32(len(ring))
		ends = append(ends, cumulative)
	}

	return xy, ends
}

func boundToPolygon(b orb.Bound) orb.Polygon {
	return orb.Polygon{
		orb.Ring{
			{b.Min[0], b.Min[1]},
			{b.Max[0], b.Min[1]},
			{b.Max[0], b.Max[1]},
			{b.Min[0], b.Max[1]},
			{b.Min[0], b.Min[1]},
		},
	}
}

// Helper functions for reading

func pointFromXY(fgbGeom *flattypes.Geometry) orb.Point {
	if fgbGeom.XyLength() < 2 {
		return orb.Point{}
	}
	return orb.Point{fgbGeom.Xy(0), fgbGeom.Xy(1)}
}

func multiPointFromXY(fgbGeom *flattypes.Geometry) orb.MultiPoint {
	xyLen := fgbGeom.XyLength()
	if xyLen < 2 {
		return orb.MultiPoint{}
	}

	numPoints := xyLen / 2
	mp := make(orb.MultiPoint, 0, numPoints)
	for i := 0; i < xyLen; i += 2 {
		mp = append(mp, orb.Point{fgbGeom.Xy(i), fgbGeom.Xy(i + 1)})
	}
	return mp
}

func lineStringFromXY(fgbGeom *flattypes.Geometry) orb.LineString {
	xyLen := fgbGeom.XyLength()
	if xyLen < 2 {
		return orb.LineString{}
	}

	numPoints := xyLen / 2
	ls := make(orb.LineString, 0, numPoints)
	for i := 0; i < xyLen; i += 2 {
		ls = append(ls, orb.Point{fgbGeom.Xy(i), fgbGeom.Xy(i + 1)})
	}
	return ls
}

func multiLineStringFromXYEnds(fgbGeom *flattypes.Geometry) orb.MultiLineString {
	xyLen := fgbGeom.XyLength()
	endsLen := fgbGeom.EndsLength()

	if xyLen < 2 || endsLen == 0 {
		// If no ends, treat as single linestring
		if xyLen >= 2 {
			return orb.MultiLineString{lineStringFromXY(fgbGeom)}
		}
		return orb.MultiLineString{}
	}

	mls := make(orb.MultiLineString, 0, endsLen)
	start := uint32(0)

	for i := 0; i < endsLen; i++ {
		end := fgbGeom.Ends(i)
		ls := make(orb.LineString, 0, end-start)

		for j := start; j < end; j++ {
			idx := int(j) * 2
			if idx+1 < xyLen {
				ls = append(ls, orb.Point{fgbGeom.Xy(idx), fgbGeom.Xy(idx + 1)})
			}
		}

		mls = append(mls, ls)
		start = end
	}

	return mls
}

func polygonFromXYEnds(fgbGeom *flattypes.Geometry) orb.Polygon {
	xyLen := fgbGeom.XyLength()
	endsLen := fgbGeom.EndsLength()

	if xyLen < 2 {
		return orb.Polygon{}
	}

	// If no ends array, treat all points as a single ring
	if endsLen == 0 {
		numPoints := xyLen / 2
		ring := make(orb.Ring, 0, numPoints)
		for i := 0; i < xyLen; i += 2 {
			ring = append(ring, orb.Point{fgbGeom.Xy(i), fgbGeom.Xy(i + 1)})
		}
		return orb.Polygon{ring}
	}

	poly := make(orb.Polygon, 0, endsLen)
	start := uint32(0)

	for i := 0; i < endsLen; i++ {
		end := fgbGeom.Ends(i)
		ring := make(orb.Ring, 0, end-start)

		for j := start; j < end; j++ {
			idx := int(j) * 2
			if idx+1 < xyLen {
				ring = append(ring, orb.Point{fgbGeom.Xy(idx), fgbGeom.Xy(idx + 1)})
			}
		}

		poly = append(poly, ring)
		start = end
	}

	return poly
}

func multiPolygonFromParts(fgbGeom *flattypes.Geometry) orb.MultiPolygon {
	partsLen := fgbGeom.PartsLength()
	if partsLen == 0 {
		// Fallback: treat as single polygon
		poly := polygonFromXYEnds(fgbGeom)
		if len(poly) > 0 {
			return orb.MultiPolygon{poly}
		}
		return orb.MultiPolygon{}
	}

	mp := make(orb.MultiPolygon, 0, partsLen)
	for i := 0; i < partsLen; i++ {
		var part flattypes.Geometry
		if fgbGeom.Parts(&part, i) {
			poly := polygonFromXYEnds(&part)
			if len(poly) > 0 {
				mp = append(mp, poly)
			}
		}
	}

	return mp
}

func collectionFromParts(fgbGeom *flattypes.Geometry) orb.Collection {
	partsLen := fgbGeom.PartsLength()
	if partsLen == 0 {
		return orb.Collection{}
	}

	coll := make(orb.Collection, 0, partsLen)
	for i := 0; i < partsLen; i++ {
		var part flattypes.Geometry
		if fgbGeom.Parts(&part, i) {
			geom := geometryFromFGB(&part)
			if geom != nil {
				coll = append(coll, geom)
			}
		}
	}

	return coll
}

// computeBoundingBox computes the bounding box of an orb.Geometry.
func computeBoundingBox(geom orb.Geometry) [4]float64 {
	bound := geom.Bound()
	return [4]float64{bound.Min[0], bound.Min[1], bound.Max[0], bound.Max[1]}
}

// computeCollectionBoundingBox computes the combined bounding box of multiple geometries.
func computeCollectionBoundingBox(geometries []orb.Geometry) [4]float64 {
	if len(geometries) == 0 {
		return [4]float64{0, 0, 0, 0}
	}

	minX, minY := geometries[0].Bound().Min[0], geometries[0].Bound().Min[1]
	maxX, maxY := geometries[0].Bound().Max[0], geometries[0].Bound().Max[1]

	for _, geom := range geometries[1:] {
		bound := geom.Bound()
		if bound.Min[0] < minX {
			minX = bound.Min[0]
		}
		if bound.Min[1] < minY {
			minY = bound.Min[1]
		}
		if bound.Max[0] > maxX {
			maxX = bound.Max[0]
		}
		if bound.Max[1] > maxY {
			maxY = bound.Max[1]
		}
	}

	return [4]float64{minX, minY, maxX, maxY}
}

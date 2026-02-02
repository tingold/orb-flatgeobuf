package flatgeobuf

import (
	"io"

	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	"github.com/flatgeobuf/flatgeobuf/src/go/writer"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// Write writes geometries to FlatGeobuf format.
// This is a convenience function for writing geometry-only data without properties.
func Write(w io.Writer, geometries []orb.Geometry, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	if len(geometries) == 0 {
		return ErrNilGeometry
	}

	// Determine geometry type from first geometry
	geomType := orbToFGBGeometryType(geometries[0])

	// Check if all geometries are the same type
	for _, g := range geometries[1:] {
		if orbToFGBGeometryType(g) != geomType {
			geomType = flattypes.GeometryTypeUnknown
			break
		}
	}

	// Create feature generator
	gen := &geometryFeatureGenerator{
		geometries: geometries,
		index:      0,
	}

	// Build and write (no features/columns for geometry-only write)
	return writeWithGenerator(w, gen, geomType, nil, nil, opts)
}

// WriteFeatures writes a FeatureCollection to FlatGeobuf format.
func WriteFeatures(w io.Writer, fc *geojson.FeatureCollection, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	if fc == nil || len(fc.Features) == 0 {
		return ErrNilGeometry
	}

	// Determine geometry type
	geomType := flattypes.GeometryTypeUnknown
	if fc.Features[0].Geometry != nil {
		geomType = orbToFGBGeometryType(fc.Features[0].Geometry)

		// Check if all geometries are the same type
		for _, f := range fc.Features[1:] {
			if f.Geometry != nil && orbToFGBGeometryType(f.Geometry) != geomType {
				geomType = flattypes.GeometryTypeUnknown
				break
			}
		}
	}

	// Get column names for property encoding
	columnNames := getColumnNames(fc.Features)

	// Create feature generator (columns will be inferred inside writeWithGenerator)
	gen := &featureCollectionGenerator{
		features:    fc.Features,
		columnNames: columnNames,
		index:       0,
	}

	return writeWithGenerator(w, gen, geomType, fc.Features, columnNames, opts)
}

// WriteFeature writes a single feature to FlatGeobuf format.
func WriteFeature(w io.Writer, f *geojson.Feature, opts *Options) error {
	if f == nil {
		return ErrNilGeometry
	}

	fc := &geojson.FeatureCollection{
		Features: []*geojson.Feature{f},
	}

	return WriteFeatures(w, fc, opts)
}

// writeWithGenerator handles the common writing logic.
func writeWithGenerator(
	w io.Writer,
	gen writer.FeatureGenerator,
	geomType flattypes.GeometryType,
	features []*geojson.Feature,
	columnNames []string,
	opts *Options,
) error {
	builder := flatbuffers.NewBuilder(4096)

	// Create header
	header := writer.NewHeader(builder)
	header.SetGeometryType(geomType)

	if opts.Name != "" {
		header.SetName(opts.Name)
	}
	if opts.Description != "" {
		header.SetDescription(opts.Description)
	}

	// Infer and set columns if we have features with properties
	var columns []*writer.Column
	var columnMap map[string]int
	if len(features) > 0 && len(columnNames) > 0 {
		columns = inferColumns(features, builder)
		columnMap = buildColumnMap(columns, columnNames)
		header.SetColumns(columns)
	}

	// Update the generator with column info if it's a feature collection generator
	if fcGen, ok := gen.(*featureCollectionGenerator); ok {
		fcGen.columns = columns
		fcGen.columnMap = columnMap
	}

	// Set CRS if provided
	if opts.CRS != nil {
		crs := writer.NewCrs(builder)
		crs.SetOrg("EPSG") // Default organization
		if opts.CRS.Code > 0 {
			crs.SetCode(int32(opts.CRS.Code))
		}
		if opts.CRS.Name != "" {
			crs.SetName(opts.CRS.Name)
		}
		if opts.CRS.Description != "" {
			crs.SetDescription(opts.CRS.Description)
		}
		// WKT can be stored in description if needed
		if opts.CRS.WKT != "" && opts.CRS.Description == "" {
			crs.SetDescription(opts.CRS.WKT)
		}
		header.SetCrs(crs)
	}

	// Create writer with or without index
	fgbWriter := writer.NewWriter(header, opts.IncludeIndex, gen, nil)

	// Write to destination
	_, err := fgbWriter.Write(w)
	return err
}

// geometryFeatureGenerator generates features from raw geometries.
type geometryFeatureGenerator struct {
	geometries []orb.Geometry
	index      int
}

func (g *geometryFeatureGenerator) Generate() *writer.Feature {
	if g.index >= len(g.geometries) {
		return nil
	}

	geom := g.geometries[g.index]
	g.index++

	if geom == nil {
		return g.Generate() // Skip nil geometries
	}

	builder := flatbuffers.NewBuilder(1024)
	fgbGeom := geometryToFGB(geom, builder)
	if fgbGeom == nil {
		return g.Generate() // Skip unsupported geometries
	}

	feature := writer.NewFeature(builder)
	feature.SetGeometry(fgbGeom)

	return feature
}

// featureCollectionGenerator generates features from a FeatureCollection.
type featureCollectionGenerator struct {
	features    []*geojson.Feature
	columns     []*writer.Column
	columnMap   map[string]int
	columnNames []string
	index       int
}

func (g *featureCollectionGenerator) Generate() *writer.Feature {
	if g.index >= len(g.features) {
		return nil
	}

	f := g.features[g.index]
	g.index++

	if f == nil || f.Geometry == nil {
		return g.Generate() // Skip nil features/geometries
	}

	builder := flatbuffers.NewBuilder(1024)
	fgbGeom := geometryToFGB(f.Geometry, builder)
	if fgbGeom == nil {
		return g.Generate() // Skip unsupported geometries
	}

	feature := writer.NewFeature(builder)
	feature.SetGeometry(fgbGeom)

	// Encode properties if present
	if f.Properties != nil && len(g.columns) > 0 {
		propBytes := encodeProperties(f.Properties, g.columns, g.columnMap)
		if len(propBytes) > 0 {
			feature.SetProperties(propBytes)
		}
	}

	return feature
}

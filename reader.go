package flatgeobuf

import (
	flatgeobuf "github.com/flatgeobuf/flatgeobuf/src/go"
	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// Reader provides read access to a FlatGeobuf file.
type Reader struct {
	fgb *flatgeobuf.FlatGeoBuf
}

// NewReader creates a reader from a file path.
// The file is memory-mapped for efficient access.
func NewReader(path string) (*Reader, error) {
	fgb, err := flatgeobuf.New(path)
	if err != nil {
		return nil, err
	}

	return &Reader{fgb: fgb}, nil
}

// NewReaderFromData creates a reader from byte data.
func NewReaderFromData(data []byte) (*Reader, error) {
	fgb, err := flatgeobuf.NewWithData(data)
	if err != nil {
		return nil, err
	}

	return &Reader{fgb: fgb}, nil
}

// Header returns metadata about the FlatGeobuf file.
func (r *Reader) Header() *Header {
	h := r.fgb.Header()
	if h == nil {
		return nil
	}

	header := &Header{
		Name:          string(h.Name()),
		Description:   string(h.Description()),
		FeaturesCount: h.FeaturesCount(),
		HasIndex:      h.IndexNodeSize() > 0,
	}

	// Geometry type
	header.GeometryType = flattypes.EnumNamesGeometryType[h.GeometryType()]

	// Envelope
	envLen := h.EnvelopeLength()
	if envLen >= 4 {
		header.Envelope = [4]float64{
			h.Envelope(0),
			h.Envelope(1),
			h.Envelope(2),
			h.Envelope(3),
		}
	}

	// CRS
	var crs flattypes.Crs
	if h.Crs(&crs) != nil {
		header.CRS = &CRS{
			Code:        int(crs.Code()),
			Name:        string(crs.Name()),
			Description: string(crs.Description()),
		}
	}

	// Columns
	colLen := h.ColumnsLength()
	if colLen > 0 {
		header.Columns = make([]ColumnInfo, 0, colLen)
		for i := 0; i < colLen; i++ {
			var col flattypes.Column
			if h.Columns(&col, i) {
				header.Columns = append(header.Columns, ColumnInfo{
					Name:        string(col.Name()),
					Type:        flattypes.EnumNamesColumnType[col.Type()],
					Title:       string(col.Title()),
					Description: string(col.Description()),
					Nullable:    col.Nullable(),
				})
			}
		}
	}

	return header
}

// ReadAll reads all features as a FeatureCollection.
func (r *Reader) ReadAll() (*geojson.FeatureCollection, error) {
	fc := geojson.NewFeatureCollection()
	h := r.fgb.Header()

	featuresCount := h.FeaturesCount()
	if featuresCount == 0 {
		return fc, nil
	}

	// If we have an index, we can use Search with full bounds
	if h.IndexNodeSize() > 0 {
		// Get the envelope from header
		envLen := h.EnvelopeLength()
		if envLen >= 4 {
			minX := h.Envelope(0)
			minY := h.Envelope(1)
			maxX := h.Envelope(2)
			maxY := h.Envelope(3)

			features, err := r.fgb.Search(minX, minY, maxX, maxY)
			if err != nil {
				return nil, err
			}

			for _, fgbFeature := range features {
				feature := convertFeature(fgbFeature, h)
				if feature != nil {
					fc.Append(feature)
				}
			}

			return fc, nil
		}
	}

	// Without index or envelope, we can't iterate features
	// This is a limitation of the official Go implementation
	return fc, nil
}

// ReadGeometries reads all geometries without properties.
func (r *Reader) ReadGeometries() ([]orb.Geometry, error) {
	fc, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	geometries := make([]orb.Geometry, 0, len(fc.Features))
	for _, f := range fc.Features {
		if f.Geometry != nil {
			geometries = append(geometries, f.Geometry)
		}
	}

	return geometries, nil
}

// Search performs a spatial query using the built-in index.
// Returns features whose bounding boxes intersect the query bounds.
func (r *Reader) Search(bounds orb.Bound) (*geojson.FeatureCollection, error) {
	h := r.fgb.Header()

	if h.IndexNodeSize() == 0 {
		return nil, ErrNoIndex
	}

	features, err := r.fgb.Search(bounds.Min[0], bounds.Min[1], bounds.Max[0], bounds.Max[1])
	if err != nil {
		return nil, err
	}

	fc := geojson.NewFeatureCollection()
	for _, fgbFeature := range features {
		feature := convertFeature(fgbFeature, h)
		if feature != nil {
			fc.Append(feature)
		}
	}

	return fc, nil
}

// SearchGeometries performs a spatial query returning only geometries.
func (r *Reader) SearchGeometries(bounds orb.Bound) ([]orb.Geometry, error) {
	fc, err := r.Search(bounds)
	if err != nil {
		return nil, err
	}

	geometries := make([]orb.Geometry, 0, len(fc.Features))
	for _, f := range fc.Features {
		if f.Geometry != nil {
			geometries = append(geometries, f.Geometry)
		}
	}

	return geometries, nil
}

// Close releases resources associated with the reader.
// This is important for memory-mapped files.
func (r *Reader) Close() error {
	// The FlatGeoBuf type doesn't expose a public Close method,
	// but the finalizer will clean up when garbage collected.
	// Setting to nil allows GC to collect it.
	r.fgb = nil
	return nil
}

// convertFeature converts a FlatGeobuf feature to a geojson.Feature.
func convertFeature(fgbFeature *flattypes.Feature, header *flattypes.Header) *geojson.Feature {
	if fgbFeature == nil {
		return nil
	}

	// Convert geometry
	var geomObj flattypes.Geometry
	geom := fgbFeature.Geometry(&geomObj)
	if geom == nil {
		return nil
	}

	orbGeom := geometryFromFGB(geom)
	if orbGeom == nil {
		return nil
	}

	feature := geojson.NewFeature(orbGeom)

	// Convert properties
	propsLen := fgbFeature.PropertiesLength()
	if propsLen > 0 && header.ColumnsLength() > 0 {
		propsBytes := make([]byte, propsLen)
		for i := 0; i < propsLen; i++ {
			propsBytes[i] = byte(fgbFeature.Properties(i))
		}
		feature.Properties = decodeProperties(propsBytes, header)
	}

	return feature
}

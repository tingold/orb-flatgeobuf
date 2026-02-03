package flatgeobuf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"

	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	"github.com/flatgeobuf/flatgeobuf/src/go/writer"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/paulmach/orb/geojson"
)

// inferColumns analyzes features and infers the column schema.
// It examines all properties across all features to determine
// the appropriate column types.
func inferColumns(features []*geojson.Feature, builder *flatbuffers.Builder) []*writer.Column {
	if len(features) == 0 {
		return nil
	}

	// Collect all unique property names and their types
	columnTypes := make(map[string]flattypes.ColumnType)
	columnOrder := make([]string, 0)

	for _, f := range features {
		if f.Properties == nil {
			continue
		}
		for name, value := range f.Properties {
			// Track column order (first occurrence)
			if _, exists := columnTypes[name]; !exists {
				columnOrder = append(columnOrder, name)
			}

			// Infer type from value
			inferredType := inferColumnType(value)

			// If we already have a type, prefer the more general one
			if existingType, exists := columnTypes[name]; exists {
				columnTypes[name] = promoteColumnType(existingType, inferredType)
			} else {
				columnTypes[name] = inferredType
			}
		}
	}

	// Create columns in order
	columns := make([]*writer.Column, 0, len(columnOrder))
	for _, name := range columnOrder {
		col := writer.NewColumn(builder)
		col.SetName(name)
		col.SetTitle(name) // Set title to match name for JS library compatibility
		col.SetType(columnTypes[name])
		col.SetNullable(true) // Allow null values
		columns = append(columns, col)
	}

	return columns
}

// inferColumnType determines the FlatGeobuf column type for a Go value.
func inferColumnType(value interface{}) flattypes.ColumnType {
	if value == nil {
		return flattypes.ColumnTypeString // Default to string for nil
	}

	switch v := value.(type) {
	case bool:
		return flattypes.ColumnTypeBool
	case int:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return flattypes.ColumnTypeInt
		}
		return flattypes.ColumnTypeLong
	case int8, int16, int32:
		return flattypes.ColumnTypeInt
	case int64:
		return flattypes.ColumnTypeLong
	case uint, uint8, uint16, uint32:
		return flattypes.ColumnTypeUInt
	case uint64:
		return flattypes.ColumnTypeULong
	case float32:
		return flattypes.ColumnTypeFloat
	case float64:
		return flattypes.ColumnTypeDouble
	case string:
		return flattypes.ColumnTypeString
	case json.Number:
		// Try to parse as int first, then float
		if _, err := v.Int64(); err == nil {
			return flattypes.ColumnTypeLong
		}
		return flattypes.ColumnTypeDouble
	case map[string]interface{}, []interface{}:
		return flattypes.ColumnTypeJson
	default:
		return flattypes.ColumnTypeJson
	}
}

// promoteColumnType returns the more general type when there's a conflict.
func promoteColumnType(a, b flattypes.ColumnType) flattypes.ColumnType {
	if a == b {
		return a
	}

	// If either is JSON, use JSON
	if a == flattypes.ColumnTypeJson || b == flattypes.ColumnTypeJson {
		return flattypes.ColumnTypeJson
	}

	// If either is String, use String
	if a == flattypes.ColumnTypeString || b == flattypes.ColumnTypeString {
		return flattypes.ColumnTypeString
	}

	// Numeric promotions
	numericTypes := map[flattypes.ColumnType]int{
		flattypes.ColumnTypeBool:   0,
		flattypes.ColumnTypeByte:   1,
		flattypes.ColumnTypeUByte:  2,
		flattypes.ColumnTypeShort:  3,
		flattypes.ColumnTypeUShort: 4,
		flattypes.ColumnTypeInt:    5,
		flattypes.ColumnTypeUInt:   6,
		flattypes.ColumnTypeLong:   7,
		flattypes.ColumnTypeULong:  8,
		flattypes.ColumnTypeFloat:  9,
		flattypes.ColumnTypeDouble: 10,
	}

	rankA, okA := numericTypes[a]
	rankB, okB := numericTypes[b]

	if okA && okB {
		if rankA > rankB {
			return a
		}
		return b
	}

	// Default to JSON for unknown combinations
	return flattypes.ColumnTypeJson
}

// encodeProperties encodes geojson.Properties to FlatGeobuf binary format.
// The format is: [2-byte column index][value bytes]... repeated for each property.
func encodeProperties(props geojson.Properties, columns []*writer.Column, columnMap map[string]int) []byte {
	if props == nil || len(columns) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for name, value := range props {
		if value == nil {
			continue // Skip null values
		}

		colIndex, ok := columnMap[name]
		if !ok {
			continue // Column not in schema
		}

		// Write column index (uint16, little-endian)
		indexBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(indexBytes, uint16(colIndex))
		buf.Write(indexBytes)

		// Write value based on column type
		col := columns[colIndex]
		writePropertyValue(&buf, value, col)
	}

	return buf.Bytes()
}

// writePropertyValue writes a single property value to the buffer.
func writePropertyValue(buf *bytes.Buffer, value interface{}, col *writer.Column) {
	// Get column type - we need to access it via reflection since Column
	// doesn't expose Type() directly. For now, infer from value.
	colType := inferColumnType(value)

	switch colType {
	case flattypes.ColumnTypeBool:
		if v, ok := value.(bool); ok {
			if v {
				buf.WriteByte(1)
			} else {
				buf.WriteByte(0)
			}
		}

	case flattypes.ColumnTypeByte:
		if v, ok := toInt64(value); ok {
			buf.WriteByte(byte(v))
		}

	case flattypes.ColumnTypeUByte:
		if v, ok := toInt64(value); ok {
			buf.WriteByte(byte(v))
		}

	case flattypes.ColumnTypeShort:
		if v, ok := toInt64(value); ok {
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(int16(v)))
			buf.Write(b)
		}

	case flattypes.ColumnTypeUShort:
		if v, ok := toInt64(value); ok {
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(v))
			buf.Write(b)
		}

	case flattypes.ColumnTypeInt:
		if v, ok := toInt64(value); ok {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(int32(v)))
			buf.Write(b)
		}

	case flattypes.ColumnTypeUInt:
		if v, ok := toInt64(value); ok {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(v))
			buf.Write(b)
		}

	case flattypes.ColumnTypeLong:
		if v, ok := toInt64(value); ok {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(v))
			buf.Write(b)
		}

	case flattypes.ColumnTypeULong:
		if v, ok := toUint64(value); ok {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, v)
			buf.Write(b)
		}

	case flattypes.ColumnTypeFloat:
		if v, ok := toFloat64(value); ok {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v)))
			buf.Write(b)
		}

	case flattypes.ColumnTypeDouble:
		if v, ok := toFloat64(value); ok {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, math.Float64bits(v))
			buf.Write(b)
		}

	case flattypes.ColumnTypeString:
		s := toString(value)
		buf.WriteString(s)
		buf.WriteByte(0) // Null terminator

	case flattypes.ColumnTypeJson:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			jsonBytes = []byte("{}")
		}
		buf.Write(jsonBytes)
		buf.WriteByte(0) // Null terminator

	case flattypes.ColumnTypeDateTime:
		s := toString(value)
		buf.WriteString(s)
		buf.WriteByte(0) // Null terminator

	case flattypes.ColumnTypeBinary:
		if b, ok := value.([]byte); ok {
			// Write length followed by bytes
			lenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(lenBytes, uint32(len(b)))
			buf.Write(lenBytes)
			buf.Write(b)
		}
	}
}

// decodeProperties decodes FlatGeobuf binary properties to geojson.Properties.
func decodeProperties(data []byte, header *flattypes.Header) geojson.Properties {
	if len(data) == 0 || header == nil {
		return nil
	}

	props := make(geojson.Properties)
	offset := 0

	for offset < len(data) {
		// Need at least 2 bytes for column index
		if offset+2 > len(data) {
			break
		}

		// Read column index
		colIndex := binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2

		// Validate column index
		if int(colIndex) >= header.ColumnsLength() {
			break
		}

		// Get column info
		var col flattypes.Column
		if !header.Columns(&col, int(colIndex)) {
			break
		}

		colName := string(col.Name())
		colType := col.Type()

		// Read value based on type
		value, bytesRead := readPropertyValue(data[offset:], colType)
		if bytesRead == 0 && colType != flattypes.ColumnTypeBool {
			break
		}
		offset += bytesRead

		props[colName] = value
	}

	return props
}

// readPropertyValue reads a property value from the buffer.
// Returns the value and number of bytes read.
func readPropertyValue(data []byte, colType flattypes.ColumnType) (interface{}, int) {
	switch colType {
	case flattypes.ColumnTypeBool:
		if len(data) < 1 {
			return nil, 0
		}
		return data[0] != 0, 1

	case flattypes.ColumnTypeByte:
		if len(data) < 1 {
			return nil, 0
		}
		return int8(data[0]), 1

	case flattypes.ColumnTypeUByte:
		if len(data) < 1 {
			return nil, 0
		}
		return data[0], 1

	case flattypes.ColumnTypeShort:
		if len(data) < 2 {
			return nil, 0
		}
		return int16(binary.LittleEndian.Uint16(data[:2])), 2

	case flattypes.ColumnTypeUShort:
		if len(data) < 2 {
			return nil, 0
		}
		return binary.LittleEndian.Uint16(data[:2]), 2

	case flattypes.ColumnTypeInt:
		if len(data) < 4 {
			return nil, 0
		}
		return int32(binary.LittleEndian.Uint32(data[:4])), 4

	case flattypes.ColumnTypeUInt:
		if len(data) < 4 {
			return nil, 0
		}
		return binary.LittleEndian.Uint32(data[:4]), 4

	case flattypes.ColumnTypeLong:
		if len(data) < 8 {
			return nil, 0
		}
		return int64(binary.LittleEndian.Uint64(data[:8])), 8

	case flattypes.ColumnTypeULong:
		if len(data) < 8 {
			return nil, 0
		}
		return binary.LittleEndian.Uint64(data[:8]), 8

	case flattypes.ColumnTypeFloat:
		if len(data) < 4 {
			return nil, 0
		}
		bits := binary.LittleEndian.Uint32(data[:4])
		return math.Float32frombits(bits), 4

	case flattypes.ColumnTypeDouble:
		if len(data) < 8 {
			return nil, 0
		}
		bits := binary.LittleEndian.Uint64(data[:8])
		return math.Float64frombits(bits), 8

	case flattypes.ColumnTypeString, flattypes.ColumnTypeDateTime:
		// Find null terminator
		nullIdx := bytes.IndexByte(data, 0)
		if nullIdx == -1 {
			// No null terminator, read to end
			return string(data), len(data)
		}
		return string(data[:nullIdx]), nullIdx + 1

	case flattypes.ColumnTypeJson:
		// Find null terminator
		nullIdx := bytes.IndexByte(data, 0)
		if nullIdx == -1 {
			nullIdx = len(data)
		}
		var jsonValue interface{}
		if err := json.Unmarshal(data[:nullIdx], &jsonValue); err != nil {
			return string(data[:nullIdx]), nullIdx + 1
		}
		if nullIdx < len(data) {
			return jsonValue, nullIdx + 1
		}
		return jsonValue, nullIdx

	case flattypes.ColumnTypeBinary:
		if len(data) < 4 {
			return nil, 0
		}
		length := binary.LittleEndian.Uint32(data[:4])
		if len(data) < int(4+length) {
			return nil, 0
		}
		return data[4 : 4+length], int(4 + length)

	default:
		return nil, 0
	}
}

// Type conversion helpers

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int8:
		return int64(val), true
	case int16:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case uint:
		return int64(val), true
	case uint8:
		return int64(val), true
	case uint16:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	case float32:
		return int64(val), true
	case float64:
		return int64(val), true
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i, true
		}
		if f, err := val.Float64(); err == nil {
			return int64(f), true
		}
	}
	return 0, false
}

func toUint64(v interface{}) (uint64, bool) {
	switch val := v.(type) {
	case uint:
		return uint64(val), true
	case uint8:
		return uint64(val), true
	case uint16:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case uint64:
		return val, true
	case int:
		if val >= 0 {
			return uint64(val), true
		}
	case int64:
		if val >= 0 {
			return uint64(val), true
		}
	case float64:
		if val >= 0 {
			return uint64(val), true
		}
	case json.Number:
		if i, err := val.Int64(); err == nil && i >= 0 {
			return uint64(i), true
		}
	}
	return 0, false
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		// For other types, use JSON encoding
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// buildColumnMap creates a map from column name to index.
func buildColumnMap(columns []*writer.Column, columnNames []string) map[string]int {
	m := make(map[string]int, len(columnNames))
	for i, name := range columnNames {
		m[name] = i
	}
	return m
}

// getColumnNames extracts names from the columns slice.
func getColumnNames(features []*geojson.Feature) []string {
	if len(features) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	names := make([]string, 0)

	for _, f := range features {
		if f.Properties == nil {
			continue
		}
		for name := range f.Properties {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	return names
}

package flatgeobuf

import (
	"encoding/json"
	"testing"

	"github.com/flatgeobuf/flatgeobuf/src/go/flattypes"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func TestInferColumnType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected flattypes.ColumnType
	}{
		{"nil", nil, flattypes.ColumnTypeString},
		{"bool true", true, flattypes.ColumnTypeBool},
		{"bool false", false, flattypes.ColumnTypeBool},
		{"int", 42, flattypes.ColumnTypeInt},
		{"int64", int64(9999999999), flattypes.ColumnTypeLong},
		{"float32", float32(3.14), flattypes.ColumnTypeFloat},
		{"float64", 3.14159, flattypes.ColumnTypeDouble},
		{"string", "hello", flattypes.ColumnTypeString},
		{"map", map[string]interface{}{"key": "value"}, flattypes.ColumnTypeJson},
		{"slice", []interface{}{1, 2, 3}, flattypes.ColumnTypeJson},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferColumnType(tt.value)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestInferColumnType_JsonNumber(t *testing.T) {
	intNum := json.Number("42")
	result := inferColumnType(intNum)
	if result != flattypes.ColumnTypeLong {
		t.Errorf("expected Long for integer json.Number, got %v", result)
	}

	floatNum := json.Number("3.14")
	result = inferColumnType(floatNum)
	if result != flattypes.ColumnTypeDouble {
		t.Errorf("expected Double for float json.Number, got %v", result)
	}
}

func TestPromoteColumnType(t *testing.T) {
	tests := []struct {
		name     string
		a, b     flattypes.ColumnType
		expected flattypes.ColumnType
	}{
		{"same type", flattypes.ColumnTypeInt, flattypes.ColumnTypeInt, flattypes.ColumnTypeInt},
		{"int to long", flattypes.ColumnTypeInt, flattypes.ColumnTypeLong, flattypes.ColumnTypeLong},
		{"int to double", flattypes.ColumnTypeInt, flattypes.ColumnTypeDouble, flattypes.ColumnTypeDouble},
		{"any to json", flattypes.ColumnTypeInt, flattypes.ColumnTypeJson, flattypes.ColumnTypeJson},
		{"any to string", flattypes.ColumnTypeInt, flattypes.ColumnTypeString, flattypes.ColumnTypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := promoteColumnType(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestInferColumns(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	features := []*geojson.Feature{
		{
			Geometry: orb.Point{1, 2},
			Properties: geojson.Properties{
				"name":   "test",
				"value":  42,
				"active": true,
			},
		},
		{
			Geometry: orb.Point{3, 4},
			Properties: geojson.Properties{
				"name":  "test2",
				"value": 100,
				"score": 3.14,
			},
		},
	}

	columns := inferColumns(features, builder)

	if len(columns) != 4 { // name, value, active, score
		t.Errorf("expected 4 columns, got %d", len(columns))
	}
}

func TestInferColumns_EmptyFeatures(t *testing.T) {
	builder := flatbuffers.NewBuilder(256)
	columns := inferColumns([]*geojson.Feature{}, builder)

	if columns != nil {
		t.Error("expected nil columns for empty features")
	}
}

func TestGetColumnNames(t *testing.T) {
	features := []*geojson.Feature{
		{
			Geometry:   orb.Point{1, 2},
			Properties: geojson.Properties{"a": 1, "b": 2},
		},
		{
			Geometry:   orb.Point{3, 4},
			Properties: geojson.Properties{"b": 3, "c": 4},
		},
	}

	names := getColumnNames(features)

	// Should have a, b, c (order may vary)
	if len(names) != 3 {
		t.Errorf("expected 3 column names, got %d", len(names))
	}

	// Check all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, expected := range []string{"a", "b", "c"} {
		if !nameSet[expected] {
			t.Errorf("missing column name: %s", expected)
		}
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected int64
		ok       bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(100), 100, true},
		{"float64", 3.9, 3, true},
		{"string", "hello", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toInt64(tt.value)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got ok=%v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected float64
		ok       bool
	}{
		{"float64", 3.14, 3.14, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", 42, 42.0, true},
		{"string", "hello", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.value)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got ok=%v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"bytes", []byte("world"), "world"},
		{"int", 42, "42"},
		{"bool", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

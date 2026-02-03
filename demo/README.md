# FlatGeobuf Demo

This demo shows how to serve FlatGeobuf data from a Go server and render it on a MapLibre GL map with OpenStreetMap tiles using the `mapbox-gl-flatgeobuf` extension.

## Prerequisites

- Go 1.21 or later

## Running the Demo

1. Navigate to the server directory:

```bash
cd demo/server
```

2. Run the server:

```bash
go run main.go
```

3. Open your browser to [http://localhost:8080](http://localhost:8080)

That's it! No API keys or tokens required.

## What You'll See

The demo displays a world map with major cities rendered as circles:

- **Blue circles**: Regular cities
- **Red circles**: Capital cities
- **Circle size**: Proportional to population

Click on any city to see a popup with details including name, country, population, and capital status.

## How It Works

### Server (Go)

The server (`server/main.go`):

1. Creates a GeoJSON FeatureCollection with city data
2. Uses the `orb-flatgeobuf` library to serialize it to FlatGeobuf format
3. Serves the FlatGeobuf data at `/data.fgb`
4. Serves the static client files at `/`

### Client (JavaScript)

The client (`client/index.html`):

1. Initializes a MapLibre GL map with OpenStreetMap raster tiles
2. Uses the `mapbox-gl-flatgeobuf` extension to add `/data.fgb` as a tiled GeoJSON source
3. Renders cities as circles with size based on population
4. Shows popups on click with city details

## Sample Data

The demo includes 15 major world cities with the following properties:

| Property   | Type    | Description              |
|------------|---------|--------------------------|
| name       | string  | City name                |
| country    | string  | Country name             |
| population | integer | City population          |
| capital    | boolean | Whether it's a capital   |

## Customization

### Adding More Cities

Edit the `cities` slice in `server/main.go`:

```go
var cities = []City{
    {"City Name", "Country", longitude, latitude, population, isCapital},
    // ...
}
```

### Changing the Map Style

The demo uses OpenStreetMap raster tiles by default. You can modify the `style` object in `client/index.html` to use different tile providers. Some free alternatives:

- **Stadia Maps**: https://stadiamaps.com/
- **Carto**: https://carto.com/basemaps/

Or use a pre-built MapLibre style from https://github.com/maplibre/demotiles

## Troubleshooting

### Server won't start

Make sure you're in the `demo/server` directory when running `go run main.go`, as the server looks for the client files at `../client`.

### No data appears on the map

Check the browser console for errors. Common issues:
- CORS errors: Make sure you're accessing via `http://localhost:8080`, not by opening the HTML file directly
- Network errors: Verify the server is running and `/data.fgb` returns data

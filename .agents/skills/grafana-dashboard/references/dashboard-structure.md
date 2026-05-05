# Dashboard Structure Reference

## Complete Dashboard Template

```json
{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": { "type": "grafana", "uid": "-- Grafana --" },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 1,
  "id": null,
  "links": [],
  "panels": [],
  "refresh": "1m",
  "schemaVersion": 39,
  "tags": ["generated", "prometheus"],
  "templating": { "list": [] },
  "time": { "from": "now-1h", "to": "now" },
  "timepicker": {},
  "timezone": "browser",
  "title": "Dashboard Title",
  "uid": "",
  "version": 0,
  "weekStart": ""
}
```

## Templating Variables

### Datasource Variable

**Before writing this variable:** call `grafana:list_datasources` (or `mcp__grafana__list_datasources`) to discover the actual prometheus-compatible datasource name in the target Grafana instance (e.g. `promxy`, `Prometheus`, `thanos`). Set `current.text` and `current.value` to that name. Never leave the default as `"default"` — it silently loads the wrong datasource on import and forces users to manually change the dropdown every time.

```json
{
  "current": { "selected": true, "text": "<discovered-datasource-name>", "value": "<discovered-datasource-name>" },
  "hide": 0,
  "includeAll": false,
  "multi": false,
  "name": "datasource",
  "options": [],
  "query": "prometheus",
  "queryValue": "",
  "refresh": 1,
  "regex": "",
  "skipUrlSync": false,
  "type": "datasource"
}
```

### Label Filter Variable

```json
{
  "allValue": ".*",
  "current": { "selected": true, "text": "All", "value": "$__all" },
  "datasource": { "type": "prometheus", "uid": "${datasource}" },
  "definition": "label_values(up, job)",
  "hide": 0,
  "includeAll": true,
  "multi": true,
  "name": "job",
  "options": [],
  "query": { "qryType": 1, "query": "label_values(up, job)", "refId": "PrometheusVariableQueryEditor-VariableQuery" },
  "refresh": 2,
  "regex": "",
  "skipUrlSync": false,
  "sort": 1,
  "type": "query"
}
```

## Panel Templates

### Time Series Panel

```json
{
  "datasource": { "type": "prometheus", "uid": "${datasource}" },
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "palette-classic" },
      "custom": {
        "axisBorderShow": false,
        "axisCenteredZero": false,
        "axisColorMode": "text",
        "axisLabel": "",
        "axisPlacement": "auto",
        "barAlignment": 0,
        "barWidthFactor": 0.6,
        "drawStyle": "line",
        "fillOpacity": 10,
        "gradientMode": "none",
        "hideFrom": { "legend": false, "tooltip": false, "viz": false },
        "insertNulls": false,
        "lineInterpolation": "linear",
        "lineWidth": 1,
        "pointSize": 5,
        "scaleDistribution": { "type": "linear" },
        "showPoints": "auto",
        "spanNulls": false,
        "stacking": { "group": "A", "mode": "none" },
        "thresholdsStyle": { "mode": "off" }
      },
      "mappings": [],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          { "color": "green", "value": null },
          { "color": "red", "value": 80 }
        ]
      },
      "unit": "short"
    },
    "overrides": []
  },
  "gridPos": { "h": 5, "w": 8, "x": 0, "y": 0 },
  "id": 1,
  "options": {
    "legend": {
      "calcs": ["max", "mean", "last"],
      "displayMode": "table",
      "placement": "bottom",
      "showLegend": true
    },
    "tooltip": { "hideZeroes": false, "mode": "multi", "sort": "desc" }
  },
  "targets": [
    {
      "datasource": { "type": "prometheus", "uid": "${datasource}" },
      "editorMode": "code",
      "expr": "rate(http_requests_total{job=~\"$job\"}[$__rate_interval])",
      "legendFormat": "{{method}} {{status}}",
      "range": true,
      "refId": "A"
    }
  ],
  "title": "Panel Title",
  "type": "timeseries"
}
```

### Heatmap Panel (for histograms)

```json
{
  "datasource": { "type": "prometheus", "uid": "${datasource}" },
  "fieldConfig": {
    "defaults": {
      "custom": {
        "hideFrom": { "legend": false, "tooltip": false, "viz": false },
        "scaleDistribution": { "type": "linear" }
      }
    },
    "overrides": []
  },
  "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
  "id": 2,
  "options": {
    "calculate": false,
    "cellGap": 1,
    "color": {
      "exponent": 0.5,
      "fill": "dark-orange",
      "mode": "scheme",
      "reverse": false,
      "scale": "exponential",
      "scheme": "Spectral",
      "steps": 64
    },
    "exemplars": { "color": "rgba(255,0,255,0.7)" },
    "filterValues": { "le": 1e-9 },
    "legend": { "show": true },
    "rowsFrame": { "layout": "auto" },
    "tooltip": { "show": true, "yHistogram": true },
    "yAxis": { "axisPlacement": "left", "reverse": false, "unit": "s" }
  },
  "pluginVersion": "11.4.0",
  "targets": [
    {
      "datasource": { "type": "prometheus", "uid": "${datasource}" },
      "editorMode": "code",
      "expr": "sum(rate(http_request_duration_seconds_bucket{job=~\"$job\"}[$__rate_interval])) by (le)",
      "format": "heatmap",
      "legendFormat": "{{le}}",
      "range": true,
      "refId": "A"
    }
  ],
  "title": "Request Duration Heatmap",
  "type": "heatmap"
}
```

### Table Panel (for info metrics)

```json
{
  "datasource": { "type": "prometheus", "uid": "${datasource}" },
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "thresholds" },
      "custom": {
        "align": "auto",
        "cellOptions": { "type": "auto" },
        "inspect": false
      },
      "mappings": [],
      "thresholds": {
        "mode": "absolute",
        "steps": [{ "color": "green", "value": null }]
      }
    },
    "overrides": []
  },
  "gridPos": { "h": 6, "w": 12, "x": 0, "y": 0 },
  "id": 3,
  "options": {
    "cellHeight": "sm",
    "footer": { "countRows": false, "fields": "", "reducer": ["sum"], "show": false },
    "showHeader": true,
    "sortBy": []
  },
  "pluginVersion": "11.4.0",
  "targets": [
    {
      "datasource": { "type": "prometheus", "uid": "${datasource}" },
      "editorMode": "code",
      "exemplar": false,
      "expr": "build_info{job=~\"$job\"}",
      "format": "table",
      "instant": true,
      "legendFormat": "__auto",
      "range": false,
      "refId": "A"
    }
  ],
  "title": "Build Info",
  "transformations": [
    { "id": "labelsToFields", "options": {} }
  ],
  "type": "table"
}
```

### Stat Panel

```json
{
  "datasource": { "type": "prometheus", "uid": "${datasource}" },
  "fieldConfig": {
    "defaults": {
      "color": { "mode": "thresholds" },
      "mappings": [],
      "thresholds": {
        "mode": "absolute",
        "steps": [
          { "color": "green", "value": null },
          { "color": "yellow", "value": 70 },
          { "color": "red", "value": 90 }
        ]
      },
      "unit": "short"
    },
    "overrides": []
  },
  "gridPos": { "h": 4, "w": 4, "x": 0, "y": 0 },
  "id": 4,
  "options": {
    "colorMode": "value",
    "graphMode": "area",
    "justifyMode": "auto",
    "orientation": "auto",
    "reduceOptions": { "calcs": ["lastNotNull"], "fields": "", "values": false },
    "showPercentChange": false,
    "textMode": "auto",
    "wideLayout": true
  },
  "pluginVersion": "11.4.0",
  "targets": [
    {
      "datasource": { "type": "prometheus", "uid": "${datasource}" },
      "editorMode": "code",
      "expr": "sum(up{job=~\"$job\"})",
      "legendFormat": "__auto",
      "range": true,
      "refId": "A"
    }
  ],
  "title": "Instances Up",
  "type": "stat"
}
```

### Row Panel (Collapsible)

```json
{
  "collapsed": false,
  "gridPos": { "h": 1, "w": 24, "x": 0, "y": 0 },
  "id": 100,
  "panels": [],
  "title": "Row Title",
  "type": "row"
}
```

## Grid Positioning

Panels use a 24-column grid system:
- `x`: Column position (0-23)
- `y`: Row position (0+)
- `w`: Width in columns (1-24)
- `h`: Height in grid units (typically 4-10)

Common layouts:
- 3 panels per row: `w: 8` each
- 2 panels per row: `w: 12` each
- Full width: `w: 24`

## Legend Placement Options

```json
"options": {
  "legend": {
    "calcs": ["max", "mean", "last"],
    "displayMode": "table",
    "placement": "bottom",
    "showLegend": true
  }
}
```

Placement values: `"bottom"`, `"right"`

## Unit Mappings

Common Grafana unit codes:

| Unit | Code |
|------|------|
| Seconds | `s` |
| Milliseconds | `ms` |
| Bytes | `decbytes` |
| Bits | `bits` |
| Bytes/sec | `Bps` |
| Requests/sec | `reqps` |
| Percent (0-100) | `percent` |
| Percent (0-1) | `percentunit` |
| Short | `short` |
| None | `none` |

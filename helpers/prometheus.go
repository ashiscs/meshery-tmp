package helpers

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/grafana-tools/sdk"
	"github.com/layer5io/meshery/models"
	"github.com/pkg/errors"
	promAPI "github.com/prometheus/client_golang/api"
	promQAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	promModel "github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// PrometheusClient represents a prometheus client in Meshery
type PrometheusClient struct {
	grafanaClient *GrafanaClient
	promURL       string
}

// NewPrometheusClient returns a PrometheusClient
func NewPrometheusClient(ctx context.Context, promURL string, validate bool) (*PrometheusClient, error) {
	// client, err := promAPI.NewClient(promAPI.Config{Address: promURL})
	// if err != nil {
	// 	msg := errors.New("unable to connect to prometheus")
	// 	logrus.Error(errors.Wrap(err, msg.Error()))
	// 	return nil, msg
	// }
	// queryAPI := promQAPI.NewAPI(client)
	// return &PrometheusClient{
	// 	client:      client,
	// 	queryClient: queryAPI,
	// }, nil
	p := &PrometheusClient{
		grafanaClient: NewGrafanaClientForPrometheus(promURL),
		promURL:       promURL,
	}
	if validate {
		_, err := p.grafanaClient.makeRequest(ctx, promURL+"/api/v1/status/config")
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// ImportGrafanaBoard takes raw Grafana board json and returns GrafanaBoard pointer for use in Meshery
func (p *PrometheusClient) ImportGrafanaBoard(ctx context.Context, boardData []byte) (*models.GrafanaBoard, error) {
	board := &sdk.Board{}
	if err := json.Unmarshal(boardData, board); err != nil {
		msg := errors.New("unable to parse grafana board data")
		logrus.Error(errors.Wrap(err, msg.Error()))
		return nil, msg
	}
	return p.grafanaClient.ProcessBoard(board, &sdk.FoundBoard{
		Title: board.Title,
		URI:   board.Slug,
	})
}

// Query queries prometheus using the GrafanaClient
func (p *PrometheusClient) Query(ctx context.Context, queryData *url.Values) ([]byte, error) {
	return p.grafanaClient.GrafanaQuery(ctx, queryData)
}

// QueryRange queries prometheus using the GrafanaClient
func (p *PrometheusClient) QueryRange(ctx context.Context, queryData *url.Values) ([]byte, error) {
	return p.grafanaClient.GrafanaQueryRange(ctx, queryData)
}

// GetStaticBoard retrieves the static board config
func (p *PrometheusClient) GetStaticBoard(ctx context.Context) (*models.GrafanaBoard, error) {
	return p.ImportGrafanaBoard(ctx, []byte(staticBoard))
}

// QueryRangeUsingClient performs a range query within a window
func (p *PrometheusClient) QueryRangeUsingClient(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration) (promModel.Value, error) {
	c, _ := promAPI.NewClient(promAPI.Config{
		Address: p.promURL,
	})
	qc := promQAPI.NewAPI(c)
	result, _, err := qc.QueryRange(ctx, query, promQAPI.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	})
	if err != nil {
		err := errors.Wrapf(err, "error fetching data for query: %s, with start: %v, end: %v, step: %v", query, startTime, endTime, step)
		logrus.Error(err)
		return nil, err
	}
	return result, nil
}

// ComputeStep computes the step size for a window
func (p *PrometheusClient) ComputeStep(ctx context.Context, start, end time.Time) time.Duration {
	step := 5 * time.Second
	diff := end.Sub(start)
	// all calc. here are approx.
	if diff <= 10*time.Minute { // 10 mins
		step = 5 * time.Second
	} else if diff <= 30*time.Minute { // 30 mins
		step = 10 * time.Second
	} else if diff > 30*time.Minute && diff <= time.Hour { // 60 mins/1hr
		step = 20 * time.Second
	} else if diff > 1*time.Hour && diff <= 3*time.Hour { // 3 time.Hour
		step = 1 * time.Minute
	} else if diff > 3*time.Hour && diff <= 6*time.Hour { // 6 time.Hour
		step = 2 * time.Minute
	} else if diff > 6*time.Hour && diff <= 1*24*time.Hour { // 24 time.Hour/1 day
		step = 8 * time.Minute
	} else if diff > 1*24*time.Hour && diff <= 2*24*time.Hour { // 2 24*time.Hour
		step = 16 * time.Minute
	} else if diff > 2*24*time.Hour && diff <= 4*24*time.Hour { // 4 24*time.Hour
		step = 32 * time.Minute
	} else if diff > 4*24*time.Hour && diff <= 7*24*time.Hour { // 7 24*time.Hour
		step = 56 * time.Minute
	} else if diff > 7*24*time.Hour && diff <= 15*24*time.Hour { // 15 24*time.Hour
		step = 2 * time.Hour
	} else if diff > 15*24*time.Hour && diff <= 1*30*24*time.Hour { // 30 24*time.Hour/1 month
		step = 4 * time.Hour
	} else if diff > 1*30*24*time.Hour && diff <= 3*30*24*time.Hour { // 3 months
		step = 12 * time.Hour
	} else if diff > 3*30*24*time.Hour && diff <= 6*30*24*time.Hour { // 6 months
		step = 1 * 24 * time.Hour
	} else if diff > 6*30*24*time.Hour && diff <= 1*12*30*24*time.Hour { // 1 year/12 months
		step = 2 * 24 * time.Hour
	} else if diff > 1*12*30*24*time.Hour && diff <= 2*12*30*24*time.Hour { // 2 years
		step = 4 * 24 * time.Hour
	} else if diff > 2*12*30*24*time.Hour && diff <= 5*12*30*24*time.Hour { // 5 years
		step = 10 * 24 * time.Hour
	} else {
		step = 30 * 24 * time.Hour
	}
	return step
}

const staticBoard = `
{
	"annotations": {
	  "list": [
		{
		  "builtIn": 1,
		  "datasource": "-- Grafana --",
		  "enable": true,
		  "hide": true,
		  "iconColor": "rgba(0, 211, 255, 1)",
		  "name": "Annotations & Alerts",
		  "type": "dashboard"
		}
	  ]
	},
	"editable": false,
	"gnetId": null,
	"graphTooltip": 0,
	"id": 7,
	"links": [],
	"panels": [
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 5,
		  "w": 24,
		  "x": 0,
		  "y": 0
		},
		"id": 46,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(istio_build{component=\"galley\"}) by (tag)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "{{ tag }}",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Galley Versions",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": false
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"collapsed": false,
		"gridPos": {
		  "h": 1,
		  "w": 24,
		  "x": 0,
		  "y": 5
		},
		"id": 40,
		"panels": [],
		"title": "Resource Usage",
		"type": "row"
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 8,
		  "w": 6,
		  "x": 0,
		  "y": 6
		},
		"id": 36,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "process_virtual_memory_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Virtual Memory",
			"refId": "A"
		  },
		  {
			"expr": "process_resident_memory_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Resident Memory",
			"refId": "B"
		  },
		  {
			"expr": "go_memstats_heap_sys_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "heap sys",
			"refId": "C"
		  },
		  {
			"expr": "go_memstats_heap_alloc_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "heap alloc",
			"refId": "D"
		  },
		  {
			"expr": "go_memstats_alloc_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Alloc",
			"refId": "F"
		  },
		  {
			"expr": "go_memstats_heap_inuse_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Heap in-use",
			"refId": "G"
		  },
		  {
			"expr": "go_memstats_stack_inuse_bytes{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Stack in-use",
			"refId": "H"
		  },
		  {
			"expr": "sum(container_memory_usage_bytes{container_name=~\"galley\", pod_name=~\"istio-galley-.*\"})",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Total (kis)",
			"refId": "E"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Memory",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": false
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 8,
		  "w": 6,
		  "x": 6,
		  "y": 6
		},
		"id": 38,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(rate(container_cpu_usage_seconds_total{container_name=~\"galley\", pod_name=~\"istio-galley-.*\"}[1m]))",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Total (k8s)",
			"refId": "A"
		  },
		  {
			"expr": "sum(rate(container_cpu_usage_seconds_total{container_name=~\"galley\", pod_name=~\"istio-galley-.*\"}[1m])) by (container_name)",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "{{ container_name }} (k8s)",
			"refId": "B"
		  },
		  {
			"expr": "irate(process_cpu_seconds_total{job=\"galley\"}[1m])",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "galley (self-reported)",
			"refId": "C"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "CPU",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 8,
		  "w": 6,
		  "x": 12,
		  "y": 6
		},
		"id": 42,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "process_open_fds{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "Open FDs (galley)",
			"refId": "A"
		  },
		  {
			"expr": "container_fs_usage_bytes{container_name=~\"galley\", pod_name=~\"istio-galley-.*\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "{{ container_name }} ",
			"refId": "B"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Disk",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": false
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 8,
		  "w": 6,
		  "x": 18,
		  "y": 6
		},
		"id": 44,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "go_goroutines{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 2,
			"legendFormat": "goroutines_total",
			"refId": "A"
		  },
		  {
			"expr": "galley_mcp_source_clients_total",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "clients_total",
			"refId": "B"
		  },
		  {
			"expr": "go_goroutines{job=\"galley\"}/galley_mcp_source_clients_total",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "avg_goroutines_per_client",
			"refId": "C"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Goroutines",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"collapsed": false,
		"gridPos": {
		  "h": 1,
		  "w": 24,
		  "x": 0,
		  "y": 14
		},
		"id": 10,
		"panels": [],
		"title": "Runtime",
		"type": "row"
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 0,
		  "y": 15
		},
		"id": 2,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(rate(galley_runtime_strategy_on_change_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Strategy Change Events",
			"refId": "A"
		  },
		  {
			"expr": "sum(rate(galley_runtime_processor_events_processed_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Processed Events",
			"refId": "B"
		  },
		  {
			"expr": "sum(rate(galley_runtime_processor_snapshots_published_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Snapshot Published",
			"refId": "C"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Event Rates",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Events/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": "",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 8,
		  "y": 15
		},
		"id": 4,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(rate(galley_runtime_strategy_timer_max_time_reached_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Max Time Reached",
			"refId": "A"
		  },
		  {
			"expr": "sum(rate(galley_runtime_strategy_timer_quiesce_reached_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Quiesce Reached",
			"refId": "B"
		  },
		  {
			"expr": "sum(rate(galley_runtime_strategy_timer_resets_total[1m])) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Timer Resets",
			"refId": "C"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Timer Rates",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Events/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 16,
		  "y": 15
		},
		"id": 8,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 3,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": true,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "histogram_quantile(0.50, sum by (le) (galley_runtime_processor_snapshot_events_total_bucket))",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "P50",
			"refId": "A"
		  },
		  {
			"expr": "histogram_quantile(0.90, sum by (le) (galley_runtime_processor_snapshot_events_total_bucket))",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "P90",
			"refId": "B"
		  },
		  {
			"expr": "histogram_quantile(0.95, sum by (le) (galley_runtime_processor_snapshot_events_total_bucket))",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "P95",
			"refId": "C"
		  },
		  {
			"expr": "histogram_quantile(0.99, sum by (le) (galley_runtime_processor_snapshot_events_total_bucket))",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "P99",
			"refId": "D"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Events Per Snapshot",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 8,
		  "y": 21
		},
		"id": 6,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum by (typeURL) (galley_runtime_state_type_instances_total)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "{{ typeURL }}",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "State Type Instances",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Count",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"collapsed": false,
		"gridPos": {
		  "h": 1,
		  "w": 24,
		  "x": 0,
		  "y": 27
		},
		"id": 34,
		"panels": [],
		"title": "Validation",
		"type": "row"
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 0,
		  "y": 28
		},
		"id": 28,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "galley_validation_cert_key_updates{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Key Updates",
			"refId": "A"
		  },
		  {
			"expr": "galley_validation_cert_key_update_errors{job=\"galley\"}",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Key Update Errors: {{ error }}",
			"refId": "B"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Validation Webhook Certificate",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 8,
		  "y": 28
		},
		"id": 30,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(galley_validation_passed{job=\"galley\"}) by (group, version, resource)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Passed: {{ group }}/{{ version }}/{{resource}}",
			"refId": "A"
		  },
		  {
			"expr": "sum(galley_validation_failed{job=\"galley\"}) by (group, version, resource, reason)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Failed: {{ group }}/{{ version }}/{{resource}} ({{ reason}})",
			"refId": "B"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Resource Validation",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 16,
		  "y": 28
		},
		"id": 32,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(galley_validation_http_error{job=\"galley\"}) by (status)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "{{ status }}",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Validation HTTP Errors",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"collapsed": false,
		"gridPos": {
		  "h": 1,
		  "w": 24,
		  "x": 0,
		  "y": 34
		},
		"id": 12,
		"panels": [],
		"title": "Kubernetes Source",
		"type": "row"
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 0,
		  "y": 35
		},
		"id": 14,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "rate(galley_source_kube_event_success_total[1m]) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Success",
			"refId": "A"
		  },
		  {
			"expr": "rate(galley_source_kube_event_error_total[1m]) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Error",
			"refId": "B"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Source Event Rate",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Events/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 8,
		  "y": 35
		},
		"id": 16,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "rate(galley_source_kube_dynamic_converter_success_total[1m]) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "{apiVersion=\"{{apiVersion}}\",group=\"{{group}}\",kind=\"{{kind}}\"}",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Kubernetes Object Conversion Successes",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Conversions/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 16,
		  "y": 35
		},
		"id": 24,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "rate(galley_source_kube_dynamic_converter_failure_total[1m]) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Error",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Kubernetes Object Conversion Failures",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "Failures/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"collapsed": false,
		"gridPos": {
		  "h": 1,
		  "w": 24,
		  "x": 0,
		  "y": 41
		},
		"id": 18,
		"panels": [],
		"title": "Mesh Configuration Protocol",
		"type": "row"
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 0,
		  "y": 42
		},
		"id": 20,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum(galley_mcp_source_clients_total)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "Clients",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Connected Clients",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 8,
		  "y": 42
		},
		"id": 22,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "sum by(collection)(irate(galley_mcp_source_request_acks_total[1m]) * 60)",
			"format": "time_series",
			"intervalFactor": 1,
			"legendFormat": "",
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Request ACKs",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "ACKs/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  },
	  {
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "Prometheus",
		"fill": 1,
		"gridPos": {
		  "h": 6,
		  "w": 8,
		  "x": 16,
		  "y": 42
		},
		"id": 26,
		"legend": {
		  "avg": false,
		  "current": false,
		  "max": false,
		  "min": false,
		  "show": true,
		  "total": false,
		  "values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"paceLength": 10,
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"seriesOverrides": [],
		"spaceLength": 10,
		"stack": false,
		"steppedLine": false,
		"targets": [
		  {
			"expr": "rate(galley_mcp_source_request_nacks_total[1m]) * 60",
			"format": "time_series",
			"intervalFactor": 1,
			"refId": "A"
		  }
		],
		"thresholds": [],
		"timeFrom": null,
		"timeRegions": [],
		"timeShift": null,
		"title": "Request NACKs",
		"tooltip": {
		  "shared": true,
		  "sort": 0,
		  "value_type": "individual"
		},
		"type": "graph",
		"xaxis": {
		  "buckets": null,
		  "mode": "time",
		  "name": null,
		  "show": true,
		  "values": []
		},
		"yaxes": [
		  {
			"format": "short",
			"label": "NACKs/min",
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  },
		  {
			"format": "short",
			"label": null,
			"logBase": 1,
			"max": null,
			"min": null,
			"show": true
		  }
		],
		"yaxis": {
		  "align": false,
		  "alignLevel": null
		}
	  }
	],
	"refresh": "5s",
	"schemaVersion": 18,
	"style": "dark",
	"tags": [],
	"templating": {
	  "list": []
	},
	"time": {
	  "from": "now-5m",
	  "to": "now"
	},
	"timepicker": {
	  "refresh_intervals": [
		"5s",
		"10s",
		"30s",
		"1m",
		"5m",
		"15m",
		"30m",
		"1h",
		"2h",
		"1d"
	  ],
	  "time_options": [
		"5m",
		"15m",
		"1h",
		"6h",
		"12h",
		"24h",
		"2d",
		"7d",
		"30d"
	  ]
	},
	"timezone": "",
	"title": "Istio Galley Dashboard",
	"uid": "TSEY6jLmk",
	"version": 1
  }
`

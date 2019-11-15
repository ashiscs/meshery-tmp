package models

import (
	"encoding/gob"

	"github.com/grafana-tools/sdk"
)

// K8SConfig represents all the k8s session config
type K8SConfig struct {
	InClusterConfig   bool   `json:"inClusterConfig,omitempty"`
	K8Sfile           string `json:"k8sfile,omitempty"`
	Config            []byte `json:"config,omitempty"`
	Server            string `json:"configuredServer,omitempty"`
	ContextName       string `json:"contextName,omitempty"`
	ClusterConfigured bool   `json:"clusterConfigured,omitempty"`
	// ConfiguredServer  string `json:"configuredServer,omitempty"`
}

// K8SContext is just used to send contexts to the UI
type K8SContext struct {
	ContextName string `json:"contextName"`
	ClusterName string `json:"clusterName"`
	// ContextDisplayName string `json:"context-display-name"`
	IsCurrentContext bool `json:"currentContext"`
}

// Grafana represents the Grafana session config
type Grafana struct {
	GrafanaURL    string `json:"grafanaURL,omitempty"`
	GrafanaAPIKey string `json:"grafanaAPIKey,omitempty"`
	// GrafanaBoardSearch string          `json:"grafanaBoardSearch,omitempty"`
	GrafanaBoards []*SelectedGrafanaConfig `json:"selectedBoardsConfigs,omitempty"`
}

// SelectedGrafanaConfig represents the selected boards, panels, and template variables
type SelectedGrafanaConfig struct {
	GrafanaBoard         *GrafanaBoard `json:"board,omitempty"`
	GrafanaPanels        []*sdk.Panel  `json:"panels,omitempty"`
	SelectedTemplateVars []string      `json:"templateVars,omitempty"`
}

// Prometheus represents the prometheus session config
type Prometheus struct {
	PrometheusURL                   string                   `json:"prometheusURL,omitempty"`
	SelectedPrometheusBoardsConfigs []*SelectedGrafanaConfig `json:"selectedPrometheusBoardsConfigs,omitempty"`
}

// Session represents the data stored in session / local DB
type Session struct {
	// User         *User       `json:"user,omitempty"`
	K8SConfig    *K8SConfig  `json:"k8sConfig,omitempty"`
	MeshAdapters []*Adapter  `json:"meshAdapters,omitempty"`
	Grafana      *Grafana    `json:"grafana,omitempty"`
	Prometheus   *Prometheus `json:"prometheus,omitempty"`
}

func init() {
	gob.Register(&Session{})
	gob.Register(map[string]interface{}{})
}

// SessionPersister defines methods for a session persister
type SessionPersister interface {
	Read(userID string) (*Session, error)
	Write(userID string, data *Session) error
	Delete(userID string) error

	// Lock(userID string)
	// Unlock(userID string)
	Close()
}

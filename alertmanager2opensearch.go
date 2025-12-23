package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const supportedWebhookVersion = "4"

type (
	AlertmanagerOpenSearchExporter struct {
		openSearchClient    *opensearchapi.Client
		openSearchIndexName string

		prometheus struct {
			alertsReceived   *prometheus.CounterVec
			alertsInvalid    *prometheus.CounterVec
			alertsSuccessful *prometheus.CounterVec
		}
	}

	AlertmanagerEntry struct {
		Alerts []struct {
			Annotations  map[string]string `json:"annotations"`
			EndsAt       time.Time         `json:"endsAt"`
			GeneratorURL string            `json:"generatorURL"`
			Labels       map[string]string `json:"labels"`
			StartsAt     time.Time         `json:"startsAt"`
			Status       string            `json:"status"`
		} `json:"alerts"`
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		CommonLabels      map[string]string `json:"commonLabels"`
		ExternalURL       string            `json:"externalURL"`
		GroupLabels       map[string]string `json:"groupLabels"`
		Receiver          string            `json:"receiver"`
		Status            string            `json:"status"`
		Version           string            `json:"version"`
		GroupKey          string            `json:"groupKey"`

		// Timestamp records when the alert notification was received
		Timestamp string `json:"@timestamp"`
	}
)

func (e *AlertmanagerOpenSearchExporter) Init() {
	e.prometheus.alertsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_received",
			Help: "alertmanager2es received alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsReceived)

	e.prometheus.alertsInvalid = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_invalid",
			Help: "alertmanager2es invalid alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsInvalid)

	e.prometheus.alertsSuccessful = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertmanager2es_alerts_successful",
			Help: "alertmanager2es successful stored alerts",
		},
		[]string{},
	)
	prometheus.MustRegister(e.prometheus.alertsSuccessful)
}

func (e *AlertmanagerOpenSearchExporter) ConnectOpenSearch(cfg opensearch.Config, indexName string) {
	var err error
	e.openSearchClient, err = opensearchapi.NewClient(opensearchapi.Config{Client: cfg})
	if err != nil {
		panic(err)
	}

	tries := 0
	for {
		_, err = e.openSearchClient.Info(context.Background(), &opensearchapi.InfoReq{})
		if err != nil {
			tries++
			if tries >= 5 {
				panic(err)
			} else {
				log.Info("Failed to connect to OpenSearch, retry...")
				time.Sleep(5 * time.Second)
				continue
			}
		}

		break
	}

	e.openSearchIndexName = indexName
}

func (e *AlertmanagerOpenSearchExporter) buildIndexName(createTime time.Time) string {
	ret := e.openSearchIndexName

	ret = strings.ReplaceAll(ret, "%y", createTime.Format("2006"))
	ret = strings.ReplaceAll(ret, "%m", createTime.Format("01"))
	ret = strings.ReplaceAll(ret, "%d", createTime.Format("02"))

	return ret
}

func (e *AlertmanagerOpenSearchExporter) HttpHandler(w http.ResponseWriter, r *http.Request) {
	e.prometheus.alertsReceived.WithLabelValues().Inc()

	if r.Body == nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := errors.New("got empty request body")
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	var msg AlertmanagerEntry
	err = json.Unmarshal(b, &msg)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	if msg.Version != supportedWebhookVersion {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := fmt.Errorf("do not understand webhook version %q, only version %q is supported", msg.Version, supportedWebhookVersion)
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	now := time.Now()
	msg.Timestamp = now.Format(time.RFC3339)

	incidentJson, _ := json.Marshal(msg)

	req := opensearchapi.IndexReq{
		Index: e.buildIndexName(now),
		Body:  bytes.NewReader(incidentJson),
	}
	_, err = e.openSearchClient.Index(context.Background(), req)
	if err != nil {
		e.prometheus.alertsInvalid.WithLabelValues().Inc()
		err := fmt.Errorf("unable to insert document in opensearch")
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Error(err)
		return
	}

	log.Debugf("received and stored alert: %v", msg.CommonLabels)
	e.prometheus.alertsSuccessful.WithLabelValues().Inc()
}

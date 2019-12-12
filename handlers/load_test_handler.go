package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/layer5io/meshery/helpers"
	"github.com/layer5io/meshery/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"fortio.org/fortio/periodic"
)

// LoadTestHandler runs the load test with the given parameters
func (h *Handler) LoadTestHandler(w http.ResponseWriter, req *http.Request, session *sessions.Session, user *models.User) {
	if req.Method != http.MethodPost && req.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	tokenVal, _ := session.Values[h.config.SaaSTokenName].(string)
	err := req.ParseForm()
	if err != nil {
		logrus.Errorf("Error: unable to parse form: %v", err)
		http.Error(w, "unable to process the received data", http.StatusForbidden)
		return
	}
	q := req.URL.Query()

	testName := q.Get("name")
	if testName == "" {
		logrus.Errorf("Error: name field is blank")
		http.Error(w, "Provide a name for the test.", http.StatusForbidden)
		return
	}
	meshName := q.Get("mesh")
	testUUID := q.Get("uuid")

	loadTestOptions := &models.LoadTestOptions{}

	tt, _ := strconv.Atoi(q.Get("t"))
	if tt < 1 {
		tt = 1
	}
	dur := ""
	switch strings.ToLower(q.Get("dur")) {
	case "h":
		dur = "h"
	case "m":
		dur = "m"
	// case "s":
	default:
		dur = "s"
	}
	loadTestOptions.Duration, err = time.ParseDuration(fmt.Sprintf("%d%s", tt, dur))
	if err != nil {
		logrus.Errorf("Error: unable to parse load test duration: %v", err)
		http.Error(w, "unable to process the received data", http.StatusForbidden)
		return
	}

	loadTestOptions.IsGRPC = false

	cc, _ := strconv.Atoi(q.Get("c"))
	if cc < 1 {
		cc = 1
	}
	loadTestOptions.HTTPNumThreads = cc

	loadTestURL := q.Get("url")
	ltURL, err := url.Parse(loadTestURL)
	if err != nil || !ltURL.IsAbs() {
		logrus.Errorf("unable to parse the provided load test url: %v", err)
		http.Error(w, "invalid load test URL", http.StatusBadRequest)
		return
	}
	loadTestOptions.URL = loadTestURL
	loadTestOptions.Name = testName

	qps, _ := strconv.ParseFloat(q.Get("qps"), 64)
	if qps < 0 {
		qps = 0
	}
	loadTestOptions.HTTPQPS = qps

	loadGenerator := q.Get("loadGenerator")

	switch loadGenerator {
	case "wrk2":
		loadTestOptions.LoadGenerator = models.Wrk2LG
	default:
		loadTestOptions.LoadGenerator = models.FortioLG
	}

	// q.Set("json", "on")

	// client := &http.Client{}
	// fortioURL, err := url.Parse(h.config.FortioURL)
	// if err != nil {
	// 	logrus.Errorf("unable to parse the provided fortio url: %v", err)
	// 	http.Error(w, "error while running load test", http.StatusInternalServerError)
	// 	return
	// }
	// fortioURL.RawQuery = q.Encode()
	// logrus.Infof("load test constructed url: %s", fortioURL.String())
	// fortioResp, err := client.Get(fortioURL.String())

	sessObj, err := h.config.SessionPersister.Read(user.UserID)
	if err != nil {
		logrus.Warn("Unable to read session from the session persister. Starting a new session.")
	}

	if sessObj == nil {
		sessObj = &models.Session{}
	}

	log := logrus.WithField("file", "load_test_handler")

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Error("Event streaming not supported.")
		http.Error(w, "Event streaming is not supported at the moment.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	notify := w.(http.CloseNotifier).CloseNotify()
	respChan := make(chan *models.LoadTestResponse, 100)
	endChan := make(chan struct{})
	defer close(endChan)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Recovered from panic: %v.", r)
			}
		}()
		for data := range respChan {
			bd, err := json.Marshal(data)
			if err != nil {
				logrus.Errorf("error: unable to marshal meshery result for shipping: %v", err)
				http.Error(w, "error while running load test", http.StatusInternalServerError)
				return
			}

			log.Debug("received new data on response channel")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", bd)
			if flusher != nil {
				flusher.Flush()
				log.Debugf("Flushed the messages on the wire...")
			}
		}
		endChan <- struct{}{}
		log.Debug("response channel closed")
	}()
	go func() {
		h.executeLoadTest(testName, meshName, tokenVal, testUUID, sessObj, loadTestOptions, respChan)
		close(respChan)
	}()
	select {
	case <-notify:
		log.Debugf("received signal to close connection and channels")
		break
	case <-endChan:
		log.Debugf("load test completed")
	}
}

func (h *Handler) executeLoadTest(testName, meshName, tokenVal, testUUID string, sessObj *models.Session, loadTestOptions *models.LoadTestOptions, respChan chan *models.LoadTestResponse) {
	respChan <- &models.LoadTestResponse{
		Status:  models.LoadTestInfo,
		Message: "Initiating load test . . . ",
	}
	// resultsMap, resultInst, err := helpers.FortioLoadTest(loadTestOptions)
	var (
		resultsMap map[string]interface{} 
		resultInst *periodic.RunnerResults
		err error
	)
	if loadTestOptions.LoadGenerator == models.Wrk2LG {
		resultsMap, resultInst, err = helpers.WRK2LoadTest(loadTestOptions)	
	} else {
		resultsMap, resultInst, err = helpers.FortioLoadTest(loadTestOptions)
	}
	if err != nil {
		msg := "error: unable to perform load test"
		err = errors.Wrap(err, msg)
		logrus.Error(err)
		respChan <- &models.LoadTestResponse{
			Status:  models.LoadTestError,
			Message: msg,
		}
		return
	}

	respChan <- &models.LoadTestResponse{
		Status:  models.LoadTestInfo,
		Message: "Load test completed, fetching metadata now",
	}

	if sessObj.K8SConfig != nil {
		nodesChan := make(chan []*models.K8SNode)
		versionChan := make(chan string)
		installedMeshesChan := make(chan map[string]string)

		go func() {
			var nodes []*models.K8SNode
			var err error
			if len(sessObj.K8SConfig.Nodes) == 0 {
				nodes, err = helpers.FetchKubernetesNodes(sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName)
				if err != nil {
					err = errors.Wrap(err, "unable to ping kubernetes")
					// logrus.Error(err)
					logrus.Warn(err)
					// return
				}
			}
			nodesChan <- nodes
		}()
		go func() {
			var serverVersion string
			var err error
			if sessObj.K8SConfig.ServerVersion == "" {
				serverVersion, err = helpers.FetchKubernetesVersion(sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName)
				if err != nil {
					err = errors.Wrap(err, "unable to ping kubernetes")
					// logrus.Error(err)
					logrus.Warn(err)
					// return
				}
			}
			versionChan <- serverVersion
		}()
		go func() {
			installedMeshes, err := helpers.ScanKubernetes(sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName)
			if err != nil {
				err = errors.Wrap(err, "unable to scan kubernetes")
				logrus.Warn(err)
			}
			installedMeshesChan <- installedMeshes
		}()

		sessObj.K8SConfig.Nodes = <-nodesChan
		sessObj.K8SConfig.ServerVersion = <-versionChan

		if sessObj.K8SConfig.ServerVersion != "" && len(sessObj.K8SConfig.Nodes) > 0 {
			resultsMap["kubernetes"] = map[string]interface{}{
				"server_version": sessObj.K8SConfig.ServerVersion,
				"nodes":          sessObj.K8SConfig.Nodes,
			}
		}
		installedMeshes := <-installedMeshesChan
		if len(installedMeshes) > 0 {
			resultsMap["detected-meshes"] = installedMeshes
		}
	}
	respChan <- &models.LoadTestResponse{
		Status:  models.LoadTestInfo,
		Message: "Obtained the needed metadatas, attempting to persist the result",
	}
	// // defer fortioResp.Body.Close()
	// // bd, err := ioutil.ReadAll(fortioResp.Body)
	// bd, err := json.Marshal(resp)
	// if err != nil {
	// 	logrus.Errorf("Error: unable to parse response from fortio: %v", err)
	// 	http.Error(w, "error while running load test", http.StatusInternalServerError)
	// 	return
	// }

	result := &models.MesheryResult{
		Name:   testName,
		Mesh:   meshName,
		Result: resultsMap,
	}
	// TODO: can we do something to prevent marshalling twice??
	bd, err := json.Marshal(result)
	if err != nil {
		msg := "error: unable to marshal meshery result for shipping"
		err = errors.Wrap(err, msg)
		logrus.Error(err)
		// http.Error(w, "error while running load test", http.StatusInternalServerError)
		respChan <- &models.LoadTestResponse{
			Status:  models.LoadTestError,
			Message: msg,
		}
		return
	}

	resultID, err := h.publishResultsToSaaS(h.config.SaaSTokenName, tokenVal, bd)
	if err != nil {
		// http.Error(w, "error while getting load test results", http.StatusInternalServerError)
		// return
		msg := "error: unable to persist the load test results"
		err = errors.Wrap(err, msg)
		logrus.Error(err)
		// http.Error(w, "error while running load test", http.StatusInternalServerError)
		respChan <- &models.LoadTestResponse{
			Status:  models.LoadTestError,
			Message: msg,
		}
		return
	}
	respChan <- &models.LoadTestResponse{
		Status:  models.LoadTestInfo,
		Message: "Done persisting the load test results.",
	}

	var promURL string
	if sessObj.Prometheus != nil {
		promURL = sessObj.Prometheus.PrometheusURL
	}

	logrus.Debugf("promURL: %s, testUUID: %s, resultID: %s", promURL, testUUID, resultID)
	if promURL != "" && testUUID != "" && resultID != "" {
		_ = h.task.Call(&models.SubmitMetricsConfig{
			TestUUID:  testUUID,
			ResultID:  resultID,
			PromURL:   promURL,
			StartTime: resultInst.StartTime,
			EndTime:   resultInst.StartTime.Add(resultInst.ActualDuration),
			TokenKey:  h.config.SaaSTokenName,
			TokenVal:  tokenVal,
		})
	}

	// w.Write(bd)
	respChan <- &models.LoadTestResponse{
		Status: models.LoadTestSuccess,
		Result: result,
	}
}

// CollectStaticMetrics is used for collecting static metrics from prometheus and submitting it to SaaS
func (h *Handler) CollectStaticMetrics(config *models.SubmitMetricsConfig) error {
	logrus.Debugf("initiating collecting prometheus static board metrics for test id: %s", config.TestUUID)
	ctx := context.Background()
	queries := h.config.QueryTracker.GetQueriesForUUID(ctx, config.TestUUID)
	queryResults := map[string]map[string]interface{}{}
	step := h.config.PrometheusClient.ComputeStep(ctx, config.StartTime, config.EndTime)
	for query, flag := range queries {
		if !flag {
			seriesData, err := h.config.PrometheusClient.QueryRangeUsingClient(ctx, config.PromURL, query, config.StartTime, config.EndTime, step)
			if err != nil {
				return err
			}
			queryResults[query] = map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"resultType": seriesData.Type(),
					"result":     seriesData,
				},
			}
			// sd, _ := json.Marshal(seriesData)
			// sd, _ := json.Marshal(queryResponse)
			// logrus.Debugf("Retrieved series data: %s", sd)
			h.config.QueryTracker.AddOrFlagQuery(ctx, config.TestUUID, query, true)
		}
	}

	board, err := h.config.PrometheusClient.GetClusterStaticBoard(ctx, config.PromURL)
	if err != nil {
		return err
	}
	// TODO: we are NOT persisting the Node metrics for now

	resultUUID, err := uuid.FromString(config.ResultID)
	if err != nil {
		logrus.Error(errors.Wrap(err, "error parsing result uuid"))
		return err
	}
	result := &models.MesheryResult{
		ID:                resultUUID,
		ServerMetrics:     queryResults,
		ServerBoardConfig: board,
	}
	sd, err := json.Marshal(result)
	if err != nil {
		logrus.Error(errors.Wrap(err, "error - unable to marshal meshery metrics for shipping"))
		return err
	}

	logrus.Debugf("Result: %s, size: %d", sd, len(sd))

	if err = h.publishMetricsToSaaS(config.TokenKey, config.TokenVal, sd); err != nil {
		return err
	}
	// now to remove all the queries for the uuid
	h.config.QueryTracker.RemoveUUID(ctx, config.TestUUID)
	return nil
}

func (h *Handler) publishMetricsToSaaS(tokenKey, tokenVal string, bd []byte) error {
	logrus.Infof("attempting to publish metrics to SaaS")
	bf := bytes.NewBuffer(bd)
	saasURL, _ := url.Parse(h.config.SaaSBaseURL + "/result/metrics")
	req, _ := http.NewRequest(http.MethodPut, saasURL.String(), bf)
	req.AddCookie(&http.Cookie{
		Name:     tokenKey,
		Value:    tokenVal,
		Path:     "/",
		HttpOnly: true,
		Domain:   saasURL.Hostname(),
	})
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		logrus.Errorf("unable to send metrics: %v", err)
		return err
	}
	if resp.StatusCode == http.StatusOK {
		logrus.Infof("metrics successfully published to SaaS")
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	bdr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("unable to read response body: %v", err)
		return err
	}
	logrus.Errorf("error while sending metrics: %s", bdr)
	return fmt.Errorf("error while sending metrics - Status code: %d, Body: %s", resp.StatusCode, bdr)
}

func (h *Handler) publishResultsToSaaS(tokenKey, tokenVal string, bd []byte) (string, error) {
	logrus.Infof("attempting to publish results to SaaS")
	bf := bytes.NewBuffer(bd)
	saasURL, _ := url.Parse(h.config.SaaSBaseURL + "/result")
	req, _ := http.NewRequest(http.MethodPost, saasURL.String(), bf)
	req.AddCookie(&http.Cookie{
		Name:     tokenKey,
		Value:    tokenVal,
		Path:     "/",
		HttpOnly: true,
		Domain:   saasURL.Hostname(),
	})
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		logrus.Errorf("unable to send results: %v", err)
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bdr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("unable to read response body: %v", err)
		return "", err
	}
	if resp.StatusCode == http.StatusCreated {
		logrus.Infof("results successfully published to SaaS")
		idMap := map[string]string{}
		if err = json.Unmarshal(bdr, &idMap); err != nil {
			logrus.Errorf("unable to unmarshal body: %v", err)
			return "", err
		}
		resultID, ok := idMap["id"]
		if ok {
			return resultID, nil
		}
		return "", nil
	}
	logrus.Errorf("error while sending results: %s", bdr)
	return "", fmt.Errorf("error while sending results - Status code: %d, Body: %s", resp.StatusCode, bdr)
}

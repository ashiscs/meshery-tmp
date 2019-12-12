package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"encoding/json"

	"github.com/gorilla/sessions"
	"github.com/layer5io/meshery/meshes"
	"github.com/layer5io/meshery/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventStreamHandler endpoint is used for streaming events to the frontend
func (h *Handler) EventStreamHandler(w http.ResponseWriter, req *http.Request, session *sessions.Session, user *models.User) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log := logrus.WithField("file", "events_streamer")

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

	// go func() {
	// 	<-notify
	// 	// an attempt to re-establish connection
	// 	// mClient, _ = meshes.CreateClient(req.Context(), k8sConfigBytes, contextName, meshLocationURL)
	// }()
	localMeshAdapters := map[string]*meshes.MeshClient{}
	localMeshAdaptersLock := &sync.Mutex{}

	respChan := make(chan []byte, 100)
	// defer close(respChan)

	newAdaptersChan := make(chan *meshes.MeshClient)
	// defer close(newAdaptersChan)

	go func() {
		for mClient := range newAdaptersChan {
			log.Debug("received a new mesh client, listening for events")
			go func() {
				listenForAdapterEvents(req.Context(), mClient, respChan, log)
				_ = mClient.Close()
			}()
		}
		log.Debug("new adapters channel closed")
	}()

	go func() {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		log.Errorf("Recovered from panic: %v.", r)
		// 	}
		// }()
		for data := range respChan {
			log.Debug("received new data on response channel")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
				log.Debugf("Flushed the messages on the wire...")
			}
		}
		log.Debug("response channel closed")
	}()

STOP:
	for {
		select {
		case <-notify:
			log.Debugf("received signal to close connection and channels")
			close(newAdaptersChan)
			close(respChan)
			break STOP
		default:
			sessObj, err := h.config.SessionPersister.Read(user.UserID)
			if err != nil {
				log.Warn("Unable to read session from the session persister. Starting with a new session.")
			}

			if sessObj == nil {
				sessObj = &models.Session{}
			}

			meshAdapters := sessObj.MeshAdapters
			if meshAdapters == nil {
				meshAdapters = []*models.Adapter{}
			}

			if sessObj.K8SConfig == nil || !sessObj.K8SConfig.InClusterConfig && (sessObj.K8SConfig.Config == nil || len(sessObj.K8SConfig.Config) == 0) {
				log.Debug("No valid Kubernetes config found.") // switching from Error to Debug to prevent it from filling up the logs
				// http.Error(w, `No valid Kubernetes config found.`, http.StatusBadRequest)
				// return
				localMeshAdaptersLock.Lock()
				for _, mcl := range localMeshAdapters {
					_ = mcl.Close()
				}
				localMeshAdapters = map[string]*meshes.MeshClient{}
				localMeshAdaptersLock.Unlock()
			} else {
				adaptersLen := len(meshAdapters)
				if adaptersLen == 0 {
					log.Debug("No valid mesh adapter(s) found.") // switching from Error to Debug to prevent it from filling up the logs
					// http.Error(w, `No valid mesh adapter(s) found.`, http.StatusBadRequest)
					// return
					localMeshAdaptersLock.Lock()
					for _, mcl := range localMeshAdapters {
						_ = mcl.Close()
					}
					localMeshAdapters = map[string]*meshes.MeshClient{}
					localMeshAdaptersLock.Unlock()
				} else {
					localMeshAdaptersLock.Lock()
					for _, ma := range meshAdapters {
						mClient, ok := localMeshAdapters[ma.Location]
						if !ok {
							mClient, err = meshes.CreateClient(req.Context(), sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName, ma.Location)
							if err == nil {
								localMeshAdapters[ma.Location] = mClient
							}
						}
						if mClient != nil {
							_, err = mClient.MClient.MeshName(req.Context(), &meshes.MeshNameRequest{})
							if err != nil {
								_ = mClient.Close()
								delete(localMeshAdapters, ma.Location)
							} else {
								if !ok { // reusing the map check, only when ok is false a new entry will be added
									newAdaptersChan <- mClient
								}
							}
						}
					}
					localMeshAdaptersLock.Unlock()
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	defer log.Debug("events handler closed")
}

func listenForAdapterEvents(ctx context.Context, mClient *meshes.MeshClient, respChan chan []byte, log *logrus.Entry) {
	log.Debugf("Received a stream client...")

	streamClient, err := mClient.MClient.StreamEvents(ctx, &meshes.EventsRequest{})
	if err != nil {
		err = errors.Wrapf(err, "There was an error connecting to the backend to get events.")
		log.Error(err)
		// errChan <- err
		// http.Error(w, "There was an error connecting to the backend to get events", http.StatusInternalServerError)
		return
	}

	for {
		log.Debugf("Waiting to receive events.")
		event, err := streamClient.Recv()
		if err != nil {
			if err == io.EOF {
				err = errors.Wrapf(err, "Event streaming ended.")
				log.Error(err)
				// errChan <- nil
				return
			}
			err = errors.Wrapf(err, "Event streaming ended with an unknown error.")
			log.Error(err)
			// http.Error(w, "streaming events was interrupted", http.StatusInternalServerError)
			// return
			// errChan <- err
			return
		}
		// log.Debugf("received an event: %+#v", event)
		log.Debugf("Received an event.")
		data, err := json.Marshal(event)
		if err != nil {
			err = errors.Wrapf(err, "Error marshalling event to json.")
			log.Error(err)
			// errChan <- err
			// log.Errorf(
			// http.Error(w, "error while sending event to client", http.StatusInternalServerError)
			return
		}
		respChan <- data
	}
}

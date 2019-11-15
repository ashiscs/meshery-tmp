package handlers

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/layer5io/meshery/meshes"
	"github.com/layer5io/meshery/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	gob.Register([]*models.Adapter{})
}

// GetAllAdaptersHandler is used to fetch all the adapters
func (h *Handler) GetAllAdaptersHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, err := h.config.SessionStore.Get(req, h.config.SessionName)
	if err != nil {
		logrus.Errorf("Error getting session: %v.", err)
		http.Error(w, "Unable to get session.", http.StatusUnauthorized)
		return
	}

	err = json.NewEncoder(w).Encode(h.config.AdapterTracker.GetAdapters(req.Context()))
	if err != nil {
		logrus.Errorf("Error marshalling data: %v.", err)
		http.Error(w, "Unable to retrieve the requested data.", http.StatusInternalServerError)
		return
	}
}

// MeshAdapterConfigHandler is used to persist adapter config
func (h *Handler) MeshAdapterConfigHandler(w http.ResponseWriter, req *http.Request) {
	session, err := h.config.SessionStore.Get(req, h.config.SessionName)
	if err != nil {
		logrus.Errorf("Error getting session: %v.", err)
		http.Error(w, "Unable to get session.", http.StatusUnauthorized)
		return
	}

	var user *models.User
	user, _ = session.Values["user"].(*models.User)

	// h.config.SessionPersister.Lock(user.UserID)
	// defer h.config.SessionPersister.Unlock(user.UserID)

	sessObj, err := h.config.SessionPersister.Read(user.UserID)
	if err != nil {
		logrus.Warn("Unable to read session from the session persister. Starting a new session.")
	}

	if sessObj == nil {
		sessObj = &models.Session{}
	}

	meshAdapters := sessObj.MeshAdapters
	if meshAdapters == nil {
		meshAdapters = []*models.Adapter{}
	}

	switch req.Method {
	case http.MethodPost:
		meshLocationURL := req.FormValue("meshLocationURL")

		logrus.Debugf("meshLocationURL: %s", meshLocationURL)
		if strings.TrimSpace(meshLocationURL) == "" {
			err := errors.New("meshLocationURL cannot be empty to add an adapter.")
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if sessObj.K8SConfig == nil || !sessObj.K8SConfig.InClusterConfig && (sessObj.K8SConfig.Config == nil || len(sessObj.K8SConfig.Config) == 0) {
			err := errors.New("no valid kubernetes config found")
			logrus.Error(err)
			http.Error(w, "No valid Kubernetes config found.", http.StatusBadRequest)
			return
		}

		meshAdapters, err = h.addAdapter(req.Context(), meshAdapters, sessObj, meshLocationURL)
		if err != nil {
			http.Error(w, "Unable to retrieve the requested data.", http.StatusInternalServerError)
			return // error is handled appropriately in the relevant method
		}
	case http.MethodDelete:
		meshAdapters, err = h.deleteAdapter(meshAdapters, w, req)
		if err != nil {
			return // error is handled appropriately in the relevant method
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	sessObj.MeshAdapters = meshAdapters
	err = h.config.SessionPersister.Write(user.UserID, sessObj)
	if err != nil {
		logrus.Errorf("Unable to save session: %v.", err)
		http.Error(w, "Unable to save session.", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(meshAdapters)
	if err != nil {
		logrus.Errorf("error marshalling data: %v.", err)
		http.Error(w, "Unable to retrieve the requested data.", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) addAdapter(ctx context.Context, meshAdapters []*models.Adapter, sessObj *models.Session, meshLocationURL string) ([]*models.Adapter, error) {
	alreadyConfigured := false
	for _, adapter := range meshAdapters {
		if adapter.Location == meshLocationURL {
			// err := errors.New("Adapter with the given meshLocationURL already exists.")
			// logrus.Error(err)
			// http.Error(w, err.Error(), http.StatusForbidden)
			// return nil, err
			alreadyConfigured = true
			break
		}
	}

	if alreadyConfigured {
		logrus.Debugf("Adapter already configured...")
		return meshAdapters, nil
	}

	mClient, err := meshes.CreateClient(ctx, sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName, meshLocationURL)
	if err != nil {
		err = errors.Wrapf(err, "Error creating a mesh client.")
		logrus.Error(err)
		// http.Error(w, "Unable to connect to the Mesh adapter using the given config, please try again", http.StatusInternalServerError)
		return nil, err
	}
	defer mClient.Close()
	respOps, err := mClient.MClient.SupportedOperations(ctx, &meshes.SupportedOperationsRequest{})
	if err != nil {
		logrus.Errorf("Error getting operations for the mesh: %v.", err)
		// http.Error(w, "unable to retrieve the requested data", http.StatusInternalServerError)
		return nil, err
	}

	meshNameOps, err := mClient.MClient.MeshName(ctx, &meshes.MeshNameRequest{})
	if err != nil {
		err = errors.Wrapf(err, "Error getting service mesh name.")
		logrus.Error(err)
		// http.Error(w, "unable to retrieve the requested data", http.StatusInternalServerError)
		return nil, err
	}

	result := &models.Adapter{
		Location: meshLocationURL,
		Name:     meshNameOps.GetName(),
		Ops:      respOps.GetOps(),
	}

	h.config.AdapterTracker.AddAdapter(ctx, meshLocationURL)

	return append(meshAdapters, result), nil
}

func (h *Handler) deleteAdapter(meshAdapters []*models.Adapter, w http.ResponseWriter, req *http.Request) ([]*models.Adapter, error) {

	adapterLoc := req.URL.Query().Get("adapter")
	logrus.Debugf("URL of adapter to be removed: %s.", adapterLoc)

	adaptersLen := len(meshAdapters)

	aID := -1
	for i, ad := range meshAdapters {
		if adapterLoc == ad.Location {
			aID = i
		}
	}
	if aID < 0 {
		err := errors.New("Unable to find a valid adapter for the given adapter URL.")
		logrus.Error(err)
		http.Error(w, "Given adapter URL is not valid.", http.StatusBadRequest)
		return nil, err
	}

	newMeshAdapters := []*models.Adapter{}
	if aID == 0 {
		newMeshAdapters = append(newMeshAdapters, meshAdapters[1:]...)
	} else if aID == adaptersLen-1 {
		newMeshAdapters = append(newMeshAdapters, meshAdapters[:adaptersLen-1]...)
	} else {
		newMeshAdapters = append(newMeshAdapters, meshAdapters[0:aID]...)
		newMeshAdapters = append(newMeshAdapters, meshAdapters[aID+1:]...)
	}
	if logrus.GetLevel() == logrus.DebugLevel {
		b, _ := json.Marshal(meshAdapters)
		logrus.Debugf("Old adapters: %s.", b)
		b, _ = json.Marshal(newMeshAdapters)
		logrus.Debugf("New adapters: %s.", b)
	}
	return newMeshAdapters, nil
}

// MeshOpsHandler is used to send operations to the adapters
func (h *Handler) MeshOpsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	session, err := h.config.SessionStore.Get(req, h.config.SessionName)
	if err != nil {
		logrus.Error("Unable to get session data.")
		http.Error(w, "Unable to get user data.", http.StatusUnauthorized)
		return
	}

	var user *models.User
	user, _ = session.Values["user"].(*models.User)

	// h.config.SessionPersister.Lock(user.UserID)
	// defer h.config.SessionPersister.Unlock(user.UserID)

	sessObj, err := h.config.SessionPersister.Read(user.UserID)
	if err != nil {
		logrus.Warn("Unable to read session from the session persister. Starting a new session.")
	}

	if sessObj == nil {
		sessObj = &models.Session{}
	}

	meshAdapters := sessObj.MeshAdapters
	if meshAdapters == nil {
		meshAdapters = []*models.Adapter{}
	}

	adapterLoc := req.PostFormValue("adapter")
	logrus.Debugf("Adapter URL to execute operations on: %s.", adapterLoc)

	aID := -1
	for i, ad := range meshAdapters {
		if adapterLoc == ad.Location {
			aID = i
		}
	}
	if aID < 0 {
		err := errors.New("Unable to find a valid adapter for the given adapter URL.")
		logrus.Error(err)
		http.Error(w, "Adapter could not be pinged.", http.StatusBadRequest)
		return
	}

	opName := req.PostFormValue("query")
	customBody := req.PostFormValue("customBody")
	namespace := req.PostFormValue("namespace")
	delete := req.PostFormValue("deleteOp")
	if namespace == "" {
		namespace = "default"
	}

	if sessObj.K8SConfig == nil || !sessObj.K8SConfig.InClusterConfig && (sessObj.K8SConfig.Config == nil || len(sessObj.K8SConfig.Config) == 0) {
		logrus.Error("No valid kubernetes config found.")
		http.Error(w, `No valid kubernetes config found.`, http.StatusBadRequest)
		return
	}

	mClient, err := meshes.CreateClient(req.Context(), sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName, meshAdapters[aID].Location)
	if err != nil {
		logrus.Errorf("Error creating a mesh client: %v.", err)
		http.Error(w, "Unable to create a mesh client.", http.StatusBadRequest)
		return
	}
	defer mClient.Close()

	operationId, err := uuid.NewV4()
	if err != nil {
		logrus.Errorf("Error generating an operation id: %v.", err)
		http.Error(w, "Error generating an operation id.", http.StatusInternalServerError)
		return
	}

	_, err = mClient.MClient.ApplyOperation(req.Context(), &meshes.ApplyRuleRequest{
		OperationId: operationId.String(),
		OpName:      opName,
		Username:    user.UserID,
		Namespace:   namespace,
		CustomBody:  customBody,
		DeleteOp:    (delete != ""),
	})
	if err != nil {
		logrus.Error(err)
		http.Error(w, "There was an error applying the change.", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("{}"))
}

// AdapterPingHandler is used to ping a given adapter
func (h *Handler) AdapterPingHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	session, err := h.config.SessionStore.Get(req, h.config.SessionName)
	if err != nil {
		logrus.Error("Unable to get session data.")
		http.Error(w, "Unable to get user data.", http.StatusUnauthorized)
		return
	}

	var user *models.User
	user, _ = session.Values["user"].(*models.User)

	sessObj, err := h.config.SessionPersister.Read(user.UserID)
	if err != nil {
		logrus.Warn("Unable to read session from the session persister. Starting a new session.")
	}

	if sessObj == nil {
		sessObj = &models.Session{}
	}

	meshAdapters := sessObj.MeshAdapters
	if meshAdapters == nil {
		meshAdapters = []*models.Adapter{}
	}

	// adapterLoc := req.PostFormValue("adapter")
	adapterLoc := req.URL.Query().Get("adapter")
	logrus.Debugf("Adapter url to ping: %s.", adapterLoc)

	aID := -1
	for i, ad := range meshAdapters {
		if adapterLoc == ad.Location {
			aID = i
		}
	}
	if aID < 0 {
		err := errors.New("Unable to find a valid adapter for the given adapter URL.")
		logrus.Error(err)
		http.Error(w, "Adapter could not be pinged.", http.StatusBadRequest)
		return
	}

	if sessObj.K8SConfig == nil || !sessObj.K8SConfig.InClusterConfig && (sessObj.K8SConfig.Config == nil || len(sessObj.K8SConfig.Config) == 0) {
		logrus.Error("No valid kubernetes config found.")
		http.Error(w, `No valid kubernetes config found.`, http.StatusBadRequest)
		return
	}

	mClient, err := meshes.CreateClient(req.Context(), sessObj.K8SConfig.Config, sessObj.K8SConfig.ContextName, meshAdapters[aID].Location)
	if err != nil {
		logrus.Errorf("Error creating a mesh client: %v.", err)
		http.Error(w, "Adapter could not be pinged.", http.StatusBadRequest)
		return
	}
	defer mClient.Close()

	_, err = mClient.MClient.MeshName(req.Context(), &meshes.MeshNameRequest{})
	if err != nil {
		err = errors.Wrapf(err, "Error pinging service mesh adapter.")
		logrus.Error(err)
		http.Error(w, "Adapter could not be pinged.", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("{}"))
}

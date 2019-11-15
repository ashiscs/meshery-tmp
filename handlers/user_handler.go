package handlers

import (
	"net/http"

	"encoding/json"

	"github.com/layer5io/meshery/models"
	"github.com/sirupsen/logrus"
)

// UserHandler returns info about the logged in user
func (h *Handler) UserHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	session, err := h.config.SessionStore.Get(req, h.config.SessionName)
	if err != nil {
		logrus.Errorf("error getting session: %v", err)
		http.Error(w, "unable to get session", http.StatusUnauthorized)
		return
	}

	var user *models.User
	user, _ = session.Values["user"].(*models.User)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		logrus.Errorf("error getting user data: %v", err)
		http.Error(w, "unable to get session", http.StatusInternalServerError)
		return
	}
}

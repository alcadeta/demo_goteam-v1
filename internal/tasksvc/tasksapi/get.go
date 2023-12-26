package tasksapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db"
	"github.com/kxplxn/goteam/pkg/db/tasktbl"
	"github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/validator"
)

// GetResp defines the body of GET tasks responses.
type GetResp []tasktbl.Task

// GetHandler is an api.MethodHandler that can handle GET requests sent to the
// tasks route.
type GetHandler struct {
	boardIDValidator validator.String
	authDecoder      cookie.Decoder[cookie.Auth]
	retriever        db.Retriever[[]tasktbl.Task]
	log              log.Errorer
}

// NewGetHandler creates and returns a new GetHandler.
func NewGetHandler(
	boardIDValidator validator.String,
	authDecoder cookie.Decoder[cookie.Auth],
	retriever db.Retriever[[]tasktbl.Task],
	log log.Errorer,
) GetHandler {
	return GetHandler{
		boardIDValidator: boardIDValidator,
		authDecoder:      authDecoder,
		retriever:        retriever,
		log:              log,
	}
}

// Handle handles GET requests sent to the tasks route.
func (h GetHandler) Handle(w http.ResponseWriter, r *http.Request, _ string) {
	// validate board id
	boardID := r.URL.Query().Get("boardID")
	if err := h.boardIDValidator.Validate(boardID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// get auth token
	ckAuth, err := r.Cookie(cookie.AuthName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// decode auth token
	auth, err := h.authDecoder.Decode(*ckAuth)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// retrieve tasks
	tasks, err := h.retriever.Retrieve(r.Context(), auth.TeamID)
	if errors.Is(err, db.ErrNoItem) {
		// if no items, set tasks to empty slice
		tasks = []tasktbl.Task{}
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// write response
	if err = json.NewEncoder(w).Encode(tasks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}
}

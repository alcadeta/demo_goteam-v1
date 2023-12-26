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
	stateDecoder     cookie.Decoder[cookie.State]
	retrieverByBoard db.Retriever[[]tasktbl.Task]
	authDecoder      cookie.Decoder[cookie.Auth]
	retrieverByTeam  db.Retriever[[]tasktbl.Task]
	log              log.Errorer
}

// NewGetHandler creates and returns a new GetHandler.
func NewGetHandler(
	boardIDValidator validator.String,
	stateDecoder cookie.Decoder[cookie.State],
	retrieverByBoard db.Retriever[[]tasktbl.Task],
	authDecoder cookie.Decoder[cookie.Auth],
	retrieverByTeam db.Retriever[[]tasktbl.Task],
	log log.Errorer,
) GetHandler {
	return GetHandler{
		boardIDValidator: boardIDValidator,
		stateDecoder:     stateDecoder,
		retrieverByBoard: retrieverByBoard,
		authDecoder:      authDecoder,
		retrieverByTeam:  retrieverByTeam,
		log:              log,
	}
}

// Handle handles GET requests sent to the tasks route.
func (h GetHandler) Handle(w http.ResponseWriter, r *http.Request, _ string) {
	if boardID := r.URL.Query().Get("boardID"); boardID != "" {
		// if board ID was present, retrive tasks by board ID
		h.getByBoardID(w, r, boardID)
	} else {
		// if board ID was not present, retrieve tasks by team ID
		h.getByTeamID(w, r)
	}
}

// getByBoardID validates the board ID and retrieves all tasks for the board,
// writing them to the response.
func (h GetHandler) getByBoardID(
	w http.ResponseWriter, r *http.Request, boardID string,
) {
	if err := h.boardIDValidator.Validate(boardID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// get auth token
	ckState, err := r.Cookie(cookie.StateName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// decode state token
	state, err := h.stateDecoder.Decode(*ckState)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// validate board access
	var hasAccess bool
	for _, b := range state.Boards {
		if b.ID == boardID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// retrieve tasks
	tasks, err := h.retrieverByBoard.Retrieve(r.Context(), boardID)
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

// getByTeamID gets the team ID from the auth token, retrieves all tasks for
// the team, and writes the ones with the first task's board ID to the response.
func (h GetHandler) getByTeamID(w http.ResponseWriter, r *http.Request) {
	// get auth token
	ckAuth, err := r.Cookie(cookie.AuthName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// decode state token
	auth, err := h.authDecoder.Decode(*ckAuth)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// retrieve tasks
	tasks, err := h.retrieverByTeam.Retrieve(r.Context(), auth.TeamID)
	if errors.Is(err, db.ErrNoItem) {
		// if no items, set tasks to empty slice
		tasks = []tasktbl.Task{}
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// if more than one task, only return the ones with the first task's board
	// ID
	if len(tasks) > 1 {
		singleBoardTasks := []tasktbl.Task{}
		var boardID string
		for _, t := range tasks {
			switch boardID {
			case "":
				boardID = t.BoardID
				singleBoardTasks = append(singleBoardTasks, t)
			case t.BoardID:
				singleBoardTasks = append(singleBoardTasks, t)
			}
		}
		tasks = singleBoardTasks
	}

	// write response
	if err = json.NewEncoder(w).Encode(tasks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}
}

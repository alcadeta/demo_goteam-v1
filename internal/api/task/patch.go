package task

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kxplxn/goteam/internal/api"
	"github.com/kxplxn/goteam/pkg/db"
	taskTable "github.com/kxplxn/goteam/pkg/db/task"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/token"
)

// PatchReq defines the body of PATCH task requests.
type PatchReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Subtasks    []struct {
		Title  string `json:"title"`
		IsDone bool   `json:"done"`
	} `json:"subtasks"`
}

// PatchResp defines the body of PATCH task responses.
type PatchResp struct {
	Error string `json:"error"`
}

// PatchHandler handles PATCH requests sent to the task route.
type PatchHandler struct {
	decodeAuth         token.DecodeFunc[token.Auth]
	decodeState        token.DecodeFunc[token.State]
	titleValidator     api.StringValidator
	subtTitleValidator api.StringValidator
	taskUpdater        db.Updater[taskTable.Task]
	log                pkgLog.Errorer
}

// NewPatchHandler returns a new PatchHandler.
func NewPatchHandler(
	decodeAuth token.DecodeFunc[token.Auth],
	decodeState token.DecodeFunc[token.State],
	taskTitleValidator api.StringValidator,
	subtaskTitleValidator api.StringValidator,
	taskUpdater db.Updater[taskTable.Task],
	log pkgLog.Errorer,
) *PatchHandler {
	return &PatchHandler{
		decodeAuth:         decodeAuth,
		decodeState:        decodeState,
		titleValidator:     taskTitleValidator,
		subtTitleValidator: subtaskTitleValidator,
		taskUpdater:        taskUpdater,
		log:                log,
	}
}

// Handle handles PATCH requests sent to the task route.
func (h *PatchHandler) Handle(
	w http.ResponseWriter, r *http.Request, username string,
) {
	// get auth token
	ckAuth, err := r.Cookie(token.AuthName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		if encodeErr := json.NewEncoder(w).Encode(PostResp{
			Error: "Auth token not found.",
		}); encodeErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}

	// decode auth token
	auth, err := h.decodeAuth(ckAuth.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if err = json.NewEncoder(w).Encode(DeleteResp{
			Error: "Invalid auth token.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
			return
		}
	}

	// validate user is admin
	if !auth.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Only team admins can edit tasks.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// get state token
	ckState, err := r.Cookie(token.StateName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusBadRequest)
		if err = json.NewEncoder(w).Encode(PatchResp{
			Error: "State token not found.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}

	// decode state token
	state, err := h.decodeState(ckState.Value)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err = json.NewEncoder(w).Encode(PatchResp{
			Error: "Invalid state token.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// validate id exists in state and determine location
	id := r.URL.Query().Get("id")
	var (
		idFound bool
		boardID string
		colNo   int
		order   int
	)
	for _, b := range state.Boards {
		for i, c := range b.Columns {
			for j, t := range c.Tasks {
				if t.ID == id {
					idFound = true
					order = j
					break
				}
			}
			if idFound {
				colNo = i
				break
			}
		}
		if idFound {
			boardID = b.ID
			break
		}
	}
	if !idFound {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Invalid task ID.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// read request body
	var reqBody PatchReq
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}

	// validate task title
	if err := h.titleValidator.Validate(reqBody.Title); err != nil {
		var errMsg string
		if errors.Is(err, api.ErrEmpty) {
			errMsg = "Task title cannot be empty."
		} else if errors.Is(err, api.ErrTooLong) {
			errMsg = "Task title cannot be longer than 50 characters."
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: errMsg,
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// validate subtask titles
	var subtasks []taskTable.Subtask
	for _, subtask := range reqBody.Subtasks {
		if err := h.subtTitleValidator.Validate(subtask.Title); err != nil {
			var errMsg string
			if errors.Is(err, api.ErrEmpty) {
				errMsg = "Subtask title cannot be empty."
			} else if errors.Is(err, api.ErrTooLong) {
				errMsg = "Subtask title cannot be longer than 50 characters."
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err.Error())
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(PatchResp{
				Error: errMsg,
			}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err.Error())
			}
			return
		}
		subtasks = append(
			subtasks,
			taskTable.NewSubtask(subtask.Title, subtask.IsDone),
		)
	}

	// Update the task and subtasks in the database.
	if err = h.taskUpdater.Update(r.Context(), taskTable.NewTask(
		id, reqBody.Title, reqBody.Description, order, subtasks, boardID, colNo,
	)); errors.Is(err, db.ErrNoItem) {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Task not found.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return

	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}
}
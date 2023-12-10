package task

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kxplxn/goteam/internal/api"
	"github.com/kxplxn/goteam/pkg/dbaccess"
	boardTable "github.com/kxplxn/goteam/pkg/dbaccess/board"
	columnTable "github.com/kxplxn/goteam/pkg/dbaccess/column"
	taskTable "github.com/kxplxn/goteam/pkg/dbaccess/task"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/token"
)

// PatchReq defines the body of PATCH task requests.
type PatchReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Subtasks    []struct {
		Title  string `json:"title"`
		Order  int    `json:"order"`
		IsDone bool   `json:"done"`
	} `json:"subtasks"`
}

// PatchResp defines the body of PATCH task responses.
type PatchResp struct {
	Error string `json:"error"`
}

// PatchHandler handles PATCH requests sent to the task route.
type PatchHandler struct {
	decodeAuth            token.DecodeFunc[token.Auth]
	decodeState           token.DecodeFunc[token.State]
	taskTitleValidator    api.StringValidator
	subtaskTitleValidator api.StringValidator
	taskSelector          dbaccess.Selector[taskTable.Record]
	columnSelector        dbaccess.Selector[columnTable.Record]
	boardSelector         dbaccess.Selector[boardTable.Record]
	taskUpdater           dbaccess.Updater[taskTable.UpRecord]
	log                   pkgLog.Errorer
}

// NewPatchHandler returns a new PatchHandler.
func NewPatchHandler(
	decodeAuth token.DecodeFunc[token.Auth],
	decodeState token.DecodeFunc[token.State],
	taskTitleValidator api.StringValidator,
	subtaskTitleValidator api.StringValidator,
	taskSelector dbaccess.Selector[taskTable.Record],
	columnSelector dbaccess.Selector[columnTable.Record],
	boardSelector dbaccess.Selector[boardTable.Record],
	taskUpdater dbaccess.Updater[taskTable.UpRecord],
	log pkgLog.Errorer,
) *PatchHandler {
	return &PatchHandler{
		decodeAuth:            decodeAuth,
		decodeState:           decodeState,
		taskTitleValidator:    taskTitleValidator,
		subtaskTitleValidator: subtaskTitleValidator,
		taskSelector:          taskSelector,
		columnSelector:        columnSelector,
		boardSelector:         boardSelector,
		taskUpdater:           taskUpdater,
		log:                   log,
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

	// validate id exists in state
	id := r.URL.Query().Get("id")
	var idFound bool
	for _, b := range state.Boards {
		for _, c := range b.Columns {
			for _, t := range c.Tasks {
				if t.ID == id {
					idFound = true
				}
				if idFound {
					break
				}
			}
			if idFound {
				break
			}
		}
		if idFound {
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
	if err := h.taskTitleValidator.Validate(reqBody.Title); err != nil {
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

	// Validate subtask titles and transform them into db-insertable types.
	var subtaskRecords []taskTable.Subtask
	for _, subtask := range reqBody.Subtasks {
		if err := h.subtaskTitleValidator.Validate(subtask.Title); err != nil {
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
		subtaskRecords = append(
			subtaskRecords,
			taskTable.NewSubtask(subtask.Title, subtask.Order, subtask.IsDone),
		)
	}

	// Select the task in the database to get its columnID.
	task, err := h.taskSelector.Select(id)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Task not found.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}

	// Get the column from the database to access its board ID for
	// authorization.
	column, err := h.columnSelector.Select(strconv.Itoa(task.ColumnID))
	if err != nil {
		// Return 500 on any error (even sql.ErrNoRows) because if task was
		// found, so must the column because the columnID is a foreign key for
		// the column table.
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}

	// Validate that the board belongs to the team that the user is the admin
	// of.
	board, err := h.boardSelector.Select(strconv.Itoa(column.BoardID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return

	}
	if strconv.Itoa(board.TeamID) != auth.TeamID {
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "You do not have access to this board.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// Update the task and subtasks in the database.
	if err = h.taskUpdater.Update(id, taskTable.NewUpRecord(
		reqBody.Title, reqBody.Description, subtaskRecords,
	)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err.Error())
		return
	}
}

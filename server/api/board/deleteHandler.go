package board

import (
	"database/sql"
	"net/http"

	"server/db"
	"server/log"
)

// DeleteHandler is a MethodHandler that is intended to handle DELETE requests
// sent to the board endpoint.
type DeleteHandler struct {
	userBoardSelector db.RelSelector[bool]
	boardDeleter      db.Deleter
	logger            log.Logger
}

// NewDeleteHandler creates and returns a new DeleteHandler.
func NewDeleteHandler(
	userBoardSelector db.RelSelector[bool],
	boardDeleter db.Deleter,
	logger log.Logger,
) DeleteHandler {
	return DeleteHandler{
		userBoardSelector: userBoardSelector,
		boardDeleter:      boardDeleter,
		logger:            logger,
	}
}

// Handle handles the DELETE requests sent to the board endpoint.
func (h DeleteHandler) Handle(
	w http.ResponseWriter, r *http.Request, username string,
) {
	// Get id query parameter. That's our board ID.
	boardID := r.URL.Query().Get("id")

	// Validate that the user making the request is the admin of the board to be
	// deleted.
	isAdmin, err := h.userBoardSelector.Select(username, boardID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		h.logger.Log(log.LevelError, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if isAdmin == false {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Delete the board.
	if err = h.boardDeleter.Delete(boardID); err != nil {
		h.logger.Log(log.LevelError, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// All went well. Return 200.
	w.WriteHeader(http.StatusOK)
	return
}
package board

import (
	"context"
	"database/sql"
	"errors"
)

// Deleter can be used to delete a record from the board table recursively.
type Deleter struct{ db *sql.DB }

// NewDeleter creates and returns a new Deleter.
func NewDeleter(db *sql.DB) Deleter { return Deleter{db: db} }

// Delete deletes a board recursively, meaning that alongside the record from
// the board table with the given ID, it will delete all records from the
// user_board table and columns table associated with the board, all records
// from the tasks table associated with those columns, and all records from the
// subtasks table associated with those tasks.
func (d Deleter) Delete(id string) error {
	// Begin transaction with new empty context.
	ctx := context.Background()
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Delete all records from the user_board table with the given board ID.
	if _, err = tx.ExecContext(
		ctx, "DELETE FROM app.user_board WHERE boardID = $1", id,
	); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	// Get IDs of this board's columns so that we can delete the tasks
	// associated to them.
	var columnIDs []int
	if rows, selectErr := tx.QueryContext(
		ctx, `SELECT id FROM app."column" WHERE boardID = $1`, id,
	); selectErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(selectErr, rollbackErr)
		}
		return selectErr
	} else {
		var columnID int
		for rows.Next() {
			if scanErr := rows.Scan(&columnID); scanErr != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					return errors.Join(scanErr, rollbackErr)
				}
				return scanErr
			}
			columnIDs = append(columnIDs, columnID)
		}
	}

	// Get IDs of each column's tasks so that we can delete the subtasks
	// associated to them.
	var taskIDs []int
	for _, columnID := range columnIDs {
		if rows, selectErr := tx.QueryContext(
			ctx, `SELECT id FROM app.task WHERE columnID = $1`, columnID,
		); selectErr != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.Join(selectErr, rollbackErr)
			}
			return selectErr
		} else {
			var taskID int
			for rows.Next() {
				if scanErr := rows.Scan(&taskID); scanErr != nil {
					if rollbackErr := tx.Rollback(); rollbackErr != nil {
						return errors.Join(scanErr, rollbackErr)
					}
					return scanErr
				}
				taskIDs = append(taskIDs, taskID)
			}
		}
	}

	// Delete each subtask associated with each task.
	for _, taskID := range taskIDs {
		if _, err = tx.ExecContext(
			ctx, "DELETE FROM app.subtask WHERE taskID = $1", taskID,
		); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.Join(err, rollbackErr)
			}
			return err
		}
	}

	// Delete each task associated with each column.
	for _, columnID := range columnIDs {
		if _, err = tx.ExecContext(
			ctx, "DELETE FROM app.task WHERE columnID = $1", columnID,
		); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.Join(err, rollbackErr)
			}
			return err
		}
	}

	// Delete all records from the column table with the given board ID.
	if _, err = tx.ExecContext(
		ctx, `DELETE FROM app."column" WHERE boardID = $1`, id,
	); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	// Delete the record from the board table with the given board ID.
	if _, err = tx.ExecContext(
		ctx, "DELETE FROM app.board WHERE id = $1", id,
	); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

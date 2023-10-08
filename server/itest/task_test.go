//go:build itest

package itest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/api"
	taskAPI "server/api/task"
	"server/assert"
	"server/auth"
	columnTable "server/dbaccess/column"
	userboardTable "server/dbaccess/userboard"
	pkgLog "server/log"
)

func TestTaskAPI(t *testing.T) {
	sut := api.NewHandler(
		auth.NewBearerTokenReader(),
		auth.NewJWTValidator(jwtKey),
		map[string]api.MethodHandler{
			http.MethodPost: taskAPI.NewPOSTHandler(
				taskAPI.NewTitleValidator(),
				taskAPI.NewTitleValidator(),
				columnTable.NewSelector(db),
				userboardTable.NewSelector(db),
				pkgLog.New(),
			),
		},
	)

	t.Run("Auth", func(t *testing.T) {
		for _, c := range []struct {
			name     string
			authFunc func(*http.Request)
		}{
			// Auth Cases
			{name: "HeaderEmpty", authFunc: func(*http.Request) {}},
			{name: "HeaderInvalid", authFunc: addBearerAuth("asdfasldfkjasd")},
		} {
			t.Run(c.name, func(t *testing.T) {
				t.Run(http.MethodPost, func(t *testing.T) {
					req, err := http.NewRequest(http.MethodPost, "", nil)
					if err != nil {
						t.Fatal(err)
					}
					c.authFunc(req)
					w := httptest.NewRecorder()

					sut.ServeHTTP(w, req)
					res := w.Result()

					if err = assert.Equal(
						http.StatusUnauthorized, res.StatusCode,
					); err != nil {
						t.Error(err)
					}

					if err = assert.Equal(
						"Bearer", res.Header.Values("WWW-Authenticate")[0],
					); err != nil {
						t.Error(err)
					}
				})
			})
		}
	})

	for _, c := range []struct {
		name           string
		task           map[string]any
		wantStatusCode int
		wantErrMsg     string
	}{
		{
			name: "TaskTitleEmpty",
			task: map[string]any{
				"title":       "",
				"description": "",
				"column":      0,
				"subtasks":    []map[string]any{},
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "Task title cannot be empty.",
		},
		{
			name: "TaskTitleTooLong",
			task: map[string]any{
				"title": "asdqweasdqweasdqweasdqweasdqweasdqweasdqweasdqwe" +
					"asd",
				"description": "",
				"column":      0,
				"subtasks":    []map[string]any{},
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "Task title cannot be longer than 50 characters.",
		},
		{
			name: "SubtaskTitleEmpty",
			task: map[string]any{
				"title":       "Some Task",
				"description": "",
				"column":      0,
				"subtasks": []map[string]any{
					{"title": ""},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg:     "Subtask title cannot be empty.",
		},
		{
			name: "SubtaskTitleTooLong",
			task: map[string]any{
				"title":       "Some Task",
				"description": "",
				"column":      0,
				"subtasks": []map[string]any{
					{
						"title": "asdqweasdqweasdqweasdqweasdqweasdqweasdqwea" +
							"sdqweasd",
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
			wantErrMsg: "Subtask title cannot be longer than 50 " +
				"characters.",
		},
		{
			name: "ColumnNotFound",
			task: map[string]any{
				"title":       "Some Task",
				"description": "",
				"column":      1001,
				"subtasks":    []map[string]any{},
			},
			wantStatusCode: http.StatusNotFound,
			wantErrMsg:     "Column not found.",
		},
		{
			name: "NoAccess",
			task: map[string]any{
				"title":       "Some Task",
				"description": "",
				"column":      8,
				"subtasks":    []map[string]any{},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantErrMsg:     "You do not have access to this board.",
		},
		{
			name: "NotAdmin",
			task: map[string]any{
				"title":       "Some Task",
				"description": "",
				"column":      9,
				"subtasks":    []map[string]any{},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantErrMsg:     "Only board admins can create tasks.",
		},
		// todo: when testing for success, check order values too
	} {
		t.Run(c.name, func(t *testing.T) {
			task, err := json.Marshal(c.task)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(
				http.MethodPost, "", bytes.NewReader(task),
			)
			if err != nil {
				t.Fatal(err)
			}

			addBearerAuth(jwtBob123)(req)

			w := httptest.NewRecorder()

			sut.ServeHTTP(w, req)
			res := w.Result()

			if err = assert.Equal(
				c.wantStatusCode, res.StatusCode,
			); err != nil {
				t.Error(err)
			}

			resBody := taskAPI.ResBody{}
			if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
				t.Error(err)
			}
			if err := assert.Equal(c.wantErrMsg, resBody.Error); err != nil {
				t.Error(err)
			}
		})
	}
}
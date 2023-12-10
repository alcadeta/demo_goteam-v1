//go:build itest

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kxplxn/goteam/internal/api"
	columnAPI "github.com/kxplxn/goteam/internal/api/column"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/auth"
	boardTable "github.com/kxplxn/goteam/pkg/dbaccess/board"
	columnTable "github.com/kxplxn/goteam/pkg/dbaccess/column"
	userTable "github.com/kxplxn/goteam/pkg/dbaccess/user"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
)

// TestColumnHandler tests the http.Handler for the column API route and asserts
// that it behaves correctly during various execution paths.
func TestColumnHandler(t *testing.T) {
	// Create board API handler.
	log := pkgLog.New()
	sut := api.NewHandler(
		auth.NewJWTValidator(jwtKey),
		map[string]api.MethodHandler{
			http.MethodPatch: columnAPI.NewPATCHHandler(
				userTable.NewSelector(db),
				columnAPI.NewIDValidator(),
				columnTable.NewSelector(db),
				boardTable.NewSelector(db),
				columnTable.NewUpdater(db),
				log,
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
			{name: "HeaderInvalid", authFunc: addCookieAuth("asdfasldfkjasd")},
		} {
			t.Run(c.name, func(t *testing.T) {
				t.Run(http.MethodPatch, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodPatch, "/", nil)
					c.authFunc(req)
					w := httptest.NewRecorder()

					sut.ServeHTTP(w, req)
					res := w.Result()

					assert.Equal(t.Error,
						res.StatusCode, http.StatusUnauthorized,
					)

					assert.Equal(t.Error,
						res.Header.Values("WWW-Authenticate")[0], "Bearer",
					)
				})
			})
		}
	})
	t.Run("PATCH", func(t *testing.T) {
		for _, c := range []struct {
			name       string
			id         string
			reqBody    columnAPI.PATCHReq
			authFunc   func(*http.Request)
			statusCode int
			assertFunc func(*testing.T, *http.Response, string)
		}{
			{
				name:       "IDEmpty",
				id:         "",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr("Column ID cannot be empty."),
			},
			{
				name:       "IDNotInt",
				id:         "A",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr("Column ID must be an integer."),
			},
			{
				name:       "ColumnNotFound",
				id:         "1001",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusNotFound,
				assertFunc: assert.OnResErr("Column not found."),
			},
			{
				name:       "NotAdmin",
				id:         "5",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam1Member),
				statusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr("Only team admins can move tasks."),
			},
			{
				name:       "NoAccess",
				id:         "5",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam2Admin),
				statusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"You do not have access to this board.",
				),
			},
			{
				name:       "TaskNotFound",
				id:         "5",
				reqBody:    columnAPI.PATCHReq{{ID: 0, Order: 0}},
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusNotFound,
				assertFunc: assert.OnResErr("Task not found."),
			},
			{
				name:       "Success",
				id:         "6",
				reqBody:    columnAPI.PATCHReq{{ID: 5, Order: 2}},
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					var columnID, order int
					if err := db.QueryRow(
						`SELECT columnID, "order" FROM app.task WHERE id = $1`,
						5,
					).Scan(&columnID, &order); err != nil {
						t.Fatal(err)
					}
					assert.Equal(t.Error, columnID, 6)
					assert.Equal(t.Error, order, 2)
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				tasks, err := json.Marshal(c.reqBody)
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(
					http.MethodPatch, "/?id="+c.id, bytes.NewReader(tasks),
				)
				c.authFunc(req)
				w := httptest.NewRecorder()

				sut.ServeHTTP(w, req)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.statusCode)

				c.assertFunc(t, res, "")
			})
		}
	})
}
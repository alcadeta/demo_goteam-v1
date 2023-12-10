//go:build utest

package user

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/dbaccess"
)

func TestSelectorByTeamID(t *testing.T) {
	teamID := "21"
	sqlSelect := `SELECT username, isAdmin FROM app.\"user\" WHERE teamID = \$1`

	db, mock, teardown := dbaccess.SetUpDBTest(t)
	defer teardown()

	sut := NewSelectorByTeamID(db)

	t.Run("Error", func(t *testing.T) {
		wantErr := errors.New("error selecting user")

		mock.ExpectQuery(sqlSelect).WithArgs(teamID).WillReturnError(wantErr)

		_, err := sut.Select(teamID)

		assert.Equal(t.Error, wantErr, err)
	})

	t.Run("OK", func(t *testing.T) {
		wantRecs := []Record{
			{Username: "foo", IsAdmin: true},
			{Username: "bar", IsAdmin: false},
			{Username: "baz", IsAdmin: false},
		}
		rows := sqlmock.NewRows([]string{"username", "isAdmin"})
		for _, user := range wantRecs {
			rows.AddRow(user.Username, user.IsAdmin)
		}
		mock.ExpectQuery(sqlSelect).WithArgs(teamID).WillReturnRows(rows)

		recs, err := sut.Select(teamID)
		assert.Nil(t.Fatal, err)

		for i, user := range wantRecs {
			assert.Equal(t.Error, recs[i].Username, user.Username)
			assert.Equal(t.Error, recs[i].IsAdmin, user.IsAdmin)
		}
	})
}
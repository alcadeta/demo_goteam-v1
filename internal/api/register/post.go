package register

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/kxplxn/goteam/pkg/db"
	userTable "github.com/kxplxn/goteam/pkg/db/user"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/token"
)

// PostReq defines the body of POST register requests.
type PostReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PostResp defines the body of POST register responses.
type PostResp struct {
	Err            string         `json:"error,omitempty"`
	ValidationErrs ValidationErrs `json:"validationErrors,omitempty"`
}

// ValidationErrs defines the validation errors returned in POSTResp.
type ValidationErrs struct {
	Username []string `json:"username,omitempty"`
	Password []string `json:"password,omitempty"`
}

// Any checks whether there are any validation errors within the ValidationErrors.
func (e ValidationErrs) Any() bool {
	return len(e.Username) > 0 || len(e.Password) > 0
}

// PostHandler is a api.MethodHandler that can be used to handle POST register
// requests.
type PostHandler struct {
	reqValidator ReqValidator
	hasher       Hasher
	decodeInvite token.DecodeFunc[token.Invite]
	userInserter db.Inserter[userTable.User]
	encodeAuth   token.EncodeFunc[token.Auth]
	log          pkgLog.Errorer
}

// NewPostHandler creates and returns a new HandlerPost.
func NewPostHandler(
	userValidator ReqValidator,
	decodeInvite token.DecodeFunc[token.Invite],
	hasher Hasher,
	userInserter db.Inserter[userTable.User],
	encodeAuth token.EncodeFunc[token.Auth],
	log pkgLog.Errorer,
) PostHandler {
	return PostHandler{
		reqValidator: userValidator,
		hasher:       hasher,
		decodeInvite: decodeInvite,
		userInserter: userInserter,
		encodeAuth:   encodeAuth,
		log:          log,
	}
}

// ServeHTTP responds to requests made to the register route.
func (h PostHandler) Handle(w http.ResponseWriter, r *http.Request, _ string) {
	// decode request
	req := PostReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// validate request
	vdtErrs := h.reqValidator.Validate(req)
	if vdtErrs.Any() {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(
			PostResp{ValidationErrs: vdtErrs},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// determine teamID and isAdmin based on invite token.
	ck, err := r.Cookie(token.InviteName)
	var teamID string
	var isAdmin bool
	if err == http.ErrNoCookie {
		teamID = uuid.NewString()
		isAdmin = true
	} else if err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		invite, err := h.decodeInvite(ck.Value)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(
				PostResp{Err: "Invalid invite token."},
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err.Error())
			}
			return
		}
		teamID = invite.TeamID
		isAdmin = false
	}

	// hash password
	pwdHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// insert a new user into the user table
	if err = h.userInserter.Insert(r.Context(), userTable.NewUser(
		req.Username, pwdHash, isAdmin, teamID,
	)); err == db.ErrDupKey {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(
			PostResp{ValidationErrs: ValidationErrs{
				Username: []string{"Username is already taken."},
			}},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	} else if err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// generate an auth token
	exp := time.Now().Add(token.DefaultDuration).UTC()
	tkAuth, err := h.encodeAuth(exp, token.NewAuth(
		req.Username, isAdmin, teamID,
	))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(
			PostResp{
				Err: "You have been registered successfully but something " +
					"went wrong. Please log in using the credentials you " +
					"registered with.",
			},
		); err != nil {
			h.log.Error(err.Error())
		}
		return
	}

	// set auth cookie and respond OK
	http.SetCookie(w, &http.Cookie{
		Name:     token.AuthName,
		Value:    tkAuth,
		Expires:  exp,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})
	w.WriteHeader(http.StatusOK)
}
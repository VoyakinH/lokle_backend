package delivery

import (
	"net/http"
	"strconv"

	"github.com/VoyakinH/lokle_backend/internal/models"
	"github.com/VoyakinH/lokle_backend/internal/pkg/ctx_utils"
	"github.com/VoyakinH/lokle_backend/internal/pkg/ioutils"
	"github.com/VoyakinH/lokle_backend/internal/pkg/middleware"
	"github.com/VoyakinH/lokle_backend/internal/pkg/tools"
	"github.com/VoyakinH/lokle_backend/internal/user/usecase"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type UserDelivery struct {
	userUseCase usecase.IUserUsecase
	logger      logrus.Logger
}

func SetUserRouting(router *mux.Router,
	uu usecase.IUserUsecase,
	auth middleware.AuthMiddleware,
	roleMw middleware.RoleMiddleware,
	logger logrus.Logger) {
	userDelivery := &UserDelivery{
		userUseCase: uu,
		logger:      logger,
	}

	userAPI := router.PathPrefix("/api/v1/user/").Subrouter()
	userAPI.Use(middleware.WithJSON)

	userAPI.HandleFunc("/auth", userDelivery.CreateUserSession).Methods(http.MethodPost)
	userAPI.HandleFunc("/auth", userDelivery.DeleteUserSession).Methods(http.MethodDelete)
	userAPI.Handle("/auth", auth.WithAuth(http.HandlerFunc(userDelivery.CheckUserSession))).Methods(http.MethodGet)

	userAPI.HandleFunc("/parent", userDelivery.SignupParent).Methods(http.MethodPost)
	userAPI.Handle("/parent", auth.WithAuth(http.HandlerFunc(userDelivery.GetParent))).Methods(http.MethodGet)
	userAPI.Handle("/parent/children", auth.WithAuth(roleMw.CheckParent(http.HandlerFunc(userDelivery.GetParentChildren)))).Methods(http.MethodGet)

	userAPI.HandleFunc("/email", userDelivery.EmailVerification).Methods(http.MethodGet)
	userAPI.HandleFunc("/email", userDelivery.RepeatEmailVerification).Methods(http.MethodPost)

	userAPI.Handle("/admin/manager", auth.WithAuth(roleMw.CheckAdmin(http.HandlerFunc(userDelivery.SignupManager)))).Methods(http.MethodPost)
	userAPI.Handle("/admin/managers", auth.WithAuth(roleMw.CheckAdmin(http.HandlerFunc(userDelivery.GetManagers)))).Methods(http.MethodGet)

	userAPI.Handle("/manager/child", auth.WithAuth(roleMw.CheckManager(http.HandlerFunc(userDelivery.GetChildByUID)))).Methods(http.MethodGet)
	userAPI.Handle("/manager/parent", auth.WithAuth(roleMw.CheckManager(http.HandlerFunc(userDelivery.GetParentByUID)))).Methods(http.MethodGet)
}

const expCookieTime = 1382400

func (ud *UserDelivery) CreateUserSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var credentials models.Credentials
	err := ioutils.ReadJSON(r, &credentials)
	if err != nil || credentials.Email == "" || credentials.Password == "" {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusBadRequest, err)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	user, status, err := ud.userUseCase.CheckUser(ctx, credentials)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	if !user.EmailVerified && user.Role == models.ParentRole {
		ud.logger.Errorf("%s user email not verified [status=%d]", r.URL, http.StatusUnauthorized)
		ioutils.SendDefaultError(w, http.StatusUnauthorized)
		return
	}

	sessionID, status, err := ud.userUseCase.CreateSession(ctx, credentials.Email, expCookieTime)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	cookie := &http.Cookie{
		Name:   "session-id",
		Value:  sessionID,
		MaxAge: expCookieTime,
		Path:   "/api/v1",
	}

	http.SetCookie(w, cookie)
	ioutils.Send(w, status, tools.UserToUserRes(user))
}

func (ud *UserDelivery) DeleteUserSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookieToken, err := r.Cookie("session-id")
	if err != nil {
		ud.logger.Warnf("%s cookie not found with [status=%d] [error=%s]", r.URL, http.StatusOK, err)
		ioutils.SendDefaultError(w, http.StatusOK)
		return
	}

	status, err := ud.userUseCase.DeleteSession(ctx, cookieToken.Value)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	cookie := &http.Cookie{
		Name:   "session-id",
		Value:  "",
		MaxAge: -1,
		Path:   "/api/v1",
	}

	http.SetCookie(w, cookie)
}

func (ud *UserDelivery) CheckUserSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx_utils.GetUser(ctx)
	if user == nil {
		ud.logger.Errorf("%s failed get ctx user with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	cookieToken, err := r.Cookie("session-id")
	if err != nil {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusUnauthorized, err)
		ioutils.SendDefaultError(w, http.StatusUnauthorized)
		return
	}

	status, err := ud.userUseCase.ProlongSession(ctx, cookieToken.Value, expCookieTime)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	cookie := &http.Cookie{
		Name:   "session-id",
		Value:  cookieToken.Value,
		MaxAge: expCookieTime,
		Path:   "/api/v1",
	}

	http.SetCookie(w, cookie)
	ioutils.Send(w, status, tools.UserToUserRes(*user))
}

func (ud *UserDelivery) SignupParent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var parent models.User
	err := ioutils.ReadJSON(r, &parent)
	if err != nil || parent.Email == "" || parent.Password == "" {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusBadRequest, err)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	createdParent, status, err := ud.userUseCase.CreateParentUser(ctx, parent)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, tools.UserToUserRes(createdParent))
}

func (ud *UserDelivery) EmailVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	if token == "" {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusBadRequest, "empty token")
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	status, err := ud.userUseCase.VerifyEmail(ctx, token)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.SendWithoutBody(w, status)
}

func (ud *UserDelivery) RepeatEmailVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var credentials models.Credentials
	err := ioutils.ReadJSON(r, &credentials)
	if err != nil || credentials.Email == "" || credentials.Password == "" {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusBadRequest, err)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	status, err := ud.userUseCase.RepeatEmailVerification(ctx, credentials)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.SendWithoutBody(w, status)
}

func (ud *UserDelivery) GetParent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx_utils.GetUser(ctx)
	if user == nil {
		ud.logger.Errorf("%s failed get ctx user with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	parent, status, err := ud.userUseCase.GetParentByUID(ctx, user.ID)
	// if parent for created user not found and email verified
	// we try to create parent without email verification
	if status == http.StatusNotFound {
		ud.logger.Errorf("%s parent not found for user <%d:%s> [status=%d]", r.URL, user.ID, user.Email, http.StatusNotFound)
		parent, status, err = ud.userUseCase.CreateParent(ctx, user.ID)
		if err != nil || status != http.StatusOK {
			ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
			ioutils.SendDefaultError(w, status)
			return
		}
	}
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, tools.ParentToParentRes(parent))
}

func (ud *UserDelivery) SignupManager(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx_utils.GetUser(ctx)
	if user == nil {
		ud.logger.Errorf("%s failed get ctx user with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	var manager models.User
	err := ioutils.ReadJSON(r, &manager)
	if err != nil || manager.Email == "" || manager.Password == "" {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, http.StatusBadRequest, err)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	createdManager, status, err := ud.userUseCase.CreateManager(ctx, manager)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, tools.UserToUserRes(createdManager))
}

func (ud *UserDelivery) GetParentChildren(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parent := ctx_utils.GetParent(ctx)
	if parent == nil {
		ud.logger.Errorf("%s failed get ctx parent with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	respList, status, err := ud.userUseCase.GetParentChildren(ctx, parent.ID)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, respList)
}

func (ud *UserDelivery) GetChildByUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	manager := ctx_utils.GetUser(ctx)
	if manager == nil {
		ud.logger.Errorf("%s failed get ctx parent with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	childIDString := r.URL.Query().Get("child")
	if childIDString == "" {
		ud.logger.Errorf("%s empty query [status=%d]", r.URL, http.StatusBadRequest)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}
	childID, err := strconv.ParseUint(childIDString, 10, 64)
	if err != nil {
		ud.logger.Errorf("%s invalid child id parameter [status=%d]", r.URL, http.StatusBadRequest)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	child, status, err := ud.userUseCase.GetChildByUID(ctx, childID)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, child)
}

func (ud *UserDelivery) GetParentByUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	manager := ctx_utils.GetUser(ctx)
	if manager == nil {
		ud.logger.Errorf("%s failed get ctx parent with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	parentIDString := r.URL.Query().Get("parent")
	if parentIDString == "" {
		ud.logger.Errorf("%s empty query [status=%d]", r.URL, http.StatusBadRequest)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}
	parentID, err := strconv.ParseUint(parentIDString, 10, 64)
	if err != nil {
		ud.logger.Errorf("%s invalid child id parameter [status=%d]", r.URL, http.StatusBadRequest)
		ioutils.SendDefaultError(w, http.StatusBadRequest)
		return
	}

	parent, status, err := ud.userUseCase.GetParentByUID(ctx, parentID)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, parent)
}

func (ud *UserDelivery) GetManagers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parent := ctx_utils.GetUser(ctx)
	if parent == nil {
		ud.logger.Errorf("%s failed get ctx user with [status=%d]", r.URL, http.StatusForbidden)
		ioutils.SendDefaultError(w, http.StatusForbidden)
		return
	}

	respList, status, err := ud.userUseCase.GetManagers(ctx)
	if err != nil || status != http.StatusOK {
		ud.logger.Errorf("%s failed with [status=%d] [error=%s]", r.URL, status, err)
		ioutils.SendDefaultError(w, status)
		return
	}

	ioutils.Send(w, status, tools.UsersToUserResList(respList))
}

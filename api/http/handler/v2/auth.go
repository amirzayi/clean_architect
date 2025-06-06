package v2

import (
	"net/http"

	"github.com/amirzayi/clean_architect/api/http/handler/v2/dto"
	"github.com/amirzayi/clean_architect/internal/domain"
	"github.com/amirzayi/clean_architect/internal/service"
	"github.com/amirzayi/clean_architect/pkg/jsonutil"
	"github.com/amirzayi/rahjoo"
)

type authRouter struct {
	authService service.Auth
}

func AuthRoutes(auth service.Auth) rahjoo.Route {
	router := &authRouter{authService: auth}

	return rahjoo.NewGroupRoute("/v2/auth", rahjoo.Route{
		"/register": {
			http.MethodPost: rahjoo.NewHandler(router.register),
		},
		"/login": {
			http.MethodPost: rahjoo.NewHandler(router.login),
		},
	}) // todo: add throttle middleware
}

func (a *authRouter) register(w http.ResponseWriter, r *http.Request) {
	in, err := jsonutil.DecodeAndValidate[dto.RegisterRequest](r)
	if err != nil {
		jsonutil.EncodeError(w, err)
		return
	}

	err = a.authService.Register(r.Context(), domain.Auth{
		Email:       in.Email,
		PhoneNumber: in.PhoneNumber,
		Password:    in.Password,
	})
	if err != nil {
		jsonutil.EncodeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *authRouter) login(w http.ResponseWriter, r *http.Request) {
	//todo: validate request
	in, err := jsonutil.DecodeAndValidate[domain.Auth](r)
	if err != nil {
		jsonutil.EncodeError(w, err)
		return
	}

	token, err := a.authService.Login(r.Context(), in)
	if err != nil {
		jsonutil.EncodeError(w, err)
		return
	}
	_ = jsonutil.Encode(w, http.StatusOK, dto.LoginResponse{Token: token})
}

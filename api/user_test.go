package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/hafizxd/simple_bank/db/mock"
	db "github.com/hafizxd/simple_bank/db/sqlc"
	"github.com/hafizxd/simple_bank/util"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}

	e.arg.HashedPassword = arg.HashedPassword
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func TestCreateUserApi(t *testing.T) {
	user, password := createRandomUser(t)

	testCases := []struct {
		name          string
		requestBody   gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			requestBody: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
					MaxTimes(1).
					Return(user, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchBodyUser(t, recorder.Body, user)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)

		tc.buildStubs(store)

		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()

		url := fmt.Sprintf("/users")

		data, err := json.Marshal(tc.requestBody)
		require.NoError(t, err)

		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		require.NoError(t, err)

		server.router.ServeHTTP(recorder, request)

		tc.checkResponse(t, recorder)
	}
}

func createRandomUser(t *testing.T) (db.User, string) {
	password := util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	return db.User{
		Username:          util.RandomOwner(),
		HashedPassword:    hashedPassword,
		FullName:          util.RandomOwner(),
		Email:             util.RandomEmail(),
		CreatedAt:         time.Time{},
		PasswordChangedAt: time.Time{},
	}, password
}

func requireMatchBodyUser(t *testing.T, body *bytes.Buffer, user db.User) {
	response, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotUser db.User
	err = json.Unmarshal(response, &gotUser)
	require.NoError(t, err)
	require.NotEmpty(t, gotUser)
	require.Equal(t, user.Username, gotUser.Username)
}

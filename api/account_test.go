package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	mockdb "github.com/hafizxd/simple_bank/db/mock"
	db "github.com/hafizxd/simple_bank/db/sqlc"
	"github.com/hafizxd/simple_bank/token"
	"github.com/hafizxd/simple_bank/util"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateAccountApi(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)

	createRequestBody := createAccountRequest{
		Currency: account.Currency,
	}

	arg := db.CreateAccountParams{
		Owner:    user.Username,
		Currency: createRequestBody.Currency,
		Balance:  0,
	}

	testCases := []struct {
		name          string
		requestBody   createAccountRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:        "OK",
			requestBody: createRequestBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchBodyAccount(t, recorder.Body, account)
			},
		},
		{
			name:        "InternalServerError",
			requestBody: createRequestBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "BadRequest",
			requestBody: createAccountRequest{
				Currency: "WRONG",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

		url := fmt.Sprintf("/accounts")

		var reqBody bytes.Buffer
		err := json.NewEncoder(&reqBody).Encode(tc.requestBody)
		require.NoError(t, err)

		request, err := http.NewRequest(http.MethodPost, url, &reqBody)
		require.NoError(t, err)

		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
		server.router.ServeHTTP(recorder, request)
		tc.checkResponse(t, recorder)
	}
}

func TestGetAccountApi(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)

	testCases := []struct {
		name          string
		accountId     int64
		setupAuth     func(t *testing.T, r *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountId: account.ID,
			setupAuth: func(t *testing.T, r *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, r, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchBodyAccount(t, recorder.Body, account)
			},
		},
		{
			name:      "NotFound",
			accountId: account.ID,
			setupAuth: func(t *testing.T, r *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, r, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalServerError",
			accountId: account.ID,
			setupAuth: func(t *testing.T, r *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, r, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountId: 0,
			setupAuth: func(t *testing.T, r *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, r, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

		url := fmt.Sprintf("/accounts/%d", tc.accountId)
		request, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		tc.setupAuth(t, request, server.tokenMaker)
		server.router.ServeHTTP(recorder, request)
		tc.checkResponse(t, recorder)
	}
}

func TestListAccountApi(t *testing.T) {
	listAccountRequest := ListAccountRequest{
		PageID:   int32(util.RandomInt(1, 10)),
		PageSize: int32(util.RandomInt(5, 10)),
	}

	user, _ := createRandomUser(t)

	arg := db.ListAccountParams{
		Owner:  user.Username,
		Offset: (listAccountRequest.PageID * listAccountRequest.PageSize) - listAccountRequest.PageSize,
		Limit:  listAccountRequest.PageSize,
	}

	var accounts []db.Account
	n := 5
	for i := 0; i < n; i++ {
		account := createRandomAccount(user.Username)
		accounts = append(accounts, account)
	}

	testCases := []struct {
		name          string
		pageId        int32
		pageSize      int32
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "OK",
			pageId:   listAccountRequest.PageID,
			pageSize: listAccountRequest.PageSize,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchBodyAccountList(t, recorder.Body, accounts)
			},
		},
		{
			name:     "InternalServerError",
			pageId:   listAccountRequest.PageID,
			pageSize: listAccountRequest.PageSize,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return([]db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:     "BadRequest",
			pageId:   0,
			pageSize: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)

		server := newTestServer(t, store)
		recorder := httptest.NewRecorder()

		tc.buildStubs(store)

		url := fmt.Sprintf("/accounts?page_id=%d&page_size=%d", tc.pageId, tc.pageSize)
		request, err := http.NewRequest(http.MethodGet, url, nil)
		require.NoError(t, err)

		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
		server.router.ServeHTTP(recorder, request)
		tc.checkResponse(t, recorder)
	}
}

func TestUpdateAccountApi(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)

	updateRequestBody := UpdateAccountRequest{
		Balance: util.RandomMoney(),
	}

	arg := db.UpdateAccountParams{
		ID:      account.ID,
		Balance: updateRequestBody.Balance,
	}

	testCases := []struct {
		name          string
		accountId     int64
		requestBody   UpdateAccountRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:        "OK",
			accountId:   account.ID,
			requestBody: updateRequestBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchBodyAccount(t, recorder.Body, account)
			},
		},
		{
			name:        "InternalServerError",
			accountId:   account.ID,
			requestBody: updateRequestBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Eq(arg)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:        "BadRequestUrl",
			accountId:   0,
			requestBody: updateRequestBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "BadRequestBody",
			accountId: account.ID,
			requestBody: UpdateAccountRequest{
				Balance: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

		url := fmt.Sprintf("/accounts/%d", tc.accountId)

		var reqBody bytes.Buffer
		err := json.NewEncoder(&reqBody).Encode(tc.requestBody)
		require.NoError(t, err)

		request, err := http.NewRequest(http.MethodPut, url, &reqBody)
		require.NoError(t, err)

		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
		server.router.ServeHTTP(recorder, request)
		tc.checkResponse(t, recorder)
	}
}

func TestDeleteAccountApi(t *testing.T) {
	user, _ := createRandomUser(t)
	account := createRandomAccount(user.Username)

	testCases := []struct {
		name          string
		accountId     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(account, nil)

				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "InternalServerErrorGetAccount",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "InternalServerErrorDeleteAccount",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(account, nil)

				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).
					MaxTimes(1).
					Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountId: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)

				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Any()).
					MaxTimes(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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

		url := fmt.Sprintf("/accounts/%d", tc.accountId)

		request, err := http.NewRequest(http.MethodDelete, url, nil)
		require.NoError(t, err)

		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
		server.router.ServeHTTP(recorder, request)
		tc.checkResponse(t, recorder)
	}
}

func createRandomAccount(owner string) db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Time{},
	}
}

func requireMatchBodyAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
	response, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccount db.Account
	err = json.Unmarshal(response, &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account, gotAccount)
}

func requireMatchBodyAccountList(t *testing.T, body *bytes.Buffer, accounts []db.Account) {
	response, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAccounts []db.Account
	err = json.Unmarshal(response, &gotAccounts)
	require.NoError(t, err)
	require.ElementsMatch(t, gotAccounts, accounts)
}

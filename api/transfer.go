package api

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	db "github.com/hafizxd/micro-bank/db/sqlc"
	"github.com/hafizxd/micro-bank/token"
	"net/http"
)

type transferRequest struct {
	FromAccountId int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountId   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	valid, account1 := server.validAccount(ctx, req.FromAccountId, req.Currency)
	if !valid {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account1.Owner != authPayload.Username {
		err := errors.New("from account doesn't belong to authenticated users")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	valid, _ = server.validAccount(ctx, req.ToAccountId, req.Currency)
	if !valid {
		return
	}

	if account1.Balance < req.Amount {
		err := fmt.Errorf("balance is not enough: %d", account1.Balance)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	arg := db.TransferTxParams{
		FromAccountId: req.FromAccountId,
		ToAccountId:   req.ToAccountId,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (server *Server) validAccount(ctx *gin.Context, accountId int64, currency string) (bool, db.Account) {
	account, err := server.store.GetAccount(ctx, accountId)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return false, db.Account{}
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return false, db.Account{}
	}

	if account.Currency != currency {
		err = fmt.Errorf("account %d currency invalid: %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false, db.Account{}
	}

	return true, account
}

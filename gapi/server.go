package gapi

import (
	"fmt"
	db "github.com/hafizxd/micro-bank/db/sqlc"
	"github.com/hafizxd/micro-bank/pb"
	"github.com/hafizxd/micro-bank/token"
	"github.com/hafizxd/micro-bank/util"
	"github.com/hafizxd/micro-bank/worker"
)

type Server struct {
	pb.UnimplementedMicroBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token marker: %v", err)
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}

package main

import (
	"bank/api"
	db "bank/db/sqlc"
	"bank/gapi"
	"bank/pb"
	"bank/util"
	"database/sql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"

	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("can not load config: ", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("can not connect to the db")
	}
	store := db.NewStore(conn)

	// runGRPCServer
	runGRPCServer(config, store)
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("can not create server: ", err)
	}
	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot start server: ", err)
	}
}

func runGRPCServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("can not create server: ", err)
	}

	gRPCServer := grpc.NewServer()
	pb.RegisterBankServer(gRPCServer, server)
	reflection.Register(gRPCServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("can not create listener ", err)
	}

	log.Printf("starting gRPC server on %s", listener.Addr().String())
	err = gRPCServer.Serve(listener)
	if err != nil {
		log.Fatal("can not start gRPC server ", err)
	}
}

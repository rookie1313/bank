package main

import (
	"bank/api"
	db "bank/db/sqlc"
	_ "bank/doc/statik"
	"bank/gapi"
	"bank/pb"
	"bank/util"
	"context"
	"database/sql"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rakyll/statik/fs"
	"github.com/skratchdot/open-golang/open"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"log"
	"net"
	"net/http"

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

	go runGatewayServer(config, store)
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

func runGatewayServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("can not create server: ", err)
	}

	grpcMux := runtime.NewServeMux(
		//Using proto names in JSON
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("can not register gateway server: ", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	//swagger
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal("can not create statik fs: ", err)
	}
	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(statikFS))
	mux.Handle("/swagger/", swaggerHandler)
	//open swagger when start
	go func() {
		addr := "http://localhost:8080/swagger"
		err := open.Run(addr)
		if err != nil {
			log.Fatal("can not open swagger api page ", err)
		}
	}()
	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("can not create listener ", err)
	}

	log.Printf("starting http gateway server on %s", listener.Addr().String())
	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal("can not start http gateway server ", err)
	}

}

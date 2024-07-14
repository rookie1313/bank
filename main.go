package main

import (
	"bank/api"
	db "bank/db/sqlc"
	_ "bank/doc/statik"
	"bank/gapi"
	"bank/pb"
	"bank/util"
	"bank/worker"
	"context"
	"database/sql"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("can not load config")
	}
	if config.Environment == "development" {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			FormatMessage: func(i any) string {
				return fmt.Sprintf("*** %s ***", i)
			},
		}

		log.Logger = log.Output(output).With().Caller().Logger()
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("can not connect to the db")
	}
	store := db.NewStore(conn)

	//redisOpt := asynq.RedisClientOpt{
	//	Addr: config.RedisAddress,
	//}
	//taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	//go runTaskProcessor(redisOpt, store)
	//
	//go runGatewayServer(config, store, taskDistributor)
	//runGRPCServer(config, store, taskDistributor)
	runGinServer(config, store)
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("can not create server")
	}
	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}

func runTaskProcessor(opt asynq.RedisClientOpt, store db.Store) {
	task := worker.NewRedisTaskProcessor(opt, store)
	log.Info().Msg("start task processor")
	err := task.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start task processor")
	}
}

func runGRPCServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) {
	server, err := gapi.NewServer(config, store, taskDistributor)
	if err != nil {
		log.Fatal().Err(err).Msg("can not create server")
	}

	grpcLogger := grpc.UnaryInterceptor(gapi.GRPCLogger)
	gRPCServer := grpc.NewServer(grpcLogger)
	pb.RegisterBankServer(gRPCServer, server)
	reflection.Register(gRPCServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("can not create listener")
	}

	log.Info().Msgf("starting gRPC server on %s", listener.Addr().String())
	err = gRPCServer.Serve(listener)
	if err != nil {
		log.Fatal().Err(err).Msg("can not start gRPC server")
	}
}

func runGatewayServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) {
	server, err := gapi.NewServer(config, store, taskDistributor)
	if err != nil {
		log.Fatal().Err(err).Msg("can not create server")
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
		log.Fatal().Err(err).Msg("can not register gateway server")
	}
	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	//swagger
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal().Err(err).Msg("can not create statik fs")
	}
	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(statikFS))
	mux.Handle("/swagger/", swaggerHandler)
	//open swagger when start
	//go func() {
	//	addr := "http://localhost:8080/swagger"
	//	err = open.Run(addr)
	//	if err != nil {
	//		log.Fatal().Err(err).Msg("can not open swagger api page")
	//	}
	//}()
	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("can not create listener")
	}

	log.Info().Msgf("starting http gateway server on %s", listener.Addr().String())

	handler := gapi.HttpLogger(mux)
	err = http.Serve(listener, handler)
	if err != nil {
		log.Fatal().Err(err).Msg("can not start http gateway server")
	}

}

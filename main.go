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
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hibiken/asynq"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

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

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("can not connect to the db")
	}
	store := db.NewStore(conn)

	redisOpt := asynq.RedisClientOpt{
		Addr: config.RedisAddress,
	}
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)

	waitGroup, ctx := errgroup.WithContext(ctx)

	runTaskProcessor(waitGroup, ctx, redisOpt, store)
	runGatewayServer(waitGroup, ctx, config, store, taskDistributor)
	runGRPCServer(waitGroup, ctx, config, store, taskDistributor)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("wait group err")
	}
}

func runTaskProcessor(
	wg *errgroup.Group,
	ctx context.Context,
	opt asynq.RedisClientOpt,
	store db.Store) {
	task := worker.NewRedisTaskProcessor(opt, store)
	log.Info().Msg("start task processor")
	err := task.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start task processor")
	}

	wg.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("graceful shutdown of task processor")

		task.ShutDown()
		log.Info().Msg("task processor is stopped")
		return nil
	})
}

func runGRPCServer(
	wg *errgroup.Group,
	ctx context.Context,
	config util.Config,
	store db.Store,
	taskDistributor worker.TaskDistributor) {
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

	wg.Go(func() error {
		log.Info().Msgf("starting gRPC server on %s", listener.Addr().String())
		err = gRPCServer.Serve(listener)
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			log.Error().Err(err).Msg("gRPC server failed to serve")
			return err
		}

		return nil
	})

	//signal
	wg.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("gracefully shutting down gRPC server")
		gRPCServer.GracefulStop()
		log.Info().Msg("gracefully stopped gRPC server")
		return nil
	})
}

func runGatewayServer(
	wg *errgroup.Group,
	ctx context.Context,
	config util.Config,
	store db.Store,
	taskDistributor worker.TaskDistributor) {
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

	httpServer := &http.Server{
		Handler: gapi.HttpLogger(mux),
		Addr:    config.HTTPServerAddress,
	}

	wg.Go(func() error {
		log.Info().Msgf("starting http gateway server on %s", httpServer.Addr)

		err = httpServer.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			log.Error().Err(err).Msg("http gateway server failed to serve")
			return err
		}

		return nil
	})

	//signal
	wg.Go(func() error {
		<-ctx.Done()
		log.Info().Msg("gracefully shutting down gRPC gateway server")

		err = httpServer.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("can not shutdown gRPC gateway server")
			return err
		}

		return nil
	})
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

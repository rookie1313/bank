package gapi

import (
	db "bank/db/sqlc"
	"bank/pb"
	"bank/util"
	"bank/val"
	"bank/worker"
	"context"
	"errors"
	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	violations := validateCreateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password %s ", err.Error())
	}

	args := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.GetUsername(),
			HashedPassword: hashedPassword,
			FullName:       req.GetFullName(),
			Email:          req.GetEmail(),
		},
		AfterCreate: func(user db.User) error {
			opts := []asynq.Option{
				asynq.MaxRetry(10),
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue("critical"),
			}

			return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, &worker.PayloadSendVerifyEmail{
				Username: user.Username},
				opts...,
			)
		},
	}

	userTransactionResult, err := server.store.CreateUserTx(ctx, args)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			switch pgErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "username already exists %s ", err.Error())
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user %s ", err.Error())
	}

	res := &pb.CreateUserResponse{
		User: ConvertUser(userTransactionResult.User),
	}

	return res, nil
}

func validateCreateUserRequest(req *pb.CreateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	err := val.ValidateUsername(req.GetUsername())
	if err != nil {
		violations = append(violations, fieldViolation("username", err))
	}
	err = val.ValidatePassword(req.GetPassword())
	if err != nil {
		violations = append(violations, fieldViolation("password", err))
	}
	err = val.ValidateFullName(req.GetFullName())
	if err != nil {
		violations = append(violations, fieldViolation("full_name", err))
	}
	err = val.ValidateEmail(req.GetEmail())
	if err != nil {
		violations = append(violations, fieldViolation("email", err))
	}

	return violations
}

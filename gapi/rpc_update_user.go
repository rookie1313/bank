package gapi

import (
	db "bank/db/sqlc"
	"bank/pb"
	"bank/util"
	"bank/val"
	"context"
	"database/sql"
	"errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	payload, err := server.authorizeUser(ctx)
	if err != nil {
		return nil, unauthenticatedError(err)
	}

	violations := validateUpdateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	if payload.Username != req.GetUsername() {
		return nil, status.Errorf(codes.PermissionDenied, "cannot update other user's info")
	}

	args := db.UpdateUserParams{
		Username: req.GetUsername(),
		FullName: sql.NullString{
			String: req.GetFullName(),
			Valid:  req.GetFullName() != "",
		},
		Email: sql.NullString{
			String: req.GetEmail(),
			Valid:  req.GetEmail() != "",
		},
	}
	// fill hashed password if password is not empty
	if req.Password != nil {
		hashedPassword, err := util.HashPassword(req.GetPassword())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash password %s ", err.Error())
		}
		args.HashedPassword = sql.NullString{
			String: hashedPassword,
			Valid:  true,
		}
		args.PasswordChangedAt = sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		}
	}

	user, err := server.store.UpdateUser(ctx, args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found %s ", err.Error())
		}

		return nil, status.Errorf(codes.Internal, "failed to update user %s ", err.Error())
	}
	res := &pb.UpdateUserResponse{
		User: ConvertUser(user),
	}

	return res, nil
}

func validateUpdateUserRequest(req *pb.UpdateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	err := val.ValidateUsername(req.GetUsername())
	if err != nil {
		violations = append(violations, fieldViolation("username", err))
	}
	if req.Password != nil {
		err = val.ValidatePassword(req.GetPassword())
		if err != nil {
			violations = append(violations, fieldViolation("password", err))
		}
	}
	if req.FullName != nil {
		err = val.ValidateFullName(req.GetFullName())
		if err != nil {
			violations = append(violations, fieldViolation("full_name", err))
		}
	}
	if req.Email != nil {
		err = val.ValidateEmail(req.GetEmail())
		if err != nil {
			violations = append(violations, fieldViolation("email", err))
		}
	}

	return violations
}

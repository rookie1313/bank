package gapi

import (
	db "bank/db/sqlc"
	"bank/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ConvertUser(user db.User) *pb.User {
	return &pb.User{
		Username:          user.Username,
		FullName:          user.FullName,
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		Email:             user.Email,
		CreatedAt:         timestamppb.New(user.CreatedAt),
	}
}

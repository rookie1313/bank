package api

import (
	db "bank/db/sqlc"
	"bank/util"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"net/http"
	"time"
)

type createUserRequest struct {
	UserName string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}
type createUserResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	Email             string    `json:"email"`
	CreatedAt         time.Time `json:"created_at"`
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	args := db.CreateUserParams{
		Username:       req.UserName,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}
	user, err := server.store.CreateUser(ctx, args)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	res := createUserResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		PasswordChangedAt: user.PasswordChangedAt,
		Email:             user.Email,
		CreatedAt:         user.CreatedAt,
	}
	ctx.JSON(http.StatusOK, res)
}

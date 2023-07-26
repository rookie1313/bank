package api

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type reNewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type reNewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiredAt time.Time `json:"access_token_expired_at"`
}

func (server *Server) reNewAccessToken(ctx *gin.Context) {

	var req reNewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errResponse(err))
		return
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}
	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}
	if session.IsBlocked {
		err = fmt.Errorf("session is blocked")
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}
	if session.Username != refreshPayload.Username {
		err = fmt.Errorf("incorrect session user")
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}
	if session.RefreshToken != req.RefreshToken {
		err = fmt.Errorf("incorrect session token")
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}
	if time.Now().After(session.ExpiredAt) {
		err = fmt.Errorf("expired session")
		ctx.JSON(http.StatusUnauthorized, errResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(refreshPayload.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errResponse(err))
		return
	}

	res := reNewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiredAt: accessPayload.ExpiredAt,
	}
	ctx.JSON(http.StatusOK, res)
}

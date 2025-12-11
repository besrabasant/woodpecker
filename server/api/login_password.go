// Copyright 2025 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	"go.woodpecker-ci.org/woodpecker/v3/server/auth"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
	"go.woodpecker-ci.org/woodpecker/v3/shared/httputil"
	"go.woodpecker-ci.org/woodpecker/v3/shared/token"
)

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type passwordLoginResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Token     string `json:"token,omitempty"`
	ExpiresIn int64  `json:"expires_in,omitempty"`
}

func PostLoginWithPassword(c *gin.Context) {
	var req passwordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, passwordLoginResponse{Message: "invalid payload"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, passwordLoginResponse{Message: "username and password required"})
		return
	}

	_store := store.FromContext(c)
	authUser, err := _store.AuthUserFindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, types.RecordNotExist) {
			c.JSON(http.StatusUnauthorized, passwordLoginResponse{Message: "invalid credentials"})
			return
		}
		log.Error().Err(err).Msg("failed to load auth user")
		c.JSON(http.StatusInternalServerError, passwordLoginResponse{Message: "internal error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, passwordLoginResponse{Message: "invalid credentials"})
		return
	}

	user, err := _store.GetUser(authUser.UserID)
	if err != nil {
		if errors.Is(err, types.RecordNotExist) {
			log.Warn().Int64("user_id", authUser.UserID).Msg("auth user has no matching user entry")
			c.JSON(http.StatusUnauthorized, passwordLoginResponse{Message: "invalid credentials"})
			return
		}
		log.Error().Err(err).Msg("failed to load user for auth user login")
		c.JSON(http.StatusInternalServerError, passwordLoginResponse{Message: "internal error"})
		return
	}

	exp := time.Now().Add(server.Config.Server.SessionExpires).Unix()
	sessionToken := token.New(token.SessToken)
	sessionToken.Set("user-id", strconv.FormatInt(user.ID, 10))
	signed, err := sessionToken.SignExpires(user.Hash, exp)
	if err != nil {
		log.Error().Err(err).Msg("cannot sign session token for auth user")
		c.JSON(http.StatusInternalServerError, passwordLoginResponse{Message: "internal error"})
		return
	}


	httputil.SetCookie(c.Writer, c.Request, "user_sess", signed)

	jwtToken, err := auth.IssueJWT(user)
	if err != nil {
		log.Warn().Err(err).Msg("cannot issue jwt token for user, falling back to session token")
		jwtToken = signed
	}

	resp := passwordLoginResponse{
		Success:   true,
		Token:     jwtToken,
		ExpiresIn: int64(server.Config.Server.SessionExpires.Seconds()),
	}
	c.JSON(http.StatusOK, resp)
}

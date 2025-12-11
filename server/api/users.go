// Copyright 2018 Drone.IO Inc.
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
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/tink/go/subtle/random"
	"golang.org/x/crypto/bcrypt"

	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
)

const defaultForgeID = 1

// GetUsers
//
//	@Summary		List users
//	@Description	Returns all registered, active users in the system. Requires admin rights.
//	@Router			/users [get]
//	@Produce		json
//	@Success		200	{array}	User
//	@Tags			Users
//	@Param			Authorization	header	string	true	"Insert your personal access token"				default(Bearer <personal access token>)
//	@Param			page			query	int		false	"for response pagination, page offset number"	default(1)
//	@Param			perPage			query	int		false	"for response pagination, max items per page"	default(50)
func GetUsers(c *gin.Context) {
	users, err := store.FromContext(c).GetUserList(session.Pagination(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting user list. %s", err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetUser
//
//	@Summary		Get a user
//	@Description	Returns a user with the specified login name. Requires admin rights.
//	@Router			/users/{login} [get]
//	@Produce		json
//	@Success		200	{object}	User
//	@Tags			Users
//	@Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param			login			path	string	true	"the user's login name"
//	@Param			forge_id		query	string	true	"specify forge (else default will be used)"
//	@Param			forge_remote_id	query	string	false	"specify user id at forge (else fallback to login)"
func GetUser(c *gin.Context) {
	forgeID, err := strconv.ParseInt(c.DefaultQuery("forge_id", fmt.Sprint(defaultForgeID)), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	forgeRemoteID := model.ForgeRemoteID(c.Query("forge_remote_id"))

	var user *model.User

	if forgeRemoteID.IsValid() {
		user, err = store.FromContext(c).GetUserByRemoteID(forgeID, forgeRemoteID)
	} else {
		user, err = store.FromContext(c).GetUserByLogin(forgeID, c.Param("login"))
	}
	if err != nil {
		handleDBError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// PatchUser
//
//	@Summary		Update a user
//	@Description	Changes the data of an existing user. Requires admin rights.
//	@Router			/users/{login} [patch]
//	@Produce		json
//	@Accept			json
//	@Success		200	{object}	User
//	@Tags			Users
//	@Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param			login			path	string	true	"the user's login name"
//	@Param			user			body	User	true	"the user's data"
func PatchUser(c *gin.Context) {
	_store := store.FromContext(c)

	in := &model.User{}
	err := c.Bind(in)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if in.ForgeID < defaultForgeID {
		in.ForgeID = defaultForgeID
	}

	user, err := store.FromContext(c).GetUserByRemoteID(in.ForgeID, in.ForgeRemoteID)
	if err != nil && !errors.Is(err, types.RecordNotExist) {
		handleDBError(c, err)
		return
	}

	if user == nil {
		user, err = _store.GetUserByLogin(in.ForgeID, c.Param("login"))
		if err != nil {
			handleDBError(c, err)
			return
		}
	}

	// TODO: disallow to change login, email, avatar if the user is using oauth
	user.Login = in.Login
	user.Email = in.Email
	user.Avatar = in.Avatar
	user.Admin = in.Admin

	err = _store.UpdateUser(user)
	if err != nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	c.JSON(http.StatusOK, user)
}

// PostUser
//
//	@Summary		Create a user
//	@Description	Creates a new user account with the specified external login. Requires admin rights.
//	@Router			/users [post]
//	@Produce		json
//	@Success		200	{object}	User
//	@Tags			Users
//	@Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param			user			body	User	true	"the user's data"
type userCreateRequest struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	ForgeID  int64  `json:"forge_id"`
	Admin    bool   `json:"admin"`
	Password string `json:"password"`
}

func PostUser(c *gin.Context) {
	_store := store.FromContext(c)
	in := &userCreateRequest{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	login := strings.TrimSpace(in.Login)
	if login == "" {
		c.String(http.StatusBadRequest, "login is required")
		return
	}
	email := strings.TrimSpace(in.Email)
	forgeID := in.ForgeID
	if forgeID == 0 {
		forgeID = defaultForgeID
	}

	var user *model.User
	user, err = _store.GetUserByLogin(forgeID, login)
	if err != nil {
		if errors.Is(err, types.RecordNotExist) {
			user = nil
		} else {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	if user == nil && email != "" {
		user, err = _store.GetUserByEmail(forgeID, email)
		if err != nil {
			if errors.Is(err, types.RecordNotExist) {
				user = nil
			} else {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	createdUser := false
	if user == nil {
		user = &model.User{
			Login:  login,
			Email:  email,
			Avatar: in.Avatar,
			Hash: base32.StdEncoding.EncodeToString(
				random.GetRandomBytes(32),
			),
			ForgeID:       forgeID,
			ForgeRemoteID: model.ForgeRemoteID("0"), // TODO: search for the user in the forge and get the remote id
		}
		user.Admin = in.Admin
		if err = user.Validate(); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		if err = _store.CreateUser(user); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		createdUser = true
	}

	password := strings.TrimSpace(in.Password)
	if password != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			if createdUser {
				_ = _store.DeleteUser(user)
			}
			c.String(http.StatusInternalServerError, hashErr.Error())
			return
		}
		existingAuth, authErr := _store.AuthUserFindByUsername(login)
		if authErr != nil && !errors.Is(authErr, types.RecordNotExist) {
			if createdUser {
				_ = _store.DeleteUser(user)
			}
			c.String(http.StatusInternalServerError, authErr.Error())
			return
		}
		if existingAuth != nil {
			existingAuth.UserID = user.ID
			existingAuth.Username = login
			existingAuth.PasswordHash = string(hash)
			if err := _store.AuthUserUpdate(existingAuth); err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			authUser := &model.AuthUser{
				UserID:       user.ID,
				Username:     login,
				PasswordHash: string(hash),
			}
			if err := _store.AuthUserCreate(authUser); err != nil {
				if createdUser {
					_ = _store.DeleteUser(user)
				}
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
		}
	}
	c.JSON(http.StatusOK, user)
}

// DeleteUser
//
//	@Summary		Delete a user
//	@Description	Deletes the given user. Requires admin rights.
//	@Router			/users/{login} [delete]
//	@Produce		plain
//	@Success		204
//	@Tags			Users
//	@Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
//	@Param			login			path	string	true	"the user's login name"
//	@Param			forge_id		query	string	true	"specify forge (else default will be used)"
//	@Param			forge_remote_id	query	string	false	"specify user id at forge (else fallback to login)"
func DeleteUser(c *gin.Context) {
	_store := store.FromContext(c)

	forgeID, err := strconv.ParseInt(c.DefaultQuery("forge_id", fmt.Sprint(defaultForgeID)), 10, 64)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	forgeRemoteID := model.ForgeRemoteID(c.Query("forge_remote_id"))

	var user *model.User

	if forgeRemoteID.IsValid() {
		user, err = store.FromContext(c).GetUserByRemoteID(forgeID, forgeRemoteID)
	} else {
		user, err = store.FromContext(c).GetUserByLogin(forgeID, c.Param("login"))
	}
	if err != nil {
		handleDBError(c, err)
		return
	}
	if err = _store.DeleteUser(user); err != nil {
		handleDBError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

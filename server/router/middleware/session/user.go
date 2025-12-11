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

package session

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
	"go.woodpecker-ci.org/woodpecker/v3/shared/token"
)

func User(c *gin.Context) *model.User {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	u, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return u
}

func SetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *model.User

		t, err := token.ParseRequest([]token.Type{token.UserToken, token.SessToken}, c.Request, func(t *token.Token) (string, error) {
			var err error
			userID, err := strconv.ParseInt(t.Get("user-id"), 10, 64)
			if err != nil {
				return "", err
			}
			user, err = store.FromContext(c).GetUser(userID)
			return user.Hash, err
		})
		if err == nil {
			c.Set("user", user)

			// if this is a session token (ie not the API token)
			// this means the user is accessing with a web browser,
			// so we should implement CSRF protection measures.
			if t.Type == token.SessToken {
				err = token.CheckCsrf(c.Request, func(_ *token.Token) (string, error) {
					return user.Hash, nil
				})
				// if csrf token validation fails, exit immediately
				// with a not authorized error.
				if err != nil {
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
			}
		}
		if user == nil {
			if raw := getStaticAPIToken(c.Request); raw != "" {
				user, err = authenticateRequestToken(c, raw)
				if err != nil {
					log.Error().Err(err).Msg("failed to authenticate static API token")
				} else if user != nil {
					c.Set("user", user)
				}
			}
		}
		c.Next()
	}
}

func MustAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		switch {
		case user == nil:
			c.String(http.StatusUnauthorized, "User not authorized")
			c.Abort()
		case !user.Admin:
			c.String(http.StatusForbidden, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}

func MustRepoAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		perm := Perm(c)
		switch {
		case user == nil:
			c.String(http.StatusUnauthorized, "User not authorized")
			c.Abort()
		case !perm.Admin:
			c.String(http.StatusForbidden, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}

func MustUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		switch user {
		case nil:
			c.String(http.StatusUnauthorized, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}

func MustOrgMember(admin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		if user == nil {
			c.String(http.StatusUnauthorized, "User not authorized")
			c.Abort()
			return
		}

		org := Org(c)
		if org == nil {
			c.String(http.StatusBadRequest, "Organization not loaded")
			c.Abort()
			return
		}

		// User can access his own, admin can access all
		if (org.Name == user.Login) || user.Admin {
			c.Next()
			return
		}

		_forge, err := server.Config.Services.Manager.ForgeFromUser(user)
		if err != nil {
			log.Error().Err(err).Msg("Cannot get forge from user")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		perm, err := server.Config.Services.Membership.Get(c, _forge, user, org.Name)
		if err != nil {
			log.Error().Err(err).Msg("failed to check membership")
			c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			c.Abort()
			return
		}

		if perm == nil || (!admin && !perm.Member) || (admin && !perm.Admin) {
			c.String(http.StatusForbidden, "user not authorized")
			c.Abort()
			return
		}

		c.Next()
	}
}

func authenticateRequestToken(c *gin.Context, presentedToken string) (*model.User, error) {
	if presentedToken == "" {
		return nil, nil
	}

	if user, err := authenticateAdminToken(c, presentedToken); err != nil || user != nil {
		return user, err
	}

	return nil, nil
}

func authenticateAdminToken(c *gin.Context, presentedToken string) (*model.User, error) {
	expected := server.Config.Server.AdminToken
	if expected == "" {
		return nil, nil
	}
	if subtle.ConstantTimeCompare([]byte(presentedToken), []byte(expected)) != 1 {
		return nil, nil
	}

	adminLogin := server.Config.Server.AdminTokenUser
	if adminLogin == "" {
		return nil, fmt.Errorf("admin token configured but no WOODPECKER_ADMIN user available")
	}

	user, err := findAdminUserByLogin(c, adminLogin)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, fmt.Errorf("user %q is not marked as admin", user.Login)
	}

	return user, nil
}

func findAdminUserByLogin(c *gin.Context, login string) (*model.User, error) {
	_store := store.FromContext(c)
	forges, err := _store.ForgeList(&model.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	if len(forges) == 0 {
		user, err := _store.GetUserByLogin(1, login)
		if err != nil && !errors.Is(err, types.RecordNotExist) {
			return nil, err
		}
		if err == nil {
			return user, nil
		}
	}

	for _, forge := range forges {
		user, err := _store.GetUserByLogin(forge.ID, login)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, types.RecordNotExist) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("admin user %q not found", login)
}

func getStaticAPIToken(r *http.Request) string {
	if token := extractFromAuthorizationHeader(r.Header.Get("Authorization")); token != "" {
		return token
	}
	return r.Header.Get("X-Woodpecker-Token")
}

func extractFromAuthorizationHeader(header string) string {
	if header == "" {
		return ""
	}

	parts := strings.Fields(header)
	if len(parts) != 2 {
		return ""
	}

	switch strings.ToLower(parts[0]) {
	case "bearer", "token":
		return parts[1]
	default:
		return ""
	}
}

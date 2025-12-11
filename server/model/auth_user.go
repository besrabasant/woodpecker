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

package model

import "errors"

var (
	errAuthUserInvalidLogin    = errors.New("invalid auth user login")
	errAuthUserInvalidPassword = errors.New("invalid auth user password")
)

// AuthUser represents application-local credentials that map to a Woodpecker user.
type AuthUser struct {
	ID           int64  `json:"id" xorm:"pk autoincr 'id'"`
	UserID       int64  `json:"user_id" xorm:"UNIQUE 'user_id'"`
	Username     string `json:"username" xorm:"VARCHAR(255) UNIQUE NOT NULL 'username'"`
	PasswordHash string `json:"-" xorm:"TEXT 'password_hash'"`
	Created      int64  `json:"created" xorm:"created"`
	Updated      int64  `json:"updated" xorm:"updated"`
} //	@name	AuthUser

func (AuthUser) TableName() string {
	return "auth_users"
}

// Validate ensures the auth user struct contains the required data.
func (a *AuthUser) Validate() error {
	switch {
	case a.UserID == 0:
		return errAuthUserInvalidLogin
	case len(a.Username) == 0 || len(a.Username) > maxLoginLen:
		return errAuthUserInvalidLogin
	case !reUsername.MatchString(a.Username):
		return errAuthUserInvalidLogin
	case len(a.PasswordHash) == 0:
		return errAuthUserInvalidPassword
	default:
		return nil
	}
}

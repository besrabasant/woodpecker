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

package datastore

import "go.woodpecker-ci.org/woodpecker/v3/server/model"

func (s storage) AuthUserFindByUsername(username string) (*model.AuthUser, error) {
	authUser := new(model.AuthUser)
	return authUser, wrapGet(s.engine.Where("username = ?", username).Get(authUser))
}

func (s storage) AuthUserCreate(authUser *model.AuthUser) error {
	_, err := s.engine.Insert(authUser)
	return err
}

func (s storage) AuthUserUpdate(authUser *model.AuthUser) error {
	_, err := s.engine.ID(authUser.ID).AllCols().Update(authUser)
	return err
}

func (s storage) AuthUserDelete(authUser *model.AuthUser) error {
	return wrapDelete(s.engine.ID(authUser.ID).Delete(new(model.AuthUser)))
}

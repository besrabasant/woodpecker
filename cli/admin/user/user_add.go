// Copyright 2023 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"

	"go.woodpecker-ci.org/woodpecker/v3/cli/internal"
	"go.woodpecker-ci.org/woodpecker/v3/woodpecker-go/woodpecker"
)

var userAddCmd = &cli.Command{
	Name:      "add",
	Usage:     "add a user",
	ArgsUsage: "[username]",
	Action:    userAdd,
}

func userAdd(ctx context.Context, c *cli.Command) error {
	login := strings.TrimSpace(c.Args().First())
	email := ""
	password := ""
	admin := false

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&login).
				Validate(func(v string) error {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("username is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Email (optional)").
				Value(&email),
			huh.NewConfirm().
				Title("Grant admin access?").
				Affirmative("Yes").
				Negative("No").
				Value(&admin),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(func(v string) error {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("password is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	login = strings.TrimSpace(login)
	if login == "" {
		return fmt.Errorf("username is required")
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("password is required")
	}
	email = strings.TrimSpace(email)

	client, err := internal.NewClient(ctx, c)
	if err != nil {
		return err
	}

	user, err := client.UserPost(&woodpecker.User{
		Login:    login,
		Email:    email,
		Admin:    admin,
		Password: password,
	})
	if err != nil {
		var clientErr *woodpecker.ClientError
		if errors.As(err, &clientErr) {
			msg := strings.TrimSpace(clientErr.Message)
			if msg == "" {
				msg = http.StatusText(clientErr.StatusCode)
			}
			if strings.HasPrefix(msg, "<?xml") {
				msg = fmt.Sprintf("server responded with status %d", clientErr.StatusCode)
			}
			lower := strings.ToLower(msg)
			if clientErr.StatusCode == http.StatusConflict || strings.Contains(lower, "already exist") || strings.Contains(lower, "duplicate") {
				return fmt.Errorf("user %s already exists", login)
			}
			return fmt.Errorf("failed to create user: %s", msg)
		}
		return err
	}
	fmt.Printf("Successfully added user %s\n", user.Login)
	return nil
}

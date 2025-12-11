package auth

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	goauth "github.com/shaj13/go-guardian/v2/auth"
	authjwt "github.com/shaj13/go-guardian/v2/auth/strategies/jwt"
	libcache "github.com/shaj13/libcache"
	_ "github.com/shaj13/libcache/lru"

	"go.woodpecker-ci.org/woodpecker/v3/server/model"
)

var (
	jwtKeeper     authjwt.SecretsKeeper
	jwtStrategy   goauth.Strategy
	jwtOptions    []goauth.Option
	jwtCache      libcache.Cache
	configureOnce sync.Once
)

// ConfigureJWT initializes the go-guardian JWT strategy.
func ConfigureJWT(secret string, issuer string, audience string, ttl time.Duration) {
	configureOnce.Do(func() {
		if ttl <= 0 {
			ttl = time.Hour
		}
		jwtKeeper = authjwt.StaticSecret{
			ID:        "woodpecker-jwt",
			Secret:    []byte(secret),
			Algorithm: authjwt.HS256,
		}
		jwtCache = libcache.LRU.New(0)
		jwtCache.SetTTL(ttl)
		jwtOptions = []goauth.Option{
			authjwt.SetIssuer(issuer),
			authjwt.SetAudience(audience),
			authjwt.SetExpDuration(ttl),
		}
		jwtStrategy = authjwt.New(jwtCache, jwtKeeper, jwtOptions...)
	})
}

// IssueJWT issues a JWT token for the provided user.
func IssueJWT(user *model.User) (string, error) {
	if jwtKeeper == nil {
		return "", errors.New("jwt not configured")
	}
	extensions := goauth.Extensions{}
	extensions.Set("login", user.Login)
	if user.Admin {
		extensions.Set("admin", "true")
	}
	info := goauth.NewUserInfo(user.Login, strconv.FormatInt(user.ID, 10), nil, extensions)
	return authjwt.IssueAccessToken(info, jwtKeeper, jwtOptions...)
}

// AuthenticateRequest authenticates the HTTP request using the configured JWT strategy.
func AuthenticateRequest(r *http.Request) (goauth.Info, error) {
	if jwtStrategy == nil {
		return nil, errors.New("jwt strategy not configured")
	}
	return jwtStrategy.Authenticate(r.Context(), r)
}

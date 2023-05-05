package main

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/connectors/redis"
)

func main() {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJodHRwczovL29wc3dheS5pbyIsImV4cCI6MTY4NDU3NTc1MiwianRpIjoiNGQwMDc3NGQtNjc4Mi00MDczLThlNWMtYjRmYTNhYjAwNjExIiwiaWF0IjoxNjgxOTgzNzUyLCJpc3MiOiJvcHN3YXkuaW8iLCJuYmYiOjE2ODE5ODM3NTIsInN1YiI6IjEiLCJ0eXBlIjoicmVmcmVzaF90b2tlbiJ9.uQESrgreumeYiyImEJEXbvp5Jil4QFUfKF10AHn1YR8"

	redisCli, err := redis.NewClient(context.Background(), redis.Config{
		Host: "localhost",
		Port: 6379,
		DB:   0,
	})
	if err != nil {
		panic(err)
	}

	authService := authentication.NewService(authentication.Config{
		Secret:           "secret",
		ExpiresIn:        time.Second * 10,
		RefreshExpiresIn: time.Second * 600,
		Issuer:           "opsway.io",
		Audience:         "https://opsway.io",
		CookieDomain:     "localhost",
	}, redisCli)

	access, refresh, err := authService.Refresh(context.Background(), token)
	if err != nil {
		panic(err)
	}

	println(access, refresh)
}

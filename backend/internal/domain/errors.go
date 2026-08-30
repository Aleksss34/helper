package domain

import "fmt"

var ErrUserExists = fmt.Errorf("user already exists")
var ErrUsernameExists = fmt.Errorf("username already exists")
var ErrEmailExists = fmt.Errorf("email already exists")
var ErrPassSimple = fmt.Errorf("password is simple")
var ErrInvalidUsername = fmt.Errorf("invalid username")
var ErrInvalidPass = fmt.Errorf("invalid password")
var ErrExpiredRefreshToken = fmt.Errorf("expired refresh token")
var ErrLimitReached = fmt.Errorf("the limit has been reached")

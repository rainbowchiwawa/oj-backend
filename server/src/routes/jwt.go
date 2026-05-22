package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"oj/server/database"
	"oj/server/utility"
	"time"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	jwtIdentityKey  = "jid"
	userIdentityKey = "uid"
)

type JWTData struct {
	Id        string
	UserId    string
	UserType  database.UserType
	UserName  string
	ExpiredAt float64
}

func payloadFunc() func(data any) gojwt.MapClaims {
	return func(data any) gojwt.MapClaims {
		if v, ok := data.(*JWTData); ok {
			return gojwt.MapClaims{
				jwtIdentityKey:  uuid.NewString(),
				userIdentityKey: v.UserId,
			}
		}
		return gojwt.MapClaims{}
	}
}

func identityHandler() func(ctx *gin.Context) any {
	return func(ctx *gin.Context) any {
		claims := jwt.ExtractClaims(ctx)
		return &JWTData{
			Id:        claims[jwtIdentityKey].(string),
			UserId:    claims[userIdentityKey].(string),
			ExpiredAt: claims["exp"].(float64),
		}
	}
}

func authenticator() func(ctx *gin.Context) (any, error) {
	return func(ctx *gin.Context) (any, error) {
		var body UserLoginRequest
		if err := ctx.Bind(&body); err != nil {
			return "", jwt.ErrMissingLoginValues
		}

		user, err := database.GetUserByName(body.Name)
		if err != nil {
			fmt.Println(err)
			return nil, jwt.ErrFailedAuthentication
		}

		hash := sha256.Sum256([]byte(body.Password))
		if hex.EncodeToString(hash[:]) != user.PasswordHash {
			return nil, jwt.ErrFailedAuthentication
		}

		return &JWTData{
			UserId:   user.Id.String(),
			UserType: user.Type,
			UserName: user.Name,
		}, nil
	}
}

func authorizer(userType database.UserType) func(ctx *gin.Context, data any) bool {
	return func(ctx *gin.Context, data any) bool {
		if v, ok := data.(*JWTData); ok {
			tokenValid := database.IsTokenValid(v.Id)
			userValid, err := database.IsUserValid(v.UserId, userType)
			if err != nil {
				return false
			}

			return tokenValid && userValid
		}
		return false
	}
}

func unauthorized() func(ctx *gin.Context, code int, message string) {
	return func(ctx *gin.Context, code int, message string) {
		ctx.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
}

func logoutResponse() func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		claims := jwt.ExtractClaims(ctx)
		database.CreateInvalidToken(claims[jwtIdentityKey].(string), claims["exp"].(float64))
		database.ClearExpiredToken()
		ctx.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "Logout successfully",
		})
	}
}

func JWTInitParams(userType database.UserType) *jwt.GinJWTMiddleware {
	return &jwt.GinJWTMiddleware{
		Realm:           "oj-server",
		Key:             []byte(utility.EnvData.JWTSecret),
		Timeout:         time.Hour,
		MaxRefresh:      time.Hour,
		IdentityKey:     userIdentityKey,
		PayloadFunc:     payloadFunc(),
		IdentityHandler: identityHandler(),
		Authenticator:   authenticator(),
		Authorizer:      authorizer(userType),
		Unauthorized:    unauthorized(),
		LogoutResponse:  logoutResponse(),
		TokenLookup:     "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:   "Bearer",
		TimeFunc:        time.Now,
	}
}

func GetUser(ctx *gin.Context) *JWTData {
	data, _ := ctx.Get(userIdentityKey)
	user, _ := database.GetUserById(data.(*JWTData).UserId)
	return &JWTData{
		UserId:   user.Id.String(),
		UserType: user.Type,
		UserName: user.Name,
	}
}

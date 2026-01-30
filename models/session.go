package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
)

type Session struct {
	bun.BaseModel `bun:"table:session"`

	Id int64 `bun:"id,pk"`

	RefreshToken string `bun:"refresh_token"`
	IsValid      bool   `bun:"is_valid"`
}

type RefreshTokenClaimFields struct {
	Id int64 `json:"id"`
}

type RefreshTokenClaim struct {
	jwt.RegisteredClaims
	RefreshTokenClaimFields `json:",inline"`
}

type AccessTokenClaimFields struct {
	Id       int64        `json:"id"`
	Resource UserResource `json:"resource"`
	Privs    Privileges   `json:"privs"`
}

type AccessTokenClaim struct {
	jwt.RegisteredClaims
	AccessTokenClaimFields `json:",inline"`
}

func (self *Session) RotateRefreshToken() error {
	var err error
	self.RefreshToken, err =
		jwt.NewWithClaims(jwt.SigningMethodHS256, RefreshTokenClaim{
			RefreshTokenClaimFields: RefreshTokenClaimFields{
				Id: self.Id,
			},
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt: &jwt.NumericDate{time.Now()},
				ExpiresAt: &jwt.NumericDate{time.Now().Add(time.Duration(
					viper.GetInt64("refresh-token-timeout")) * time.Second)},
			}}).SignedString([]byte(viper.GetString("refresh-token-secret")))
	return err
}

func (self *Session) ToAccessToken(
	resource UserResource,
) (string, error) {
	var privs Privileges
	if resource.IsAdmin {
		privs = NewAdminPrivs(self.Id)
	} else {
		privs = NewUserPrivs(self.Id)
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, AccessTokenClaim{
		AccessTokenClaimFields: AccessTokenClaimFields{
			Id:       self.Id,
			Resource: resource,
			Privs:    privs,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: &jwt.NumericDate{time.Now()},
			ExpiresAt: &jwt.NumericDate{time.Now().Add(time.Duration(
				viper.GetInt64("access-token-timeout")) * time.Second)},
		}}).SignedString([]byte(viper.GetString("access-token-secret")))
}

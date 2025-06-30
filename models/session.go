package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
)

type Session struct {
	bun.BaseModel `bun:"table:session"`

	CsId   string   `bun:"cs_id,pk"`
	Groups []string `bun:"groups"`

	RefreshToken string `bun:"refresh_token"`
	IsValid      bool   `bun:"is_valid"`
}

type RefreshTokenClaimFields struct {
	CsId string `json:"csId"`
}

type RefreshTokenClaim struct {
	jwt.RegisteredClaims
	RefreshTokenClaimFields `json:",inline"`
}

type AccessTokenClaimFields struct {
	CsId     string       `json:"csId"`
	Groups   []string     `json:"groups"`
	Resource UserResource `json:"resource"`
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
				CsId: self.CsId,
			},
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt: &jwt.NumericDate{time.Now()},
				ExpiresAt: &jwt.NumericDate{time.Now().Add(time.Duration(
					viper.GetInt64("REFRESH_TOKEN_TIMEOUT")) * time.Second)},
			}}).SignedString([]byte(viper.GetString("REFRESH_TOKEN_SECRET")))
	return err
}

func (self *Session) ToAccessToken(
	resource UserResource,
) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, AccessTokenClaim{
		AccessTokenClaimFields: AccessTokenClaimFields{
			CsId:     self.CsId,
			Groups:   self.Groups,
			Resource: resource,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: &jwt.NumericDate{time.Now()},
			ExpiresAt: &jwt.NumericDate{time.Now().Add(time.Duration(
				viper.GetInt64("ACCESS_TOKEN_TIMEOUT")) * time.Second)},
		}}).SignedString([]byte(viper.GetString("ACCESS_TOKEN_SECRET")))
}

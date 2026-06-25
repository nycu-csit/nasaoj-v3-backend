package workflows

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jobToken JWTs are signed by the worker using its own worker token as the HMAC
// key. The worker hands a jobToken to each job (baked into the job tar) and
// later verifies it locally when the job calls back into the worker API —
// without any round-trip to the backend.

type JobClaims struct {
	SubId     int64  `json:"subId"`
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`
	OwnerId   int64  `json:"ownerId"`
	jwt.RegisteredClaims
}

// SignJobToken mints a jobToken signed with the worker's secret (its worker
// token).
func SignJobToken(
	secret []byte,
	subId int64,
	stageName string,
	testName string,
	ownerId int64,
	ttl time.Duration,
) (string, error) {
	now := time.Now()
	claims := JobClaims{
		SubId:     subId,
		StageName: stageName,
		TestName:  testName,
		OwnerId:   ownerId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// VerifyJobToken validates a jobToken's signature and expiry and returns its
// claims.
func VerifyJobToken(
	secret []byte,
	tokenStr string,
) (JobClaims, error) {
	var claims JobClaims
	_, err := jwt.ParseWithClaims(tokenStr, &claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf(
					"unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
	if err != nil {
		return JobClaims{}, err
	}
	return claims, nil
}

module gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend

replace gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models => ../models

replace gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs => ./docs

replace gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers => ./handlers

replace gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares => ./middlewares

go 1.24.0

require (
	github.com/coreos/go-oidc/v3 v3.12.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/joho/godotenv v1.5.1
	github.com/minio/minio-go/v7 v7.0.86
	github.com/redis/go-redis/v9 v9.7.0
	github.com/swaggo/http-swagger/v2 v2.0.2
	github.com/swaggo/swag v1.16.4
	github.com/uptrace/bunrouter v1.0.22
	gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.26.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/swaggo/files/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

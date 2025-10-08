module api-gateway

go 1.25.1

require (
	auth-service v0.0.0-00010101000000-000000000000
	comment-service v0.0.0-00010101000000-000000000000
	feed-service v0.0.0-00010101000000-000000000000
	follow-service v0.0.0-00010101000000-000000000000
	github.com/99designs/gqlgen v0.17.81
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.46.1
	github.com/vektah/gqlparser/v2 v2.5.30
	like-service v0.0.0-00010101000000-000000000000
	notification-service v0.0.0-00010101000000-000000000000
	post-service v0.0.0-00010101000000-000000000000
	user-service v0.0.0-00010101000000-000000000000
)

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

require (
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/sosodev/duration v1.3.1 // indirect
	google.golang.org/grpc v1.75.1
)

replace auth-service => ../auth-service

replace post-service => ../post-service

replace comment-service => ../comment-service

replace follow-service => ../follow-service

replace like-service => ../like-service

replace notification-service => ../notification-service

replace user-service => ../user-service

replace feed-service => ../feed-service

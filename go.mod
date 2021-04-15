module github.com/getchill-app/service

go 1.16

require (
	github.com/alta/protopatch v0.3.3
	github.com/davecgh/go-spew v1.1.1
	github.com/getchill-app/http/client v0.0.0-20210413003944-fa4aeeaab394
	github.com/getchill-app/http/server v0.0.0-20210412222146-088571f8d3a6
	github.com/getchill-app/messaging v0.0.0-20210328173043-840bde55799b
	github.com/getchill-app/ws v0.0.0-20210402213525-39307cde11c0
	github.com/getchill-app/ws/client v0.0.0-20210402213525-39307cde11c0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/keys-pub/keys v0.1.22-0.20210412214905-995329cc5e85
	github.com/keys-pub/keys-ext/auth/fido2 v0.0.0-20210327130412-59e9fcfcf22c
	github.com/keys-pub/keys-ext/http/api v0.0.0-20210401205654-ff14cd298c61
	github.com/keys-pub/keys-ext/http/client v0.0.0-20210327130412-59e9fcfcf22c
	github.com/keys-pub/keys-ext/http/server v0.0.0-20210401205934-8b752a983cd9
	github.com/keys-pub/keys-ext/sqlcipher v0.0.0-20210327130412-59e9fcfcf22c
	github.com/keys-pub/vault v0.0.0-20210403222024-d7c66fea4997
	github.com/mercari/go-grpc-interceptor v0.0.0-20180110035004-b8ad3827e82a
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.36.1
	google.golang.org/protobuf v1.26.0
)

replace github.com/keys-pub/keys => ../../keys.pub/keys

replace github.com/keys-pub/vault => ../../keys.pub/vault

replace github.com/keys-pub/keys-ext/http/server => ../../keys.pub/keys-ext/http/server

replace github.com/keys-pub/keys-ext/http/client => ../../keys.pub/keys-ext/http/client

replace github.com/getchill-app/http/client => ../http/client

replace github.com/getchill-app/http/server => ../http/server

replace github.com/getchill-app/http/api => ../http/api

replace github.com/getchill-app/messaging => ../messaging

replace github.com/getchill/ws => ../../getchill-app/ws

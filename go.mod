module github.com/getchill-app/service

go 1.16

require (
	github.com/alta/protopatch v0.3.4
	github.com/davecgh/go-spew v1.1.1
	github.com/getchill-app/http/api v0.0.0-20210504010216-724792fd62e1
	github.com/getchill-app/http/client v0.0.0-20210504011100-0d36c616cd37
	github.com/getchill-app/http/server v0.0.0-20210504010821-957671867b63
	github.com/getchill-app/keyring v0.0.0-20210430214439-c21449557217
	github.com/getchill-app/messaging v0.0.0-20210328173043-840bde55799b
	github.com/getchill-app/ws/api v0.0.0-20210504010258-9e1738caa70d
	github.com/getchill-app/ws/client v0.0.0-20210402213525-39307cde11c0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/keys-pub/keys v0.1.22-0.20210428191820-49dfbda60f85
	github.com/keys-pub/keys-ext/auth/fido2 v0.0.0-20210415150208-d90ac8efc4fe
	github.com/keys-pub/keys-ext/http/api v0.0.0-20210401205654-ff14cd298c61
	github.com/keys-pub/keys-ext/http/client v0.0.0-20210327130412-59e9fcfcf22c
	github.com/keys-pub/keys-ext/http/server v0.0.0-20210401205934-8b752a983cd9
	github.com/keys-pub/keys-ext/sqlcipher v0.0.0-20210327130412-59e9fcfcf22c
	github.com/mercari/go-grpc-interceptor v0.0.0-20180110035004-b8ad3827e82a
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/keys-pub/keys => ../../keys.pub/keys

replace github.com/keys-pub/keys-ext/http/server => ../../keys.pub/keys-ext/http/server

replace github.com/keys-pub/keys-ext/http/client => ../../keys.pub/keys-ext/http/client

replace github.com/getchill-app/keyring => ../keyring

replace github.com/getchill-app/http/client => ../http/client

replace github.com/getchill-app/http/server => ../http/server

replace github.com/getchill-app/http/api => ../http/api

replace github.com/getchill-app/messaging => ../messaging

replace github.com/getchill-app/ws/client => ../../getchill/ws/client

replace github.com/getchill-app/ws/api => ../../getchill/ws/api

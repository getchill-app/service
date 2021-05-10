package service

import (
	"github.com/getchill-app/keyring"
	"github.com/getchill-app/messaging"
	wsclient "github.com/getchill-app/ws/client"
	"github.com/keys-pub/keys"
	hclient "github.com/keys-pub/keys/http/client"
	"github.com/keys-pub/keys/user"
	"github.com/keys-pub/keys/user/services"
	"github.com/keys-pub/keys/users"
	"github.com/sirupsen/logrus"
)

func setupLogPackages(lg *logrus.Logger) {
	SetLogger(lg)
	hclient.SetLogger(newPackageLogger(lg, "http/client"))
	keys.SetLogger(newPackageLogger(lg, "keys"))
	keyring.SetLogger(newPackageLogger(lg, "keyring"))
	messaging.SetLogger(newPackageLogger(lg, "messaging"))
	user.SetLogger(newPackageLogger(lg, "user"))
	users.SetLogger(newPackageLogger(lg, "users"))
	services.SetLogger(newPackageLogger(lg, "services"))
	wsclient.SetLogger(newPackageLogger(lg, "ws/client"))
}

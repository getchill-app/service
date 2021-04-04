package service

import (
	hclient "github.com/getchill-app/http-client"
	"github.com/getchill-app/messaging"
	wsclient "github.com/getchill-app/ws/client"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/user"
	"github.com/keys-pub/keys/user/services"
	"github.com/keys-pub/keys/users"
	"github.com/keys-pub/vault"
	"github.com/keys-pub/vault/auth"
	vclient "github.com/keys-pub/vault/client"
	"github.com/keys-pub/vault/syncer"
	"github.com/sirupsen/logrus"
)

func setupLogPackages(lg *logrus.Logger) {
	SetLogger(lg)
	hclient.SetLogger(newPackageLogger(lg, "http/client"))
	keys.SetLogger(newPackageLogger(lg, "keys"))
	vault.SetLogger(newPackageLogger(lg, "vault"))
	messaging.SetLogger(newPackageLogger(lg, "messaging"))
	user.SetLogger(newPackageLogger(lg, "user"))
	users.SetLogger(newPackageLogger(lg, "users"))
	services.SetLogger(newPackageLogger(lg, "services"))
	syncer.SetLogger(newPackageLogger(lg, "vault/syncer"))
	vclient.SetLogger(newPackageLogger(lg, "vault/client"))
	auth.SetLogger(newPackageLogger(lg, "vault/auth"))
	wsclient.SetLogger(newPackageLogger(lg, "ws/client"))
}

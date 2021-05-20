package service

import (
	"context"
	sync "sync"

	"github.com/getchill-app/keyring"
	"github.com/getchill-app/keyring/auth"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/auth/fido2"
	kclient "github.com/keys-pub/keys-ext/http/client"
	"github.com/keys-pub/keys-ext/sqlcipher"
	"github.com/keys-pub/keys/tsutil"
	"github.com/keys-pub/keys/users"

	"github.com/getchill-app/http/client"
	"github.com/getchill-app/messaging"
)

type service struct {
	UnimplementedRPCServer

	env    *Env
	build  Build
	authIr *authInterceptor

	db      *sqlcipher.DB
	client  *client.Client
	kclient *kclient.Client
	scs     *keys.Sigchains
	users   *users.Users
	clock   tsutil.Clock

	keyring *keyring.Keyring

	unlockMtx sync.Mutex

	messenger *messaging.Messenger
	relay     *relay
}

func newService(
	env *Env,
	build Build,
	authIr *authInterceptor,
	fido2Plugin fido2.FIDO2Server,
	clock tsutil.Clock) (*service, error) {

	authPath, err := env.AppPath("auth.db", true)
	if err != nil {
		return nil, err
	}
	auth, err := auth.NewDB(authPath)
	if err != nil {
		return nil, err
	}

	path, err := env.AppPath("vault.db", true)
	if err != nil {
		return nil, err
	}
	keyring := keyring.New(path, auth)
	keyring.SetFIDO2Plugin(fido2Plugin)

	kclient, err := kclient.New(env.KeysPubServerURL())
	if err != nil {
		return nil, err
	}
	kclient.SetClock(clock)

	db := sqlcipher.New()
	db.SetClock(clock)
	scs := keys.NewSigchains(db)
	usrs := users.New(db, scs, users.HTTPClient(kclient.HTTPClient()), users.Clock(clock))

	relay := newRelay()

	client, err := client.New(env.ChillServerURL())
	client.SetClock(clock)
	if err != nil {
		return nil, err
	}

	return &service{
		authIr:  authIr,
		build:   build,
		env:     env,
		scs:     scs,
		users:   usrs,
		db:      db,
		client:  client,
		kclient: kclient,
		keyring: keyring,
		relay:   relay,
		clock:   clock,
	}, nil
}

func (s *service) Close() {
	if err := s.authLock(context.TODO()); err != nil {
		logger.Warningf("Failed to lock: %v", err)
	}
}

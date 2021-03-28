package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys-ext/http/server"
	"github.com/keys-pub/keys/dstore"
	"github.com/keys-pub/keys/http"
	"github.com/keys-pub/keys/tsutil"
	"github.com/keys-pub/keys/users"
	"github.com/stretchr/testify/require"
)

func newEnv(t *testing.T, appName string, serverURL string) (*Env, CloseFn) {
	if appName == "" {
		appName = "KeysTest-" + randName()
	}
	env, err := NewEnv(appName, build)
	require.NoError(t, err)
	env.Set(serverCfgKey, serverURL)

	closeFn := func() {
		removeErr := os.RemoveAll(env.AppDir())
		require.NoError(t, removeErr)
	}
	return env, closeFn
}

func randName() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}

func testFire(t *testing.T, clock tsutil.Clock) server.Fire {
	fi := dstore.NewMem()
	fi.SetClock(clock)
	return fi
}

type testServerEnv struct {
	clock  tsutil.Clock
	fi     server.Fire
	client http.Client
	users  *users.Users
}

func newTestServerEnv(t *testing.T) *testServerEnv {
	clock := tsutil.NewTestClock()
	fi := testFire(t, clock)
	client := http.NewClient()
	usrs := users.New(fi, keys.NewSigchains(fi), users.Client(client), users.Clock(clock))
	return &testServerEnv{
		clock:  clock,
		fi:     fi,
		client: client,
		users:  usrs,
	}
}

func newTestService(t *testing.T, serverEnv *testServerEnv) (*service, CloseFn) {
	httpServerEnv := newTestHTTPServerEnv(t, serverEnv)
	appName := "KeysTest-" + randName()

	env, closeFn := newEnv(t, appName, httpServerEnv.url)
	auth := newAuthInterceptor()

	svc, err := newService(env, Build{Version: "1.2.3", Commit: "deadbeef"}, auth, nil, serverEnv.clock)
	require.NoError(t, err)

	closeServiceFn := func() {
		httpServerEnv.closeFn()
		svc.Close()
		closeFn()
	}

	return svc, closeServiceFn
}

var authPassword = "testpassword"

func testAuthSetup(t *testing.T, service *service) {
	_, err := service.AuthSetup(context.TODO(), &AuthSetupRequest{
		Secret: authPassword,
		Type:   PasswordAuth,
	})
	require.NoError(t, err)
	_, err = service.AuthUnlock(context.TODO(), &AuthUnlockRequest{
		Secret: authPassword,
		Type:   PasswordAuth,
		Client: "test",
	})
	require.NoError(t, err)
}

func testAuthLock(t *testing.T, service *service) {
	_, err := service.AuthLock(context.TODO(), &AuthLockRequest{})
	require.NoError(t, err)
}

func testAuthUnlock(t *testing.T, service *service) {
	_, err := service.AuthUnlock(context.TODO(), &AuthUnlockRequest{
		Secret: authPassword,
		Type:   PasswordAuth,
		Client: "test",
	})
	require.NoError(t, err)
}

type testHTTPServerEnv struct {
	url     string
	closeFn func()
}

func newTestHTTPServerEnv(t *testing.T, env *testServerEnv) *testHTTPServerEnv {
	rds := server.NewRedisTest(env.clock)
	srv := server.New(env.fi, rds, env.client, env.clock, server.NewLogger(server.NoLevel))
	srv.SetClock(env.clock)
	tasks := server.NewTestTasks(srv)
	srv.SetTasks(tasks)
	srv.SetInternalAuth("testtoken")
	_ = srv.SetInternalKey("6a169a699f7683c04d127504a12ace3b326e8b56a61a9b315cf6b42e20d6a44a")
	handler := server.NewHandler(srv)
	testServer := httptest.NewServer(handler)
	srv.URL = testServer.URL

	closeFn := func() {
		testServer.Close()
	}
	return &testHTTPServerEnv{
		url:     srv.URL,
		closeFn: closeFn,
	}
}

func TestServiceCheck(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// vault.SetLogger(NewLogger(DebugLevel))
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()

	testAuthSetup(t, service)
	require.True(t, service.checking)

	testAuthLock(t, service)
	require.False(t, service.checking)

	testAuthUnlock(t, service)
	require.True(t, service.checking)

	testAuthLock(t, service)
	require.False(t, service.checking)
}

package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"net/http/httptest"
	"os"
	"testing"

	chillserver "github.com/getchill-app/http/server"
	"github.com/keys-pub/keys"
	kpserver "github.com/keys-pub/keys-ext/http/server"
	"github.com/keys-pub/keys/dstore"
	"github.com/keys-pub/keys/http"
	"github.com/keys-pub/keys/tsutil"
	"github.com/keys-pub/keys/users"
	"github.com/stretchr/testify/require"
)

func newEnv(t *testing.T, appName string, keysPubServerURL string, chillServerURL string) (*Env, CloseFn) {
	if appName == "" {
		appName = "KeysTest-" + randName()
	}
	env, err := NewEnv(appName, build)
	require.NoError(t, err)
	env.Set(keysPubServerCfgKey, keysPubServerURL)
	env.Set(chillServerCfgKey, chillServerURL)

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

func testFire(t *testing.T, clock tsutil.Clock) kpserver.Fire {
	fi := dstore.NewMem()
	fi.SetClock(clock)
	return fi
}

type testServerEnv struct {
	clock    tsutil.Clock
	fi       kpserver.Fire
	client   http.Client
	users    *users.Users
	logLevel LogLevel
}

func newTestServerEnv(t *testing.T) *testServerEnv {
	clock := tsutil.NewTestClock()
	fi := testFire(t, clock)
	client := http.NewClient()
	usrs := users.New(fi, keys.NewSigchains(fi), users.Client(client), users.Clock(clock))
	return &testServerEnv{
		clock:    clock,
		fi:       fi,
		client:   client,
		users:    usrs,
		logLevel: NoLevel,
	}
}

type serviceEnv struct {
	service        *service
	getChillAppEnv *testHTTPServerEnv
	keysPubEnv     *testHTTPServerEnv
}

func newTestServiceEnv(t *testing.T, serverEnv *testServerEnv) (*serviceEnv, CloseFn) {
	keysPubEnv := newTestKeysPubServerEnv(t, serverEnv)
	getChillAppEnv := newTestChillServerEnv(t, serverEnv)
	appName := "KeysTest-" + randName()

	env, closeFn := newEnv(t, appName, keysPubEnv.url, getChillAppEnv.url)
	auth := newAuthInterceptor()

	svc, err := newService(env, Build{Version: "1.2.3", Commit: "deadbeef"}, auth, nil, serverEnv.clock)
	require.NoError(t, err)

	closeServiceFn := func() {
		keysPubEnv.closeFn()
		getChillAppEnv.closeFn()
		svc.Close()
		closeFn()
	}

	return &serviceEnv{service: svc, getChillAppEnv: getChillAppEnv, keysPubEnv: keysPubEnv}, closeServiceFn
}

func newTestService(t *testing.T, serverEnv *testServerEnv) (*service, CloseFn) {
	serviceEnv, closeFn := newTestServiceEnv(t, serverEnv)
	return serviceEnv.service, closeFn
}

func testAccountSetup(t *testing.T, env *testServerEnv, service *service, email string, password string, account *keys.EdX25519Key) {
	req := &AccountCreateRequest{
		Email:    email,
		Password: password,
	}
	if account != nil {
		req.AccountKey = account.PaperKey()
	}
	_, err := service.AccountCreate(context.TODO(), req)
	require.NoError(t, err)
	testOrgCreate(t, env, service)
}

func testAccountCreate(t *testing.T, service *service, email string, password string) {
	ctx := context.TODO()
	req := &AccountCreateRequest{
		Email:    email,
		Password: password,
	}
	_, err := service.AccountCreate(ctx, req)
	require.NoError(t, err)
}

func testOrgCreate(t *testing.T, env *testServerEnv, service *service) {
	ctx := context.TODO()
	_, err := service.OrgKey(ctx, &OrgKeyRequest{Domain: "test.domain"})
	require.NoError(t, err)

	resp, err := service.OrgSign(ctx, &OrgSignRequest{
		Domain: "test.domain",
	})
	require.NoError(t, err)

	env.client.SetProxy("https://test.domain/.well-known/getchill.txt", func(ctx context.Context, req *http.Request) http.ProxyResponse {
		return http.ProxyResponse{Body: []byte(resp.Sig)}
	})
	_, err = service.OrgCreate(ctx, &OrgCreateRequest{
		Domain: "test.domain",
	})
	require.NoError(t, err)
}

type testHTTPServerEnv struct {
	url     string
	emailer *testEmailer
	closeFn func()
}

func newTestKeysPubServerEnv(t *testing.T, env *testServerEnv) *testHTTPServerEnv {
	rds := kpserver.NewRedisTest(env.clock)
	srv := kpserver.New(env.fi, rds, env.client, env.clock, kpserver.NewLogger(kpserver.LogLevel(env.logLevel)))
	srv.SetClock(env.clock)
	tasks := kpserver.NewTestTasks(srv)
	srv.SetTasks(tasks)
	srv.SetInternalAuth("testtoken")
	err := srv.SetInternalKey("6a169a699f7683c04d127504a12ace3b326e8b56a61a9b315cf6b42e20d6a44a")
	require.NoError(t, err)
	err = srv.SetTokenKey("f41deca7f9ef4f82e53cd7351a90bc370e2bf15ed74d147226439cfde740ac18")
	require.NoError(t, err)

	handler := kpserver.NewHandler(srv)
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

func newTestChillServerEnv(t *testing.T, env *testServerEnv) *testHTTPServerEnv {
	rds := chillserver.NewRedisTest(env.clock)
	srv := chillserver.New(env.fi, rds, env.client, env.clock, chillserver.NewLogger(chillserver.LogLevel(env.logLevel)))
	srv.SetClock(env.clock)
	err := srv.SetInternalKey("6a169a699f7683c04d127504a12ace3b326e8b56a61a9b315cf6b42e20d6a44a")
	require.NoError(t, err)
	err = srv.SetTokenKey("f41deca7f9ef4f82e53cd7351a90bc370e2bf15ed74d147226439cfde740ac18")
	require.NoError(t, err)
	emailer := newTestEmailer()
	srv.SetEmailer(emailer)

	handler := chillserver.NewHandler(srv)
	testServer := httptest.NewServer(handler)
	srv.URL = testServer.URL

	closeFn := func() {
		testServer.Close()
	}
	return &testHTTPServerEnv{
		url:     srv.URL,
		emailer: emailer,
		closeFn: closeFn,
	}
}

type testEmailer struct {
	sentVerificationEmail map[string]string
}

func newTestEmailer() *testEmailer {
	return &testEmailer{sentVerificationEmail: map[string]string{}}
}

func (t *testEmailer) SentVerificationEmail(email string) string {
	s := t.sentVerificationEmail[email]
	return s
}

func (t *testEmailer) SendVerificationEmail(email string, code string) error {
	t.sentVerificationEmail[email] = code
	return nil
}

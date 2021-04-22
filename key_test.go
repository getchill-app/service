package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/stretchr/testify/require"
)

func TestKey(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// keys.SetLogger(NewLogger(DebugLevel))
	// db.SetLogger(NewLogger(DebugLevel))

	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()

	testAccountCreate(t, service, "alice@keys.pub")
	testImportKey(t, service, alice)
	testUserSetupGithub(t, env, service, alice, "alice")

	// Alice
	resp, err := service.Key(ctx, &KeyRequest{
		Key: alice.ID().String(),
	})
	require.NoError(t, err)
	require.Equal(t, alice.ID().String(), resp.Key.ID)

	// Alice (user)
	resp, err = service.Key(ctx, &KeyRequest{
		Key: "alice@github",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Key)
	require.Equal(t, alice.ID().String(), resp.Key.ID)

	// Alice (X25519)
	resp, err = service.Key(ctx, &KeyRequest{
		Key: alice.PublicKey().X25519PublicKey().ID().String(),
	})
	require.NoError(t, err)
	require.Equal(t, alice.ID().String(), resp.Key.ID)

	// Key
	key := keys.NewEdX25519KeyFromSeed(keys.Bytes32(bytes.Repeat([]byte{0x03}, 32)))
	testImportKey(t, service, key)

	resp, err = service.Key(ctx, &KeyRequest{
		Key: "kex1a4yj333g68pvd6hfqvufqkv4vy54jfe6t33ljd3kc9rpfty8xlgs2u3qxr",
	})
	require.NoError(t, err)
	require.Equal(t, key.ID().String(), resp.Key.ID)
	resp, err = service.Key(ctx, &KeyRequest{
		Key: "kbx1wh38phef2tzhh2pk0w5xrrqh37072rdj0xwnqnn5ayvdnptgv9rqae3m96",
	})
	require.NoError(t, err)
	require.Equal(t, key.ID().String(), resp.Key.ID)

	// TODO: Test update
}

func TestKeyGenerate(t *testing.T) {
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()
	testAccountCreate(t, service, "alice@keys.pub")
	kr := service.vault.Keyring()

	// Generate EdX25519
	genResp, err := service.KeyGenerate(ctx, &KeyGenerateRequest{Type: string(keys.EdX25519)})
	require.NoError(t, err)

	resp, err := service.Key(ctx, &KeyRequest{
		Key: genResp.KID,
	})
	require.NoError(t, err)
	require.Equal(t, genResp.KID, resp.Key.ID)

	// Get EdX25519 key by X25519 ID
	kid, err := keys.ParseID(genResp.KID)
	require.NoError(t, err)
	sk, err := kr.Key(kid)
	require.NoError(t, err)
	bkid := sk.AsEdX25519().PublicKey().X25519PublicKey().ID()
	resp, err = service.Key(ctx, &KeyRequest{
		Key: bkid.String(),
	})
	require.NoError(t, err)
	require.Equal(t, kid.String(), resp.Key.ID)

	// Generate X25519
	genResp, err = service.KeyGenerate(ctx, &KeyGenerateRequest{Type: string(keys.X25519)})
	require.NoError(t, err)
	resp, err = service.Key(ctx, &KeyRequest{
		Key: genResp.KID,
	})
	require.NoError(t, err)
	require.Equal(t, genResp.KID, resp.Key.ID)

	// Generate invalid
	_, err = service.KeyGenerate(ctx, &KeyGenerateRequest{Type: "invalidkeytype"})
	require.EqualError(t, err, "unknown key type invalidkeytype")
}

func TestKeyRemove(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()
	testAccountCreate(t, service, "alice@keys.pub")
	testImportKey(t, service, alice)
	kr := service.vault.Keyring()

	genResp, err := service.KeyGenerate(ctx, &KeyGenerateRequest{Type: string(keys.EdX25519)})
	require.NoError(t, err)
	kid, err := keys.ParseID(genResp.KID)
	require.NoError(t, err)
	key, err := kr.Key(kid)
	require.NoError(t, err)
	require.NotNil(t, key)

	// Remove
	_, err = service.KeyRemove(ctx, &KeyRemoveRequest{KID: key.ID.String()})
	require.NoError(t, err)

	// Remove (X25519)
	genResp, err = service.KeyGenerate(ctx, &KeyGenerateRequest{Type: string(keys.X25519)})
	require.NoError(t, err)
	_, err = service.KeyRemove(ctx, &KeyRemoveRequest{KID: genResp.KID})
	require.NoError(t, err)

	// Remove (not found)
	randKey := keys.GenerateEdX25519Key()
	_, err = service.KeyRemove(ctx, &KeyRemoveRequest{KID: randKey.ID().String()})
	require.EqualError(t, err, fmt.Sprintf("%s not found", randKey.ID()))
}

func TestKeySearch(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	// services.SetLogger(NewLogger(DebugLevel))
	// users.SetLogger(NewLogger(DebugLevel))
	// user.SetLogger(NewLogger(DebugLevel))
	env := newTestServerEnv(t)
	service, closeFn := newTestService(t, env)
	defer closeFn()
	ctx := context.TODO()

	testAccountCreate(t, service, "alice@keys.pub")
	testImportKey(t, service, alice)
	testUserSetupGithub(t, env, service, alice, "alice")

	testImportKey(t, service, bob)
	testUserSetupGithub(t, env, service, bob, "bob")

	resp, err := service.KeySearch(ctx, &KeySearchRequest{Query: "alice"})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Keys))
	require.Equal(t, "kex132yw8ht5p8cetl2jmvknewjawt9xwzdlrk2pyxlnwjyqrdq0dawqqph077", resp.Keys[0].ID)
}

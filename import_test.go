package service

import (
	"context"
	"testing"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/stretchr/testify/require"
)

func TestKeyImport(t *testing.T) {
	// SetLogger(NewLogger(DebugLevel))
	env := newTestServerEnv(t)
	service, closeFn := testServiceSetup(t, env, "alice@keys.pub", alice)
	defer closeFn()
	ctx := context.TODO()

	key := keys.GenerateEdX25519Key()
	export, err := api.EncodeKey(api.NewKey(key), "testpassword")
	require.NoError(t, err)

	// Import
	importResp, err := service.KeyImport(ctx, &KeyImportRequest{
		In:       []byte(export),
		Password: "testpassword",
	})
	require.NoError(t, err)
	require.Equal(t, key.ID().String(), importResp.KID)

	keyResp, err := service.Key(ctx, &KeyRequest{Key: key.ID().String()})
	require.NoError(t, err)
	require.Equal(t, key.ID().String(), keyResp.Key.ID)

	// Check key
	kr := service.keyring
	out, err := kr.Key(key.ID())
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, out.ID, key.ID())

	sks, err := kr.KeysWithType(string(keys.EdX25519))
	require.NoError(t, err)
	require.Equal(t, 2, len(sks))

	// Import (bob, ID)
	importResp, err = service.KeyImport(ctx, &KeyImportRequest{
		In: []byte(bob.ID().String()),
	})
	require.NoError(t, err)
	require.Equal(t, bob.ID().String(), importResp.KID)

	// Import (charlie, ID with whitespace)
	importResp, err = service.KeyImport(ctx, &KeyImportRequest{
		In: []byte(charlie.ID().String() + "\n  "),
	})
	require.NoError(t, err)
	require.Equal(t, charlie.ID().String(), importResp.KID)

	// Import (error)
	_, err = service.KeyImport(ctx, &KeyImportRequest{In: []byte{}})
	require.EqualError(t, err, "failed to decode key")
}

func TestKeyImportSaltpack(t *testing.T) {
	msg := `BEGIN EDX25519 KEY MESSAGE.
	9tyMV66eX002JQT sWFyRoiUzCV1DFS Fl2nbyGGteXmU9M XoQcx1V9CKdUCPM
	EoszEpADNLrqULM 2MAcI8XOXSIsAFk 5peBObhA0I9IAZS OOkLndOHMOGHGCd
	dtMkQg08U1C4RtH PMpMj1RyNz9CyBF dNS9qrctSt0r.
	END EDX25519 KEY MESSAGE.`

	env := newTestServerEnv(t)
	service, closeFn := testServiceSetup(t, env, "alice@keys.pub", alice)
	defer closeFn()
	ctx := context.TODO()

	importResp, err := service.KeyImport(ctx, &KeyImportRequest{
		In:       []byte(msg),
		Password: "",
	})
	require.NoError(t, err)
	require.Equal(t, "kex16v9uk4t5wykkklpkrcane3p267n8eu95y3fd55yv4h45m6ku3hyqx2a5fn", importResp.KID)

	keysResp, err := service.Keys(ctx, &KeysRequest{})
	require.NoError(t, err)
	require.Equal(t, 2, len(keysResp.Keys))
	require.Equal(t, "kex132yw8ht5p8cetl2jmvknewjawt9xwzdlrk2pyxlnwjyqrdq0dawqqph077", keysResp.Keys[0].ID)
	require.Equal(t, "kex16v9uk4t5wykkklpkrcane3p267n8eu95y3fd55yv4h45m6ku3hyqx2a5fn", keysResp.Keys[1].ID)
}

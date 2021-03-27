package service

import (
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/tsutil"
)

func statementToRPC(st *keys.Statement) *Statement {
	return &Statement{
		Sig:       st.Sig,
		Data:      st.Data,
		KID:       st.KID.String(),
		Seq:       int32(st.Seq),
		Prev:      st.Prev,
		Revoke:    int32(st.Revoke),
		Timestamp: tsutil.Millis(st.Timestamp),
		Type:      st.Type,
	}
}

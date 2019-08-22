package blocklog

import "strings"

type StateChangeContext struct {
	branch  string
	trx, op int
	causes  []string
	trxId   string
	changes InternalStateChangeSlice
}

func newBlockEffectContext(branch string, trxId string, op int, cause string) *StateChangeContext {
	ctx := &StateChangeContext{
		branch: branch,
		trx: -1,
		trxId: trxId,
		op: op,
	}
	ctx.PushCause(cause)
	return ctx
}

func (ctx *StateChangeContext) SetOperation(op int) {
	if ctx == nil {
		return
	}
	ctx.op = op
}

func (ctx *StateChangeContext) SetTrxAndOperation(trxId string, op int) {
	if ctx == nil {
		return
	}
	ctx.trxId, ctx.op = trxId, op
}

func (ctx *StateChangeContext) SetCause(cause string) {
	if ctx == nil {
		return
	}
	ctx.causes = ctx.causes[:0]
	if len(cause) > 0 {
		ctx.causes = append(ctx.causes, cause)
	}
}

func (ctx *StateChangeContext) PushCause(cause string) {
	if ctx == nil {
		return
	}
	if len(cause) > 0 {
		ctx.causes = append(ctx.causes, cause)
	}
}

func (ctx *StateChangeContext) PopCause() {
	if ctx == nil {
		return
	}
	if count := len(ctx.causes); count > 0 {
		ctx.causes = ctx.causes[:count - 1]
	}
}

func (ctx *StateChangeContext) PopAndPushCause(cause string) {
	if ctx == nil {
		return
	}
	ctx.PopCause()
	ctx.PushCause(cause)
}

func (ctx *StateChangeContext) Cause() string {
	if ctx == nil {
		return ""
	}
	return strings.Join(ctx.causes, ".")
}

func (ctx *StateChangeContext) AddChange(seq uint64, what string, change interface{}) {
	if ctx == nil {
		return
	}
	ctx.changes = append(ctx.changes, &internalStateChange{
		StateChange: StateChange{
			Type:        what,
			Transaction: -1,
			Operation:   ctx.op,
			Cause:       ctx.Cause(),
			Change:      change,
		},
		TransactionId: ctx.trxId,
		Sequence: seq,
	})
}

func (ctx *StateChangeContext) ClearChanges() {
	if ctx == nil {
		return
	}
	ctx.changes = ctx.changes[:0]
}

func (ctx *StateChangeContext) Changes() InternalStateChangeSlice {
	if ctx == nil {
		return nil
	}
	return ctx.changes
}

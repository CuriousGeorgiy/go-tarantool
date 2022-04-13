package tarantool_test

import (
	"bytes"
	"errors"
	"testing"

	. "github.com/tarantool/go-tarantool"
	"gopkg.in/vmihailenco/msgpack.v2"
)

const invalidSpaceMsg = "invalid space"
const invalidIndexMsg = "invalid index"

const invalidSpace = 2
const invalidIndex = 2
const validSpace = 1           // Any valid value != default.
const validIndex = 3           // Any valid value != default.
const validExpr = "any string" // We don't check the value here.
const defaultSpace = 0         // And valid too.
const defaultIndex = 0         // And valid too.

type ValidSchemeResolver struct {
}

func (*ValidSchemeResolver) ResolveSpaceIndex(s, i interface{}) (spaceNo, indexNo uint32, err error) {
	if s != nil {
		spaceNo = uint32(s.(int))
	} else {
		spaceNo = defaultSpace
	}
	if i != nil {
		indexNo = uint32(i.(int))
	} else {
		indexNo = defaultIndex
	}
	if spaceNo == invalidSpace {
		return 0, 0, errors.New(invalidSpaceMsg)
	}
	if indexNo == invalidIndex {
		return 0, 0, errors.New(invalidIndexMsg)
	}
	return spaceNo, indexNo, nil
}

var resolver ValidSchemeResolver

func assertBodyFuncCall(t testing.TB, requests []Request, errorMsg string) {
	t.Helper()

	const errBegin = "An unexpected Request.BodyFunc() "
	for _, req := range requests {
		_, err := req.BodyFunc(&resolver)
		if err != nil && errorMsg != "" && err.Error() != errorMsg {
			t.Errorf(errBegin+"error %q expected %q", err.Error(), errorMsg)
		}
		if err != nil && errorMsg == "" {
			t.Errorf(errBegin+"error %q", err.Error())
		}
		if err == nil && errorMsg != "" {
			t.Errorf(errBegin+"result, expexted error %q", errorMsg)
		}
	}
}

func assertBodyEqual(t testing.TB, reference []byte, req Request) {
	t.Helper()

	var reqBuf bytes.Buffer
	reqEnc := msgpack.NewEncoder(&reqBuf)

	f, err := req.BodyFunc(&resolver)
	if err != nil {
		t.Errorf("An unexpected Response.BodyFunc() error: %q", err.Error())
	} else {
		err = f(reqEnc)
		if err != nil {
			t.Errorf("An unexpected encode body error: %q", err.Error())
		}
		reqBody := reqBuf.Bytes()
		if !bytes.Equal(reqBody, reference) {
			t.Errorf("Encoded request %v != reference %v", reqBody, reference)
		}
	}
}

func getTestOps() ([]Op, *Operations) {
	ops := []Op{
		{"+", 1, 2},
		{"-", 3, 4},
		{"&", 5, 6},
		{"|", 7, 8},
		{"^", 9, 1},
		{"^", 9, 1}, // The duplication is for test purposes.
		{":", 2, 3},
		{"!", 4, 5},
		{"#", 6, 7},
		{"=", 8, 9},
	}
	operations := NewOperations().
		Add(1, 2).
		Subtract(3, 4).
		BitwiseAnd(5, 6).
		BitwiseOr(7, 8).
		BitwiseXor(9, 1).
		BitwiseXor(9, 1). // The duplication is for test purposes.
		Splice(2, 3).
		Insert(4, 5).
		Delete(6, 7).
		Assign(8, 9)
	return ops, operations
}

func TestRequestsValidSpaceAndIndex(t *testing.T) {
	requests := []Request{
		NewSelectRequest(validSpace),
		NewSelectRequest(validSpace).Index(validIndex),
		NewUpdateRequest(validSpace),
		NewUpdateRequest(validSpace).Index(validIndex),
		NewUpsertRequest(validSpace),
		NewInsertRequest(validSpace),
		NewReplaceRequest(validSpace),
		NewDeleteRequest(validSpace),
		NewDeleteRequest(validSpace).Index(validIndex),
	}

	assertBodyFuncCall(t, requests, "")
}

func TestRequestsInvalidSpace(t *testing.T) {
	requests := []Request{
		NewSelectRequest(invalidSpace).Index(validIndex),
		NewSelectRequest(invalidSpace),
		NewUpdateRequest(invalidSpace).Index(validIndex),
		NewUpdateRequest(invalidSpace),
		NewUpsertRequest(invalidSpace),
		NewInsertRequest(invalidSpace),
		NewReplaceRequest(invalidSpace),
		NewDeleteRequest(invalidSpace).Index(validIndex),
		NewDeleteRequest(invalidSpace),
	}

	assertBodyFuncCall(t, requests, invalidSpaceMsg)
}

func TestRequestsInvalidIndex(t *testing.T) {
	requests := []Request{
		NewSelectRequest(validSpace).Index(invalidIndex),
		NewUpdateRequest(validSpace).Index(invalidIndex),
		NewDeleteRequest(validSpace).Index(invalidIndex),
	}

	assertBodyFuncCall(t, requests, invalidIndexMsg)
}

func TestRequestsCodes(t *testing.T) {
	tests := []struct {
		req  Request
		code int32
	}{
		{req: NewSelectRequest(validSpace), code: SelectRequestCode},
		{req: NewUpdateRequest(validSpace), code: UpdateRequestCode},
		{req: NewUpsertRequest(validSpace), code: UpsertRequestCode},
		{req: NewInsertRequest(validSpace), code: InsertRequestCode},
		{req: NewReplaceRequest(validSpace), code: ReplaceRequestCode},
		{req: NewDeleteRequest(validSpace), code: DeleteRequestCode},
		{req: NewCall16Request(validExpr), code: Call16RequestCode},
		{req: NewCall17Request(validExpr), code: Call17RequestCode},
		{req: NewEvalRequest(validExpr), code: EvalRequestCode},
		{req: NewExecuteRequest(validExpr), code: ExecuteRequestCode},
		{req: NewPingRequest(), code: PingRequestCode},
	}

	for _, test := range tests {
		if code := test.req.Code(); code != test.code {
			t.Errorf("An invalid request code 0x%x, expected 0x%x", code, test.code)
		}
	}
}

func TestPingRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplPingBody(refEnc)
	if err != nil {
		t.Errorf("An unexpected RefImplPingBody() error: %q", err.Error())
		return
	}

	req := NewPingRequest()
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestSelectRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplSelectBody(refEnc, validSpace, defaultIndex, 0, 0xFFFFFFFF, IterAll, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplSelectBody() error %q", err.Error())
		return
	}

	req := NewSelectRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestSelectRequestDefaultIteratorEqIfKey(t *testing.T) {
	var refBuf bytes.Buffer
	key := []interface{}{uint(18)}

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplSelectBody(refEnc, validSpace, defaultIndex, 0, 0xFFFFFFFF, IterEq, key)
	if err != nil {
		t.Errorf("An unexpected RefImplSelectBody() error %q", err.Error())
		return
	}

	req := NewSelectRequest(validSpace).
		Key(key)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestSelectRequestIteratorNotChangedIfKey(t *testing.T) {
	var refBuf bytes.Buffer
	key := []interface{}{uint(678)}
	const iter = IterGe

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplSelectBody(refEnc, validSpace, defaultIndex, 0, 0xFFFFFFFF, iter, key)
	if err != nil {
		t.Errorf("An unexpected RefImplSelectBody() error %q", err.Error())
		return
	}

	req := NewSelectRequest(validSpace).
		Iterator(iter).
		Key(key)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestSelectRequestSetters(t *testing.T) {
	const offset = 4
	const limit = 5
	const iter = IterLt
	key := []interface{}{uint(36)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplSelectBody(refEnc, validSpace, validIndex, offset, limit, iter, key)
	if err != nil {
		t.Errorf("An unexpected RefImplSelectBody() error %q", err.Error())
		return
	}

	req := NewSelectRequest(validSpace).
		Index(validIndex).
		Offset(offset).
		Limit(limit).
		Iterator(iter).
		Key(key)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestInsertRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplInsertBody(refEnc, validSpace, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplInsertBody() error: %q", err.Error())
		return
	}

	req := NewInsertRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestInsertRequestSetters(t *testing.T) {
	tuple := []interface{}{uint(24)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplInsertBody(refEnc, validSpace, tuple)
	if err != nil {
		t.Errorf("An unexpected RefImplInsertBody() error: %q", err.Error())
		return
	}

	req := NewInsertRequest(validSpace).
		Tuple(tuple)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestReplaceRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplReplaceBody(refEnc, validSpace, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplReplaceBody() error: %q", err.Error())
		return
	}

	req := NewReplaceRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestReplaceRequestSetters(t *testing.T) {
	tuple := []interface{}{uint(99)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplReplaceBody(refEnc, validSpace, tuple)
	if err != nil {
		t.Errorf("An unexpected RefImplReplaceBody() error: %q", err.Error())
		return
	}

	req := NewReplaceRequest(validSpace).
		Tuple(tuple)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestDeleteRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplDeleteBody(refEnc, validSpace, defaultIndex, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplDeleteBody() error: %q", err.Error())
		return
	}

	req := NewDeleteRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestDeleteRequestSetters(t *testing.T) {
	key := []interface{}{uint(923)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplDeleteBody(refEnc, validSpace, validIndex, key)
	if err != nil {
		t.Errorf("An unexpected RefImplDeleteBody() error: %q", err.Error())
		return
	}

	req := NewDeleteRequest(validSpace).
		Index(validIndex).
		Key(key)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestUpdateRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplUpdateBody(refEnc, validSpace, defaultIndex, []interface{}{}, []Op{})
	if err != nil {
		t.Errorf("An unexpected RefImplUpdateBody() error: %q", err.Error())
		return
	}

	req := NewUpdateRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestUpdateRequestSetters(t *testing.T) {
	key := []interface{}{uint(44)}
	refOps, reqOps := getTestOps()
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplUpdateBody(refEnc, validSpace, validIndex, key, refOps)
	if err != nil {
		t.Errorf("An unexpected RefImplUpdateBody() error: %q", err.Error())
		return
	}

	req := NewUpdateRequest(validSpace).
		Index(validIndex).
		Key(key).
		Operations(reqOps)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestUpsertRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplUpsertBody(refEnc, validSpace, []interface{}{}, []Op{})
	if err != nil {
		t.Errorf("An unexpected RefImplUpsertBody() error: %q", err.Error())
		return
	}

	req := NewUpsertRequest(validSpace)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestUpsertRequestSetters(t *testing.T) {
	tuple := []interface{}{uint(64)}
	refOps, reqOps := getTestOps()
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplUpsertBody(refEnc, validSpace, tuple, refOps)
	if err != nil {
		t.Errorf("An unexpected RefImplUpsertBody() error: %q", err.Error())
		return
	}

	req := NewUpsertRequest(validSpace).
		Tuple(tuple).
		Operations(reqOps)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestCallRequestsDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplCallBody(refEnc, validExpr, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplCallBody() error: %q", err.Error())
		return
	}

	req := NewCallRequest(validExpr)
	req16 := NewCall16Request(validExpr)
	req17 := NewCall17Request(validExpr)
	assertBodyEqual(t, refBuf.Bytes(), req)
	assertBodyEqual(t, refBuf.Bytes(), req16)
	assertBodyEqual(t, refBuf.Bytes(), req17)
}

func TestCallRequestsSetters(t *testing.T) {
	args := []interface{}{uint(34)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplCallBody(refEnc, validExpr, args)
	if err != nil {
		t.Errorf("An unexpected RefImplCallBody() error: %q", err.Error())
		return
	}

	req := NewCall16Request(validExpr).
		Args(args)
	req16 := NewCall16Request(validExpr).
		Args(args)
	req17 := NewCall17Request(validExpr).
		Args(args)
	assertBodyEqual(t, refBuf.Bytes(), req)
	assertBodyEqual(t, refBuf.Bytes(), req16)
	assertBodyEqual(t, refBuf.Bytes(), req17)
}

func TestEvalRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplEvalBody(refEnc, validExpr, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplEvalBody() error: %q", err.Error())
		return
	}

	req := NewEvalRequest(validExpr)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestEvalRequestSetters(t *testing.T) {
	args := []interface{}{uint(34), int(12)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplEvalBody(refEnc, validExpr, args)
	if err != nil {
		t.Errorf("An unexpected RefImplEvalBody() error: %q", err.Error())
		return
	}

	req := NewEvalRequest(validExpr).
		Args(args)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestExecuteRequestDefaultValues(t *testing.T) {
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplExecuteBody(refEnc, validExpr, []interface{}{})
	if err != nil {
		t.Errorf("An unexpected RefImplExecuteBody() error: %q", err.Error())
		return
	}

	req := NewExecuteRequest(validExpr)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

func TestExecuteRequestSetters(t *testing.T) {
	args := []interface{}{uint(11)}
	var refBuf bytes.Buffer

	refEnc := msgpack.NewEncoder(&refBuf)
	err := RefImplExecuteBody(refEnc, validExpr, args)
	if err != nil {
		t.Errorf("An unexpected RefImplExecuteBody() error: %q", err.Error())
		return
	}

	req := NewExecuteRequest(validExpr).
		Args(args)
	assertBodyEqual(t, refBuf.Bytes(), req)
}

package rpc

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
)

type InvokeScriptResponse struct {
	responseHeader
	Error  *Error        `json:"error,omitempty"`
	Result *InvokeResult `json:"result,omitempty"`
}

// InvokeResult represents the outcome of a script that is
// executed by the NEO VM.
type InvokeResult struct {
	State       string `json:"state"`
	GasConsumed string `json:"gas_consumed"`
	Script      string `json:"script"`
	Stack       []*StackParam
}

// StackParam respresent a stack parameter.
type StackParam struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (sp *StackParam) UnmarshalJSON(data []byte) error {
	var rsp RawStackParam
	if err := json.Unmarshal(data, &rsp); err != nil {
		return err
	}
	return rsp.Parse(sp)
}

type RawStackParam struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func (rsp *RawStackParam) Parse(sp *StackParam) error {
	switch rsp.Type {
	case "ByteArray":
		var s string
		if err := json.Unmarshal(rsp.Value, &s); err != nil {
			return err
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return err
		}
		sp.Type = rsp.Type
		sp.Value = b
		return nil
	case "String":
		var s string
		if err := json.Unmarshal(rsp.Value, &s); err != nil {
			return err
		}
		sp.Type = rsp.Type
		sp.Value = s
		return nil
	case "Integer":
		var i int64
		err := json.Unmarshal(rsp.Value, &i)
		if err == nil {
			sp.Type = rsp.Type
			sp.Value = i
			return nil
		}

		var s string
		err = json.Unmarshal(rsp.Value, &s)
		if err != nil {
			return err
		}

		i, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		sp.Type = rsp.Type
		sp.Value = i
		return nil
	case "Array":
		//https://github.com/neo-project/neo/blob/3d59ecca5a8deb057bdad94b3028a6d5e25ac088/neo/Network/RPC/RpcServer.cs#L67
		var ar []RawStackParam
		if err := json.Unmarshal(rsp.Value, &ar); err != nil {
			return err
		}
		sp.Type = rsp.Type

		a := make([]StackParam, len(ar))
		for i, x := range ar {
			//	fmt.Printf("LINE %d\n", i)
			//	spew.Dump(x, a[i])
			x.Parse(&a[i])
		}
		sp.Value = a
		return nil
	default:
		return errors.New("not implemented")
	}
}

// AccountStateResponse holds the getaccountstate response.
type AccountStateResponse struct {
	responseHeader
	Result *Account `json:"result"`
}

// Account respresents details about a NEO account.
type Account struct {
	Version    int    `json:"version"`
	ScriptHash string `json:"script_hash"`
	Frozen     bool
	// TODO: need to check this field out.
	Votes    []interface{}
	Balances []*Balance
}

// Balance respresents details about a NEO account balance.
type Balance struct {
	Asset string `json:"asset"`
	Value string `json:"value"`
}

type params struct {
	values []interface{}
}

func newParams(vals ...interface{}) params {
	p := params{}
	p.values = make([]interface{}, len(vals))
	for i := 0; i < len(p.values); i++ {
		p.values[i] = vals[i]
	}
	return p
}

type request struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type responseHeader struct {
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
}

type response struct {
	responseHeader
	Error  *Error      `json:"error"`
	Result interface{} `json:"result"`
}

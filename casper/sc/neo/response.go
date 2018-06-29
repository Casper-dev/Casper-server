package neo

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/CityOfZion/neo-go/pkg/rpc"
)

var errInvalidResult = errors.New("cannot parse contract invocation result")

func (c *Contract) parseScriptResponse(res *rpc.InvokeScriptResponse, vals ...interface{}) error {
	if len(res.Result.Stack) == 0 {
		return errInvalidResult
	}

	arr, ok := res.Result.Stack[0].Value.([]rpc.StackParam)
	if !ok || len(arr) < len(vals) {
		return errInvalidResult
	}

	var err error
	for i, sp := range arr {
		switch v := vals[i].(type) {
		case *string:
			if sp.Type != "ByteArray" {
				return errInvalidResult
			}
			*v = string(sp.Value.([]byte))
		case *[]byte:
			if sp.Type != "ByteArray" {
				return errInvalidResult
			}
			*v = sp.Value.([]byte)
		case *int64:
			*v, err = decodeInteger(sp)
			if err != nil {
				return errInvalidResult
			}
		case *bool:
			t, err := decodeInteger(sp)
			if err != nil {
				return errInvalidResult
			}
			*v = (t != 0)
		default:
			return errInvalidResult
		}
	}

	return nil
}

func decodeInteger(r rpc.StackParam) (ret int64, err error) {
	switch val := r.Value.(type) {
	case int64:
		return val, nil
	case string:
		ret, _ = strconv.ParseInt(val, 10, 64)
		return ret, nil
	case []byte:
		a := &big.Int{}
		b := make([]byte, len(val))
		for i, l := 0, len(b); i < l; i++ {
			b[i] = val[l-i-1]
		}
		a.SetBytes(b)
		return a.Int64(), nil
	default:
		return 0, errInvalidResult
	}
}

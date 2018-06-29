package neo

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/CityOfZion/neo-go/pkg/core/transaction"
	"github.com/CityOfZion/neo-go/pkg/crypto"
	"github.com/CityOfZion/neo-go/pkg/rpc"
	neosc "github.com/CityOfZion/neo-go/pkg/smartcontract"
	"github.com/CityOfZion/neo-go/pkg/util"
	"github.com/davecgh/go-spew/spew"
)

const (
	gasAssetID = "602c79718b16e442de58778e148d0b1084e3b2dffd5de6b7b16cee7969282de7"
	minFee     = util.Fixed8(10000) // 0.0001 GAS
)

func (c *Contract) waitTX(ctx context.Context, txid util.Uint256) error {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			resp, err := c.rpc.GetRawTransaction(txid.String(), true)
			if err != nil {
				return err
			}
			if res, ok := resp.Result.(map[string]interface{}); ok {
				if h, ok := res["blockhash"]; ok {
					fmt.Println("TX block hash:", h)
					return nil
				}
			}
			fmt.Printf("waiting for TX to broadcast...\n")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Contract) performTransaction(res *rpc.InvokeScriptResponse) error {
	script, _ := hex.DecodeString(res.Result.Script)
	t := transaction.NewInvocationTX(script)
	assetID, _ := util.Uint256DecodeString(gasAssetID)
	amount := minFee

	addr, err := c.wif.PrivateKey.Address()
	if err != nil {
		return err
	}

	b, err := c.getBalance(addr)
	if err != nil {
		return err
	}

	inputs, spent := c.calculateInputs(b, amount)
	if len(inputs) == 0 {
		return errors.New("insufficient funds")
	}
	t.Inputs = inputs

	pubkey, _ := c.wif.PrivateKey.PublicKey()
	scriptHash := "21" + hex.EncodeToString(pubkey) + "ac"
	s, _ := hex.DecodeString(scriptHash)
	data, err := util.Uint160FromScript(s)
	t.Attributes = []*transaction.Attribute{{Data: data.Bytes(), Usage: transaction.Script}}

	p, _ := c.wif.PrivateKey.Address()
	bs, _ := crypto.Base58Decode(p)
	hash := hex.EncodeToString(bs[1:21])
	a, err := util.Uint160DecodeString(hash)
	if err != nil {
		return err
	}
	t.AddOutput(&transaction.Output{
		AssetID:    assetID,
		Amount:     spent - amount,
		ScriptHash: a,
	})

	buf := &bytes.Buffer{}
	if err = t.EncodeBinary(buf); err != nil {
		return err
	}

	// TODO fixme remove last '00'
	bb := buf.Bytes()
	signature, err := c.wif.PrivateKey.Sign(bb[:len(bb)-1])
	if err != nil {
		return err
	}
	fmt.Printf("sign: %x\n", signature)

	invocS, _ := hex.DecodeString("40" + hex.EncodeToString(signature))
	verifS, _ := hex.DecodeString(scriptHash)
	t.Scripts = []*transaction.Witness{{invocS, verifS}}
	t.Hash()
	spew.Dump(t)

	buf = &bytes.Buffer{}
	if err = t.EncodeBinary(buf); err != nil {
		return err
	}

	rawTx := hex.EncodeToString(buf.Bytes())
	resp, err := c.rpc.SendRawTransaction(rawTx)
	if err != nil {
		return err
	}

	spew.Dump(resp)

	accepted, ok := resp.Result.(bool)
	if ok && accepted {
		ctx := context.Background()
		//nc, ca := context.WithTimeout(ctx, time.Second*20)
		//defer ca()
		return c.waitTX(ctx, t.Hash())
	}
	println(ok, accepted)
	return errors.New("invalid transaction")
}

func (c *Contract) callContractMethod(method string, args ...interface{}) (*rpc.InvokeScriptResponse, error) {
	res, err := c.rpc.InvokeFunction(c.contract, method, newParams(args...))
	spew.Dump(res)
	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, fmt.Errorf("error: %s", res.Error.Error())
	}
	if res.Result == nil {
		return nil, errors.New("empty result")
	}
	if res.Result.State != "HALT, BREAK" {
		return nil, errors.New("contract invocation failed: " + res.Result.State)
	}

	return res, nil
}

// UTXO represents unspent transaction output
type UTXO struct {
	Index uint16       `json:"index"`
	TXID  util.Uint256 `json:"txid"`
	Value util.Fixed8  `json:"value"`
}

// AssetInfo represents state of particular asset
type AssetInfo struct {
	Balance util.Fixed8 `json:"balance"`
	Unspent []UTXO      `json:"unspent"`
}

// Balance represents state of the wallet
type Balance struct {
	GAS     AssetInfo `json:"GAS,omitempty"`
	NEO     AssetInfo `json:"NEO,omitempty"`
	Address string    `json:"address"`
	Net     string    `json:"net"`
}

// TODO: we use NEON DB here only to get UTXO
// It is, probably, also possible with more widespread solutions
// e.g. NeoScan https://neoscan.io/doc/Neoscan.Api.html
func (c *Contract) getBalance(address string) (*Balance, error) {
	apiURL := &url.URL{Scheme: "http", Host: c.neonAPI, Path: "/v2/address/balance"}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", apiURL.String(), address), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json-rpc")

	hc := http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var balance Balance
	err = json.NewDecoder(resp.Body).Decode(&balance)
	spew.Dump(balance)
	return &balance, err
}

func (c *Contract) calculateInputs(balances *Balance, gasCost util.Fixed8) ([]*transaction.Input, util.Fixed8) {
	// TODO add 'intents' argument
	required := gasCost
	//assets := map[string]util.Fixed8{gasAssetID: gasCost}
	sort.Slice(balances.GAS.Unspent, func(i, j int) bool {
		return balances.GAS.Unspent[i].Value > balances.GAS.Unspent[j].Value
	})

	selected := util.Fixed8(0)
	num := uint16(0)
	for _, us := range balances.GAS.Unspent {
		if selected >= required {
			break
		}
		selected += us.Value
		num++
	}
	if selected < required {
		return nil, util.Fixed8(0)
	}
	fmt.Printf("selected balances: %s\n", selected)

	inputs := make([]*transaction.Input, num)
	for i := uint16(0); i < num; i++ {
		inputs[i] = &transaction.Input{
			PrevHash:  balances.GAS.Unspent[i].TXID,
			PrevIndex: balances.GAS.Unspent[i].Index,
		}
	}
	return inputs, selected
}

func newParams(vals ...interface{}) []neosc.Parameter {
	params := make([]neosc.Parameter, len(vals))
	spew.Dump(vals)
	for i, val := range vals {
		switch val.(type) {
		case string:
			params[i] = neosc.Parameter{Type: neosc.StringType, Value: val}
		case int, int16, int32, int64:
			params[i] = neosc.Parameter{Type: neosc.IntegerType, Value: val}
		default:
			spew.Dump(val)
			panic("unexpected type")
		}
	}
	spew.Dump(params)
	return params
}

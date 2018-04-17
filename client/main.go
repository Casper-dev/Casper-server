package client

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"context"
	"encoding/json"
	_ "expvar"
	"fmt"
	_ "net/http/pprof"

	thrift "gitlab.com/casperDev/Casper-server/casper/thrift"

	"gitlab.com/casperDev/Casper-SC/casper_sc"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"

	"github.com/ethereum/go-ethereum/core/types"
)

var log = logging.Logger("client/handler")

func recoverHandler(name string, args ...interface{}) {
	if r := recover(); r != nil {
		fmt.Printf("recovered in %s(%s): %s\n", name, args, r)
	}
}

func HandleClientUpload(ctx context.Context, ip string, hash string, size int64, ipList []string) (err error) {
	log.Infof("started upload(%s, %s, %d)", ip, hash, size)
	defer recoverHandler("upload", ip, hash, size)

	val, err := json.Marshal(ipList)
	if err != nil {
		return err
	}

	_, err = thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.SendUploadQuery(ctx, hash, string(val), size)
	})

	fmt.Println("upload()")
	return err
}

func HandleClientDownload(ctx context.Context, ip string, hash string, wallet string) (err error) {
	log.Infof("started download(%s, %s, %s)", ip, hash, wallet)
	defer recoverHandler("download", ip, hash, wallet)

	_, err = thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.SendDownloadQuery(ctx, hash, "2", wallet)
	})
	if err != nil {
		return err
	}

	casper, sclient, auth, _ := Casper_SC.GetSC()
	fmt.Println("Got SC")
	//TODO: Change ConfirmDownload call location
	Casper_SC.ValidateMineTX(func() (tx *types.Transaction, err error) {
		return casper.ConfirmDownload(auth)
	}, sclient, auth)

	fmt.Println("download()")
	return nil
}

func HandleClientDelete(ctx context.Context, ip string, hash string) (err error) {
	log.Infof("started delete(%s, %s)", ip, hash)
	defer recoverHandler("delete", ip, hash)

	_, err = thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.SendDeleteQuery(ctx, hash)
	})
	if err != nil {
		return err
	}

	fmt.Println("delete()")
	return err
}

func HandleClientUpdate(ctx context.Context, ip string, uuid string, hash string, size int64) (err error) {
	log.Infof("started update(%s, %s, %s)", ip, uuid, hash)
	defer recoverHandler("update", ip, uuid, hash, size)

	h, err := thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.SendUpdateQuery(ctx, uuid, hash, size)
	})
	if err != nil {
		return err
	}

	log.Infof("finished update() with %s", h.(string))
	return nil
}

func InvokeGetFileChecksum(ctx context.Context, ip string, uuid string, first, last int64, salt string) (string, error) {
	log.Infof("started GetFileChecksum(%s, %d, %d, %s)", uuid, first, last, salt)
	defer recoverHandler("GetFileChecksum", uuid, first, last, salt)

	h, err := thrift.RunClientClosure(ip, func(c *thrift.ThriftClient) (interface{}, error) {
		return c.GetFileChecksum(ctx, uuid, first, last, salt)
	})

	log.Infof("Hash: %s, Error: %s", h, err)

	return h.(string), err
}

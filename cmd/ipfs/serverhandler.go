package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"io"
	"math/big"
	"github.com/Casper-dev/Casper-server/exchange/bitswap/decision"

	"github.com/Casper-dev/Casper-server/core/commands"
	"github.com/Casper-dev/Casper-SC/casper_sc"
	"github.com/Casper-dev/Casper-server/casper/casper_utils"
)

type CasperServerHandler struct {
}

func NewCasperServerHandler() *CasperServerHandler {
	return &CasperServerHandler{}
}

func (serverHandler *CasperServerHandler) Ping(ctx context.Context) (int64, error) {
	return int64(time.Now().Unix()), nil
}

func (serverHandler *CasperServerHandler) SendUploadQuery(ctx context.Context, hash string, ipAddr string, size int64) (status string, err error) {
	fmt.Println(hash + " " + ipAddr);
	status, err = runCommand(ctx, []string{"files", "cp", "/ipfs/" + hash, "/"})
	status, err = runCommand(ctx, []string{"pin", "add", hash})
	status, err = runCommand(ctx, []string{"cat", "/ipfs/" + hash})
	if err == nil {
		fmt.Println("no error");
	}
	fmt.Println("Running events")
	//intSize, err := strconv.ParseInt(size, 10, 64)
	fmt.Println(err)

	fmt.Println("Waiting to get SC")
	casper, client, auth := Casper_SC.GetSC()
	tx, err := casper.ConfirmUpload(auth, casper_utils.GetCasperNodeID(), big.NewInt(size))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Got SC")
	casper_utils.MineTX(tx, client)

	return
}

func (serverHandler *CasperServerHandler) SendDownloadQuery(ctx context.Context, hash string, ipAddr string, wallet string) (status string, err error) {
	fmt.Println(hash + " " + ipAddr)
	decision.AllowedHashes = hash
	decision.Wallet = wallet
	///TODO: reimplement this in a more sane way
	///as it is now, this impl cannot serve more than one connection
	fmt.Println("got download request")
	return
}

func (serverHandler *CasperServerHandler) SendDeleteQuery(ctx context.Context, hash string) (status string, err error) {
	fmt.Println(hash)
	///TODO: look closely if we need to reimplement this without runCommand
	runCommand(ctx, []string{"ls", hash})
	status, err = runCommand(ctx, []string{"pin", "rm", hash})
	status, err = runCommand(ctx, []string{"files", "rm", "/" + hash})

	casper, client, auth := Casper_SC.GetSC()
	size := int64(commands.SizeOut)
	tx, err := casper.NotifySpaceFreed(auth, casper_utils.GetCasperNodeID(), big.NewInt(size))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Got SC")
	casper_utils.MineTX(tx, client)
	if err == nil {
		fmt.Println("no error");
	}
	return
}

func runCommand(ctx context.Context, args []string) (status string, err error) {
	var invoc cmdInvocation
	defer invoc.close()

	// parse the commandline into a command invocation
	parseErr := invoc.Parse(ctx, args)

	// ok now handle parse error (which means cli input was wrong,
	// e.g. incorrect number of args, or nonexistent subcommand)
	if parseErr != nil {
		fmt.Println(parseErr)

		// this was a user error, print help.
		if invoc.cmd != nil {
			// we need a newline space.
			fmt.Fprintf(os.Stderr, "\n")
			//printHelp(false, os.Stderr)
		}
		return "smells like ebola", parseErr
	}

	// ok, finally, run the command invocation.
	intrh, ctx := invoc.SetupInterruptHandler(ctx)
	defer intrh.Close()
	output, err := invoc.Run(ctx)
	if err != nil {
		fmt.Println(err)
		return "smells like ebola", err
	}

	// everything went better than expected :)

	_, err = io.Copy(os.Stdout, output)
	if err != nil {
		fmt.Println(err)

		// if this error was a client error, print short help too.
		if isClientError(err) {
			//printMetaHelp(os.Stderr)
		}
		return "smells like ebola", err
	}

	return "Dis is da wei", nil
}

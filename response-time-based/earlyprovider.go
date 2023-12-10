package main

import (
	"context"
	"math/rand"
	
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
)

func runEarlyProvide(ctx context.Context, runenv *runtime.RunEnv, h host.Host, bstore blockstore.Blockstore, ex exchange.Interface, initCtx *run.InitContext, r *rand.Rand) error {
	client := initCtx.SyncClient

	ai := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
	client.MustPublish(ctx, providerTopic, &ai)
	_ = client.MustSignalAndWait(ctx, earlyProviderReadyState, runenv.TestInstanceCount)

	size := runenv.SizeParam("size")
	count := runenv.IntParam("count")

	rootBlock := generateBlocksOfSize(1, size, r)
	blocks := generateBlocksOfSize(count, size, r)
	blocks[0] = rootBlock[0]

	if err := bstore.PutMany(ctx, blocks); err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		runenv.RecordMessage("Published block #%s", blocks[i].Cid())
	}

	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount)
	_ = client.MustSignalAndWait(ctx, doneState, runenv.TestInstanceCount)
	return nil
}

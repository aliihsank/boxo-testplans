package main

import (
	"context"
	"math/rand"
	
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	
	block "github.com/ipfs/go-block-format"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
)

func runLateProvide(ctx context.Context, runenv *runtime.RunEnv, h host.Host, bstore blockstore.Blockstore, ex exchange.Interface, initCtx *run.InitContext) error {
	client := initCtx.SyncClient

	ai := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
	client.MustPublish(ctx, providerTopic, &ai)
	_ = client.MustSignalAndWait(ctx, readyLateProviderState, runenv.TestInstanceCount)

	size := runenv.SizeParam("size")
	count := runenv.IntParam("count")
	for i := 0; i <= count; i++ {
		runenv.RecordMessage("generating %d-sized random block", size)
		buf := make([]byte, size)
		rand.Read(buf)
		blk := block.NewBlock(buf)
		err := bstore.Put(ctx, blk)
		if err != nil {
			return err
		}
		mh := blk.Multihash()
		runenv.RecordMessage("publishing block %s", mh.String())
		client.MustPublish(ctx, blockTopic, &mh)
	}
	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount)
	_ = client.MustSignalAndWait(ctx, doneState, runenv.TestInstanceCount)
	return nil
}

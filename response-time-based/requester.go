package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/ipfs/go-cid"
	
	bstats "github.com/ipfs/go-ipfs-regression/bitswap"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
)

func runRequest(ctx context.Context, runenv *runtime.RunEnv, h host.Host, bstore blockstore.Blockstore, ex exchange.Interface, initCtx *run.InitContext) error {
	client := initCtx.SyncClient

	providers := make(chan *peer.AddrInfo)
	blkmhs := make(chan *multihash.Multihash)
	providerSub, err := client.Subscribe(ctx, providerTopic, providers)
	if err != nil {
		return err
	}
	ai := <-providers

	runenv.RecordMessage("connecting  to provider provider: %s", fmt.Sprint(*ai))
	providerSub.Done()

	err = h.Connect(ctx, *ai)
	if err != nil {
		return fmt.Errorf("could not connect to provider: %w", err)
	}

	runenv.RecordMessage("connected to provider")

	// tell the provider that we're ready for it to publish blocks
	_ = client.MustSignalAndWait(ctx, readyEarlyProviderState, runenv.TestInstanceCount)
	// wait until the provider is ready for us to start downloading
	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount)

	blockmhSub, err := client.Subscribe(ctx, blockTopic, blkmhs)
	if err != nil {
		return fmt.Errorf("could not subscribe to block sub: %w", err)
	}
	defer blockmhSub.Done()

	begin := time.Now()
	count := runenv.IntParam("count")
	for i := 0; i <= count; i++ {
		mh := <-blkmhs
		runenv.RecordMessage("downloading block %s", mh.String())
		dlBegin := time.Now()
		blk, err := ex.GetBlock(ctx, cid.NewCidV0(*mh))
		if err != nil {
			return fmt.Errorf("could not download get block %s: %w", mh.String(), err)
		}
		dlDuration := time.Since(dlBegin)
		s := &bstats.BitswapStat{
			SingleDownloadSpeed: &bstats.SingleDownloadSpeed{
				Cid:              blk.Cid().String(),
				DownloadDuration: dlDuration,
			},
		}
		runenv.RecordMessage(bstats.Marshal(s))
	}
	duration := time.Since(begin)
	s := &bstats.BitswapStat{
		MultipleDownloadSpeed: &bstats.MultipleDownloadSpeed{
			BlockCount:    count,
			TotalDuration: duration,
		},
	}
	runenv.RecordMessage(bstats.Marshal(s))
	_ = client.MustSignalEntry(ctx, doneState)
	return nil
}

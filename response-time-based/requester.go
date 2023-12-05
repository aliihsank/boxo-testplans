package main

import (
	"context"
	"fmt"
	"time"
	"math/rand"
	
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/ipfs/go-cid"
	
	multihash "github.com/multiformats/go-multihash"
	bstats "github.com/ipfs/go-ipfs-regression/bitswap"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/boxo/exchange"
)

func runRequest(ctx context.Context, runenv *runtime.RunEnv, h host.Host, bstore blockstore.Blockstore, ex exchange.SessionExchange, initCtx *run.InitContext, r *rand.Rand) error {
	client := initCtx.SyncClient

	providers := make(chan *peer.AddrInfo)
	providerSub, err := client.Subscribe(ctx, providerTopic, providers)
	if err != nil {
		return err
	}

	providerSub.Done()
	
	providerCount := runenv.IntParam("provider_count")
	for i := 0; i <= providerCount - 1; i++ {
		ai := <-providers
		runenv.RecordMessage("connecting to provider provider[%d]: %s", i, fmt.Sprint(*ai))

		err = h.Connect(ctx, *ai)
		if err != nil {
			return fmt.Errorf("could not connect to provider: %w", err)
		}
		runenv.RecordMessage("requester connected to provider[%d]: %s", i, fmt.Sprint(*ai))
	}
	
	runenv.RecordMessage("connected to all providers")

	// tell the provider that we're ready for it to publish blocks
	_ = client.MustSignalAndWait(ctx, earlyProviderReadyState, runenv.TestInstanceCount)
		
	// generate same blocks to get multihashes
	size := runenv.SizeParam("size")
	count := runenv.IntParam("count")

	blkmhs := make([]multihash.Multihash, size)
	rootBlock := generateBlocksOfSize(1, size, r)
	blocks := generateBlocksOfSize(count, size, r)
	blocks[0] = rootBlock[0]

	for i := 0; i < count; i++ {
		blkmhs[i] = blocks[i].Cid().Hash()
		runenv.RecordMessage("[Block Cid: ", blocks[i].Cid(), "], [Hash: ", blocks[i].Cid().Hash(), "]")
	}
	
	// wait until the provider is ready for us to start downloading
	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount)

	session := ex.NewSession(ctx)

	begin := time.Now()
	for i := 0; i < count; i++ {
		mh := blkmhs[i]
		runenv.RecordMessage("Downloading block %s", mh.String())
		dlBegin := time.Now()

		blk, err := session.GetBlock(ctx, cid.NewCidV0(mh))
		if err != nil {
			return fmt.Errorf("Could not download get block %s: %w", mh.String(), err)
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


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
		runenv.RecordMessage("Connecting to provider[%d]: %s", i, fmt.Sprint(*ai))

		err = h.Connect(ctx, *ai)
		if err != nil {
			return fmt.Errorf("Could not connect to provider: %w", err)
		}
		runenv.RecordMessage("Requester connected to provider[%d]: %s", i, fmt.Sprint(*ai))
	}
	
	runenv.RecordMessage("Connected to all providers")

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
	}
	
	// wait until the provider is ready for us to start downloading
	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount)

	session := ex.NewSession(ctx)

	begin := time.Now()
	for i := 0; i < count; i++ {
		mh := blkmhs[i]
		blockCid := cid.NewCidV0(mh)
		runenv.RecordMessage("Initiating block request for %s", blockCid.String())

		dlBegin := time.Now()

		fmt.Println("------------------------------BLOCK ", (i + 1), "------------------------------")
		blk, err := session.GetBlock(ctx, blockCid)
		if err != nil {
			return fmt.Errorf("Could not download/get block %s: %w", mh.String(), err)
		}
		dlDuration := time.Since(dlBegin) / 1e6
		s := &bstats.BitswapStat{
			SingleDownloadSpeed: &bstats.SingleDownloadSpeed{
				Cid:              blk.Cid().String(),
				DownloadDuration: dlDuration,
			},
		}
		runenv.RecordMessage(bstats.Marshal(s))
	}

	duration := time.Since(begin) / 1e6
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

func runRequestForCase2(ctx context.Context, runenv *runtime.RunEnv, h host.Host, bstore blockstore.Blockstore, ex exchange.SessionExchange, initCtx *run.InitContext, r *rand.Rand) error {
	client := initCtx.SyncClient

	providers := make(chan *peer.AddrInfo)
	providerSub, err := client.Subscribe(ctx, providerTopic, providers)
	if err != nil {
		return err
	}

	providerSub.Done()
	
	providerCount := runenv.IntParam("provider_count")
	for i := 0; i <= providerCount - 2; i++ {
		ai := <-providers
		runenv.RecordMessage("Connecting to provider[%d]: %s", i, fmt.Sprint(*ai))

		err = h.Connect(ctx, *ai)
		if err != nil {
			return fmt.Errorf("Could not connect to provider: %w", err)
		}
		runenv.RecordMessage("Requester connected to provider[%d]: %s", i, fmt.Sprint(*ai))
	}
	
	runenv.RecordMessage("Connected to all early providers")

	// tell the provider that we're ready for it to publish blocks
	_ = client.MustSignalAndWait(ctx, earlyProviderReadyState, runenv.TestInstanceCount - 1)
		
	// generate same blocks to get multihashes
	size := runenv.SizeParam("size")
	count := runenv.IntParam("count")

	blkmhs := make([]multihash.Multihash, size)
	rootBlock := generateBlocksOfSize(1, size, r)
	blocks := generateBlocksOfSize(count, size, r)
	blocks[0] = rootBlock[0]

	for i := 0; i < count; i++ {
		blkmhs[i] = blocks[i].Cid().Hash()
	}
	
	// wait until the provider is ready for us to start downloading
	_ = client.MustSignalAndWait(ctx, readyDLState, runenv.TestInstanceCount - 1)

	session := ex.NewSession(ctx)

	begin := time.Now()

	// PHASE 1
	runenv.RecordMessage("Starting Phase 1")
	for i := 0; i < count / 2; i++ {
		mh := blkmhs[i]
		blockCid := cid.NewCidV0(mh)

		dlBegin := time.Now()

		fmt.Println("------------------------------BLOCK ", (i + 1), "------------------------------")
		blk, err := session.GetBlock(ctx, blockCid)
		if err != nil {
			return fmt.Errorf("Could not download/get block %s: %w", mh.String(), err)
		}
		dlDuration := time.Since(dlBegin) / 1e6
		s := &bstats.BitswapStat{
			SingleDownloadSpeed: &bstats.SingleDownloadSpeed{
				Cid:              blk.Cid().String(),
				DownloadDuration: dlDuration,
			},
		}
		runenv.RecordMessage(bstats.Marshal(s))
	}

	duration := time.Since(begin) / 1e6
	s := &bstats.BitswapStat{
		MultipleDownloadSpeed: &bstats.MultipleDownloadSpeed{
			BlockCount:    count / 2,
			TotalDuration: duration,
		},
	}
	runenv.RecordMessage("Finished Phase 1")

	runenv.RecordMessage(bstats.Marshal(s))
	
	_ = client.MustSignalAndWait(ctx, phase2BeginState, 2) // Wait for 1 late provider to send its info
	_ = client.MustSignalAndWait(ctx, lateProviderReadyState, 2) // Wait for 1 late provider to send its info

	lateProviders := make(chan *peer.AddrInfo)
	lateProviderSub, lateErr := client.Subscribe(ctx, lateProviderTopic, lateProviders)
	if lateErr != nil {
		return lateErr
	}

	_ = client.MustSignalAndWait(ctx, readyDLPhase2State, 2) // Wait for 1 late provider to be ready

	lateProviderSub.Done()
	ai := <-lateProviders
	runenv.RecordMessage("Connecting to late provider: %s", fmt.Sprint(*ai))
	err = h.Connect(ctx, *ai)
	if err != nil {
		return fmt.Errorf("Could not connect to provider: %w", err)
	}
	runenv.RecordMessage("Requester connected to late provider: %s", fmt.Sprint(*ai))

	time.Sleep(20 * time.Second)
	
	begin = time.Now()

	// PHASE 2
	runenv.RecordMessage("Starting Phase 2")
	for i := count / 2; i < count; i++ {
		mh := blkmhs[i]
		blockCid := cid.NewCidV0(mh)

		dlBegin := time.Now()
		
		fmt.Println("---------------BLOCK ", (i + 1), "---------------")
		blk, err := session.GetBlock(ctx, blockCid)
		if err != nil {
			return fmt.Errorf("Could not download/get block %s: %w", mh.String(), err)
		}
		dlDuration := time.Since(dlBegin) / 1e6
		s := &bstats.BitswapStat{
			SingleDownloadSpeed: &bstats.SingleDownloadSpeed{
				Cid:              blk.Cid().String(),
				DownloadDuration: dlDuration,
			},
		}
		runenv.RecordMessage(bstats.Marshal(s))
	}
	runenv.RecordMessage("Finished Phase 2")

	duration = time.Since(begin) / 1e6
	s = &bstats.BitswapStat{
		MultipleDownloadSpeed: &bstats.MultipleDownloadSpeed{
			BlockCount:    count - (count / 2),
			TotalDuration: duration,
		},
	}

	runenv.RecordMessage(bstats.Marshal(s))
	_ = client.MustSignalEntry(ctx, doneState)
	return nil
}



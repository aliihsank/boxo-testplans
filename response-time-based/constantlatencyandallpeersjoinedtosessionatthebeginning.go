package main

import (
	"context"
	"fmt"
	"errors"
	
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p"
	
	dht "github.com/libp2p/go-libp2p-kad-dht"
	datastore "github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	bitswap "github.com/ipfs/boxo/bitswap"
	bsnet "github.com/ipfs/boxo/bitswap/network"
)

func runConstantLatencyAndAllPeersJoinedToSessionAtTheBeginning(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("Running Case: runConstantLatencyAndAllPeersJoinedToSessionAtTheBeginning")
	ctx := context.Background()

	netclient := initCtx.NetClient

	linkShape := network.LinkShape{}
	// linkShape := network.LinkShape{
	// 	Latency:   50 * time.Millisecond,
	// 	Jitter:    20 * time.Millisecond,
	// 	Bandwidth: 3e6,
	// 	// Filter: (not implemented)
	// 	Loss:          0.02,
	// 	Corrupt:       0.01,
	// 	CorruptCorr:   0.1,
	// 	Reorder:       0.01,
	// 	ReorderCorr:   0.1,
	// 	Duplicate:     0.02,
	// 	DuplicateCorr: 0.1,
	// }
	netclient.MustConfigureNetwork(ctx, &network.Config{
		Network:        "default",
		Enable:         true,
		Default:        linkShape,
		CallbackState:  networkState,
		CallbackTarget: runenv.TestGroupInstanceCount,
		RoutingPolicy:  network.AllowAll,
	})

	fmt.Println("Configured Network")

	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/3333", netclient.MustGetDataNetworkIP().String()))
	if err != nil {
		return err
	}

	fmt.Println("Listen: ", listen)

	h, err := libp2p.New(libp2p.ListenAddrs(listen))
	if err != nil {
		return err
	}
	
	fmt.Println("Libp2p Host: ", h)

	defer h.Close()
	kad, err := dht.New(ctx, h)
	if err != nil {
		return err
	}
	for _, a := range h.Addrs() {
		runenv.RecordMessage("Listening on Addr: %s", a.String())
	}
	bstore := blockstore.NewBlockstore(datastore.NewMapDatastore())
	ex := bitswap.New(ctx, bsnet.NewFromIpfsHost(h, kad), bstore)
	switch runenv.TestGroupID {
	case "early_provider":
		runenv.RecordMessage("Running new early_provider")
		err = runEarlyProvide(ctx, runenv, h, bstore, ex, initCtx)
	case "late_provider":
		runenv.RecordMessage("Running new late_provider")
		err = runLateProvide(ctx, runenv, h, bstore, ex, initCtx)
	case "requester":
		runenv.RecordMessage("Running new requester")
		err = runRequest(ctx, runenv, h, bstore, ex, initCtx)
	default:
		runenv.RecordMessage("Not part of a group: ", runenv.TestGroupID)
		err = errors.New("Unknown test group id")
	}
	return err
}
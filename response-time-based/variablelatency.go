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

func runVariableLatency(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("running speed-test")
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
	listen, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/3333", netclient.MustGetDataNetworkIP().String()))
	if err != nil {
		return err
	}
	h, err := libp2p.New(libp2p.ListenAddrs(listen))
	if err != nil {
		return err
	}
	defer h.Close()
	kad, err := dht.New(ctx, h)
	if err != nil {
		return err
	}
	for _, a := range h.Addrs() {
		runenv.RecordMessage("listening on addr: %s", a.String())
	}
	bstore := blockstore.NewBlockstore(datastore.NewMapDatastore())
	ex := bitswap.New(ctx, bsnet.NewFromIpfsHost(h, kad), bstore)
	switch runenv.TestGroupID {
	case "early_providers":
		runenv.RecordMessage("running early_providers")
		err = runEarlyProvide(ctx, runenv, h, bstore, ex, initCtx)
	case "late_providers":
		runenv.RecordMessage("running late_providers")
		err = runLateProvide(ctx, runenv, h, bstore, ex, initCtx)
	case "requesters":
		runenv.RecordMessage("running requester")
		err = runRequest(ctx, runenv, h, bstore, ex, initCtx)
	default:
		runenv.RecordMessage("not part of a group")
		err = errors.New("unknown test group id")
	}
	return err
}
package main

import (
	"context"
	"fmt"
	"errors"
	"math/rand"
	"time"
	
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

func runConstantLatencyAndThirdPeerJoinedToSessionAfter500Blocks(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	runenv.RecordMessage("Running Case: runConstantLatencyAndThirdPeerJoinedToSessionAfter500Blocks")
	ctx := context.Background()

	netclient := initCtx.NetClient
	groupSeq := initCtx.GroupSeq

	runenv.RecordMessage("Group Seq: ", groupSeq)

	linkShape := network.LinkShape{}

	if(runenv.TestGroupID == "early_provider"){
		switch groupSeq {
		case 1:
			linkShape = network.LinkShape{
				Latency:   20 * time.Millisecond,
				Jitter:    20 * time.Millisecond,
				Bandwidth: 3e6,
			}
		case 2:
			linkShape = network.LinkShape{
				Latency:   60 * time.Millisecond,
				Jitter:    20 * time.Millisecond,
				Bandwidth: 1e6,
			}
		default:
			fmt.Println("There is something wrong with seq number.")
		}
	}

	if(runenv.TestGroupID == "late_provider"){
		switch groupSeq {
		case 1:
			linkShape = network.LinkShape{
				Latency:   10 * time.Millisecond,
				Jitter:    20 * time.Millisecond,
				Bandwidth: 6e6,
			}
		default:
			fmt.Println("There is something wrong with seq number.")
		}
	}

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
		runenv.RecordMessage("Listening on Addr: %s", a.String())
	}
	bstore := blockstore.NewBlockstore(datastore.NewMapDatastore())
	ex := bitswap.New(ctx, bsnet.NewFromIpfsHost(h, kad), bstore)

	switch runenv.TestGroupID {
	case "early_provider":
		r := rand.New(rand.NewSource(5))
		runenv.RecordMessage("Running new early_provider, Random Seed Test: ", r.Uint64())
		err = runEarlyProvideForCase2(ctx, runenv, h, bstore, ex, initCtx, r)
	case "late_provider":
		r := rand.New(rand.NewSource(5))
		runenv.RecordMessage("Running new late_provider, Random Seed Test: ", r.Uint64())
		err = runLateProvide(ctx, runenv, h, bstore, ex, initCtx, r)
	case "requester":
		r := rand.New(rand.NewSource(5))
		runenv.RecordMessage("Running new requester, Random Seed Test: ", r.Uint64())
		err = runRequestForCase2(ctx, runenv, h, bstore, ex, initCtx, r)
	default:
		runenv.RecordMessage("Not part of a group: ", runenv.TestGroupID)
		err = errors.New("Unknown test group id")
	}
	return err
}
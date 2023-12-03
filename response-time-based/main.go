package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"

	bitswap "github.com/ipfs/go-libipfs/bitswap"
	bsnet "github.com/ipfs/go-libipfs/bitswap/network"
	block "github.com/ipfs/go-libipfs/blocks"
	"github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	exchange "github.com/ipfs/go-ipfs-exchange-interface"
	bstats "github.com/ipfs/go-ipfs-regression/bitswap"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

var (
	testcases = map[string]interface{}{
		"constant-latency_all-peers-joined-to-session-at-the-beginning": run.InitializedTestCaseFn(runConstantLatencyAndAllPeersJoinedToSessionAtTheBeginning),
		"constant-latency_third-peer-joined-to-session-after-500-blocks": run.InitializedTestCaseFn(runConstantLatencyAndThirdPeerJoinedToSessionAfter500Blocks),
		"variable-latency": run.InitializedTestCaseFn(runVariableLatency),
	}
	networkState  = sync.State("network-configured")
	readyEarlyProviderState    = sync.State("early-provider-ready-to-publish")
	readyLateProviderState    = sync.State("late-provider-ready-to-publish")
	readyDLState  = sync.State("ready-to-download")
	doneState     = sync.State("done")
	providerTopic = sync.NewTopic("provider", &peer.AddrInfo{})
	blockTopic    = sync.NewTopic("blocks", &multihash.Multihash{})
)

func main() {
	run.InvokeMap(testcases)
}

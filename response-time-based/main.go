package main

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

var (
	testcases = map[string]interface{}{
		"constant-latency_all-peers-joined-to-session-at-the-beginning": run.InitializedTestCaseFn(runConstantLatencyAndAllPeersJoinedToSessionAtTheBeginning), // case 1
		"constant-latency_third-peer-joined-to-session-after-500-blocks": run.InitializedTestCaseFn(runConstantLatencyAndThirdPeerJoinedToSessionAfter500Blocks), // case 2
		"variable-latency": run.InitializedTestCaseFn(runVariableLatency), // case 3
	}
	networkState  = sync.State("network-configured")
	earlyProviderReadyState    = sync.State("early-provider-ready-to-publish")
	lateProviderReadyState    = sync.State("late-provider-ready-to-publish")
	readyDLState  = sync.State("ready-to-download")
	doneState     = sync.State("done")
	providerTopic = sync.NewTopic("provider", &peer.AddrInfo{})
	blockTopic    = sync.NewTopic("blocks", &multihash.Multihash{})
)

func main() {
	run.InvokeMap(testcases)
}

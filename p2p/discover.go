/**
 * File        : discover.go
 * Description : Peer discovery module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"math"
	"math/rand"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

// Discover peers.
func (client *client) discoverPeers() func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	// Replenish the routing table.
	go func() {
		rate := math.Log(120) / 30
		then := time.Now()
	Discovery:
		for {
			select {
			case <-notify:
				break Discovery
			default:
				if time.Since(then) < 30*time.Second {
					time.Sleep(time.Second * time.Duration(math.Exp(rate*time.Since(then).Seconds())))
				} else {
					time.Sleep(120 * time.Second)
				}
				client.replenishRoutingTable(client.config.SampleSize)
			}
		}
	}()

	// Return the shutdown function.
	return shutdown

}

// Replenish the routing table.
func (client *client) replenishRoutingTable(queries int) {

	// Randomize the routing table.
	peers := client.table.ListPeers()
	random := rand.Perm(len(peers))

	// Sample random peers from the routing table.
	for i := 0; i < len(random) && queries > 0; i++ {

		// Prevent self sampling.
		if peers[random[i]] == client.id {
			continue
		}

		// Check if contact info exists for the peer.
		info := client.peerstore.PeerInfo(peers[random[i]])
		if len(info.Addrs) != 0 {

			// Get a random sample of peers from the peer.
			sample, err := client.sample(peers[random[i]])
			if err != nil {
				continue
			}

			// Add peers from the random sample.
			for j := 0; j < len(sample); j++ {

				// Prevent the client from adding a peer with no address.
				if len(sample[j].Addrs) == 0 {
					continue
				}

				// Prevent the client from repeatedly adding a peer.
				timestamp, err := client.peerstore.Get(sample[j].ID, "LAST_PING")
				if err == nil && time.Since(timestamp.(time.Time)) < 5 * time.Minute {
					continue
				}

				// Temporarily add the peer to the peer store.
				client.peerstore.AddAddrs(
					sample[j].ID,
					sample[j].Addrs,
					peerstore.TempAddrTTL,
				)

				// Ping the peer.
				err = client.ping(sample[j].ID)
				if err != nil {
					continue
				}

				// Add the peer to the peer store.
				client.peerstore.SetAddrs(
					sample[j].ID,
					sample[j].Addrs,
					peerstore.ProviderAddrTTL,
				)

				// Update the routing table.
				client.table.Update(sample[j].ID)

			}

			// Decrement the query counter.
			queries--

		} else {

			// Remove the peer from the routing table.
			client.table.Remove(info.ID)

		}

	}

}

/**
 * File        : analytics.go
 * Description : Analytics reporting module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// Share analytics with core developers.
func (client *client) activateAnalytics() func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	go func() {

		type Report struct {
			Addrs           []string
			ClusterID       int
			Network         string
			NodeID          string
			Peers           int
			ProcessID       int
			InboundStreams  []string
			OutboundStreams []string
			Timestamp       int64
			UserData        string
			Version         string
		}

		for {

			// Check for a shutdown instruction.
			select {
			case <-notify:
				return
			default:
			}

			// Create a report.
			report := Report{
				ClusterID: client.config.ClusterID,
				Network:   string(client.config.Network),
				NodeID:    client.id.Pretty(),
				Peers:     client.table.Size(),
				ProcessID: client.config.ProcessID,
				Timestamp: time.Now().Unix(),
				UserData:  client.config.AnalyticsUserData,
				Version:   string(client.config.Version),
			}
			for _, addr := range client.host.Addrs() {
				report.Addrs = append(report.Addrs, addr.String())
			}
			for _, stream := range client.streamstore.InboundPeers() {
				report.InboundStreams = append(report.InboundStreams, stream.Pretty())
			}
			for _, stream := range client.streamstore.OutboundPeers() {
				report.OutboundStreams = append(report.OutboundStreams, stream.Pretty())
			}

			// Encode it.
			data, err := json.Marshal(&report)
			if err != nil {
				client.logger.Warning("Cannot encode report", err)
				return
			}

			// Create a request.
			req, err := http.NewRequest(
				"POST",
				client.config.AnalyticsURL,
				bytes.NewBuffer(data),
			)
			if err != nil {
				client.logger.Warning("Cannot create request", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			// Send it.
			sender := &http.Client{}
			resp, err := sender.Do(req)
			if err != nil {
				client.logger.Warning("Cannot send report", err)
			} else {
				resp.Body.Close()
			}

			// Wait.
			time.Sleep(client.config.AnalyticsInterval)

		}

	}()

	// Return the shutdown function.
	return shutdown

}

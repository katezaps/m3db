// +build integration

// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package integration

import (
	"testing"
	"time"

	"github.com/m3db/m3db/retention"
	"github.com/m3db/m3db/storage/namespace"
	"github.com/m3db/m3x/log"

	"github.com/stretchr/testify/require"
)

func TestRepairMerge(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // Just skip if we're doing a short run
	}

	// Test setups
	log := xlog.SimpleLogger
	namesp := namespace.NewMetadata(testNamespaces[0], namespace.NewOptions())
	opts := newTestOptions().
		SetNamespaces([]namespace.Metadata{namesp}).
		SetRepairerEnabled(true).
		SetRepairInterval(3 * time.Second).
		SetRepairThrottle(1 * time.Second).
		SetRepairTimeJitter(0 * time.Second).
		SetNumShards(128)

	retentionOpts := retention.NewOptions().
		SetBufferDrain(3 * time.Second).
		SetRetentionPeriod(12 * time.Hour).
		SetBlockSize(2 * time.Hour).
		SetBufferPast(10 * time.Minute).
		SetBufferFuture(2 * time.Minute)
	setupOpts := []multipleTestSetupsOptions{
		{
			disablePeersBootstrapper: true,
			enableRepairer:           true,
		},
		{
			disablePeersBootstrapper: true,
			enableRepairer:           true,
		},
	}
	setups, closeFn := newDefaultMultipleTestSetups(t, opts, retentionOpts, setupOpts)
	defer closeFn()

	// Write test data for first node
	now := setups[0].getNowFn()
	fpp0 := setups[0].storageOpts.CommitLogOptions().FilesystemOptions().FilePathPrefix()
	fpp1 := setups[1].storageOpts.CommitLogOptions().FilesystemOptions().FilePathPrefix()
	blockSize := setups[0].storageOpts.RetentionOptions().BlockSize()
	seriesMaps := generateTestDataByStart([]testData{
		{ids: []string{"foo", "bar"}, numPoints: 180, start: now.Add(-3 * blockSize)},
		{ids: []string{"foo", "baz"}, numPoints: 90, start: now.Add(-2 * blockSize)},
	})
	splitMaps := splitSeriesMaps(seriesMaps, 2)
	require.NoError(t, writeTestDataToDisk(t, namesp.ID(), setups[0], splitMaps[0]))
	require.NoError(t, writeTestDataToDisk(t, namesp.ID(), setups[1], splitMaps[1]))
	log.Debug("fs bootstrap input data written to disk")

	// Move time forward to trigger repairs
	later := now.Add(blockSize * 3).Add(30 * time.Second)
	setups[0].setNowFn(later)
	setups[1].setNowFn(later)

	// Start the servers with filesystem bootstrapper
	require.NoError(t, setups[0].startServer())
	require.NoError(t, setups[1].startServer())
	log.Debug("servers are now up")

	// Wait an empirically determined amount of time for repairs to finish
	waitTimeout := setups[1].storageOpts.RetentionOptions().BufferDrain() * 20
	require.NoError(t, waitUntilDataFlushed(fpp0, setups[0].shardSet, namesp.ID(), seriesMaps, waitTimeout, 2))
	require.NoError(t, waitUntilDataFlushed(fpp1, setups[1].shardSet, namesp.ID(), seriesMaps, waitTimeout, 2))

	// Stop the servers
	defer func() {
		setups.parallel(func(s *testSetup) {
			require.NoError(t, s.stopServer())
		})
		log.Debug("servers are now down")
	}()

	// Verify in-memory data match what we expect
	verifySeriesMaps(t, setups[1], namesp.ID(), seriesMaps)
	verifySeriesMaps(t, setups[0], namesp.ID(), seriesMaps)

	// Verify on-disk data match what we expect
	verifyFlushed(t, setups[0].shardSet, setups[0].storageOpts, namesp.ID(), 1, seriesMaps)
	verifyFlushed(t, setups[1].shardSet, setups[1].storageOpts, namesp.ID(), 1, seriesMaps)
}

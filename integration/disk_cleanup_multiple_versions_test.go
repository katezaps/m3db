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

	"github.com/stretchr/testify/require"
)

func TestMultileVersionsDiskCleanup(t *testing.T) {
	if testing.Short() {
		t.SkipNow() // Just skip if we're doing a short run
	}
	// Test setup
	testSetup, err := newTestSetup(newTestOptions())
	require.NoError(t, err)
	defer testSetup.close()

	testSetup.storageOpts = testSetup.storageOpts.SetRetentionOptions(
		testSetup.storageOpts.RetentionOptions().
			SetMaxVersionsRetained(1).
			SetBufferDrain(3 * time.Second).
			SetRetentionPeriod(6 * time.Hour))

	blockSize := testSetup.storageOpts.RetentionOptions().BlockSize()
	filePathPrefix := testSetup.storageOpts.CommitLogOptions().FilesystemOptions().FilePathPrefix()

	// Start the server
	log := testSetup.storageOpts.InstrumentOptions().Logger()
	log.Debug("disk cleanup test")
	require.NoError(t, testSetup.startServer())
	log.Debug("server is now up")

	// Stop the server
	defer func() {
		require.NoError(t, testSetup.stopServer())
		log.Debug("server is now down")
	}()

	// Now create some fileset files and commit logs
	shard := uint32(0)
	now := testSetup.getNowFn().Add(-2 * blockSize).Truncate(blockSize)
	fileTimes := []time.Time{now}
	createFilesetFiles(t, testSetup.storageOpts, testNamespaces[0], shard, fileTimes, []uint32{1, 2})

	// Check if files have been deleted
	waitTimeout := testSetup.storageOpts.RetentionOptions().BufferDrain() * 4
	require.NoError(t, waitUntilDataCleanedUp(filePathPrefix, testNamespaces[0], shard, now, 1, waitTimeout))
}

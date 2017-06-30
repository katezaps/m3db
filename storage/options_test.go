// Copyright (c) 2017 Uber Technologies, Inc.
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

package storage

import (
	"testing"

	"github.com/m3db/m3db/storage/namespace"
	"github.com/m3db/m3db/ts"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestOptionsValidateDefaults(t *testing.T) {
	dbOpts := testDatabaseOptions()
	require.NoError(t, dbOpts.Validate())
}

func TestOptionsValidateDefaultRepair(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbOpts := testDatabaseOptionsWithRepair(ctrl)
	require.NoError(t, dbOpts.Validate())
}

func TestOptionsValidateNilRegistry(t *testing.T) {
	dbOpts := testDatabaseOptions().
		SetNamespaceRegistry(nil)
	require.Error(t, dbOpts.Validate())
}

func TestOptionsValidateDifferentRetentionsSameNamespaces(t *testing.T) {
	newRetentionOpts := defaultTestRetentionOpts.
		SetBlockSize(defaultTestRetentionOpts.BlockSize() * 2)
	newNsOpts := namespace.NewOptions().SetRetentionOptions(newRetentionOpts)
	newRegistry := namespace.NewRegistry([]namespace.Metadata{
		namespace.NewMetadata(defaultTestNs1ID, newNsOpts)})

	dbOpts := testDatabaseOptions().
		SetNamespaceRegistry(newRegistry)
	require.Error(t, dbOpts.Validate())
}

func TestOptionsValidateRegistryDifferentNamespaces(t *testing.T) {
	newNsOpts := namespace.NewOptions().SetRetentionOptions(defaultTestRetentionOpts)
	newRegistry := namespace.NewRegistry([]namespace.Metadata{
		namespace.NewMetadata(ts.StringID("someString"), newNsOpts)})

	dbOpts := testDatabaseOptions().
		SetNamespaceRegistry(newRegistry)
	require.Error(t, dbOpts.Validate())
}
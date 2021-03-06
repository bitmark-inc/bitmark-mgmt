// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package api

const (
	// general error
	invalidValueErr = `Invalid value`

	// bitmarkConfig error
	bitmarkdConfigGetErr    = `Failed to get bitmarkd config`
	bitmarkdConfigUpdateErr = `Failed to update bitmarkd config`
	prooferdConfigGetErr    = `Failed to get prooferd config`
	prooferdConfigUpdateErr = `Failed to update prooferd config`

	// bitmark-webgui error
	setPasswordErr = `Failed to set up bitmark-webgui password`

	// bitmarkd api
	bitcoindStartSuccess = `start running bitcoind`
	bitcoindStopSuccess  = `stop running bitcoind`
	bitcoindStarted      = `started`
	bitcoindStopped      = `stopped`

	// bitcoind error
	bitcoindStartErr        = `Failed to start bitcoind`
	bitcoindStopErr         = `Failed to stop bitcoind`
	bitcoindAlreadyStartErr = `Already started bitcoind`
	bitcoindAlreadyStopErr  = `Already stoped bitcoind`
	bitcoindConnectErr      = `Failed to connect to bitcoind`
	bitcoindGetInfoErr      = `Failed to get bitcoind info`
	bitcoindGetConfigErr    = `Failed to get bitamrkd configuration`

	// bitmarkd api
	bitmarkdStartSuccess = `start running bitmarkd`
	bitmarkdStopSuccess  = `stop running bitmarkd`
	bitmarkdStarted      = `started`
	bitmarkdStopped      = `stopped`

	// bitmarkd error
	bitmarkdStartErr        = `Failed to start bitmarkd`
	bitmarkdStopErr         = `Failed to stop bitmarkd`
	bitmarkdAlreadyStartErr = `Already started bitmarkd`
	bitmarkdAlreadyStopErr  = `Already stoped bitmarkd`
	bitmarkdConnectErr      = `Failed to connect to bitmarkd`
	bitmarkdGetInfoErr      = `Failed to get bitmarkd info`
	bitmarkdGetConfigErr    = `Failed to get bitmarkd configuration`

	// prooferd api
	prooferdStartSuccess = `start running prooferd`
	prooferdStopSuccess  = `stop running prooferd`
	prooferdStarted      = `started`
	prooferdStopped      = `stopped`

	// prooferd error
	prooferdStartErr        = `Failed to start prooferd`
	prooferdStopErr         = `Failed to stop prooferd`
	prooferdAlreadyStartErr = `Already started prooferd`
	prooferdAlreadyStopErr  = `Already stoped prooferd`
	prooferdConnectErr      = `Failed to connect to prooferd`
	prooferdGetInfoErr      = `Failed to get prooferd info`
	prooferdGetConfigErr    = `Failed to get prooferd configuration`
	// login
	loginErr = `Failed to log in`

	// onestep
	onestepCliInfoErr = `Failed to get bitmark-cli info`
)

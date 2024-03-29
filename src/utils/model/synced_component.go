package model

type SyncedComponent string

const (
	SyncedComponentInteractions     SyncedComponent = "Interactions"
	SyncedComponentContracts        SyncedComponent = "Contracts"
	SyncedComponentForwarder        SyncedComponent = "Forwarder"
	SyncedComponentSequencer        SyncedComponent = "Sequencer"
	SyncedComponentRelayer          SyncedComponent = "Relayer"
	SyncedComponentRedstoneTxSyncer SyncedComponent = "RedstoneTxSyncer"
)

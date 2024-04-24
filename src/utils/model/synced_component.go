package model

type SyncedComponent string

const (
	SyncedComponentInteractions        SyncedComponent = "Interactions"
	SyncedComponentContracts           SyncedComponent = "Contracts"
	SyncedComponentForwarder           SyncedComponent = "Forwarder"
	SyncedComponentSequencer           SyncedComponent = "Sequencer"
	SyncedComponentRelayer             SyncedComponent = "Relayer"
	SyncedComponentWarpySyncerAvax     SyncedComponent = "WarpySyncerAvax"
	SyncedComponentWarpySyncerArbitrum SyncedComponent = "WarpySyncerArbitrum"
	SyncedComponentWarpySyncerMode     SyncedComponent = "WarpySyncerMode"
	SyncedComponentWarpySyncerManta    SyncedComponent = "WarpySyncerManta"
)

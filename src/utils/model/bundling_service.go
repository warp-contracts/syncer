package model

type BundlingService string

const (
	BundlingServiceIrys  BundlingService = "IRYS"
	BundlingServiceTurbo BundlingService = "TURBO"
)

func (self BundlingService) String() string {
	return string(self)
}

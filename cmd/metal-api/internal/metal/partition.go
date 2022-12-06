package metal

import (
	"errors"
	"strconv"
	"strings"
)

// A Partition represents a location.
type Partition struct {
	Base
	BootConfiguration          BootConfiguration `rethinkdb:"bootconfig" json:"bootconfig"`
	MgmtServiceAddress         string            `rethinkdb:"mgmtserviceaddr" json:"mgmtserviceaddr"`
	PrivateNetworkPrefixLength uint8             `rethinkdb:"privatenetworkprefixlength" json:"privatenetworkprefixlength"`
	WaitingPoolRange           ScalerRange       `rethinkdb:"waitingpoolrange" json:"waitingpoolrange"`
}

// BootConfiguration defines the metal-hammer initrd, kernel and commandline
type BootConfiguration struct {
	ImageURL    string `rethinkdb:"imageurl" json:"imageurl"`
	KernelURL   string `rethinkdb:"kernelurl" json:"kernelurl"`
	CommandLine string `rethinkdb:"commandline" json:"commandline"`
}

// Partitions is a list of partitions.
type Partitions []Partition

// PartitionMap is an indexed map of partitions
type PartitionMap map[string]Partition

// ByID creates an indexed map of partitions whre the id is the index.
func (sz Partitions) ByID() PartitionMap {
	res := make(PartitionMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}

type ScalerRange struct {
	WaitingPoolMinSize string
	WaitingPoolMaxSize string
}

func (r *ScalerRange) GetMin() (min int64, err error, inPercent bool) {
	if strings.HasSuffix(r.WaitingPoolMinSize, "%") {
		s := strings.Split(r.WaitingPoolMinSize, "%")[0]
		min, err := strconv.ParseInt(s, 10, 64)
		return min, err, true
	}
	min, err = strconv.ParseInt(r.WaitingPoolMinSize, 10, 64)
	return min, err, false
}

func (r *ScalerRange) GetMax() (min int64, err error, inPercent bool) {
	if strings.HasSuffix(r.WaitingPoolMaxSize, "%") {
		s := strings.Split(r.WaitingPoolMaxSize, "%")[0]
		min, err := strconv.ParseInt(s, 10, 64)
		return min, err, true
	}
	min, err = strconv.ParseInt(r.WaitingPoolMaxSize, 10, 64)
	return min, err, false
}

func (r *ScalerRange) IsDisabled() bool {
	return r.WaitingPoolMinSize == "" && r.WaitingPoolMaxSize == ""
}

func (r *ScalerRange) Validate() error {
	min, err, minInPercent := r.GetMin()
	if err != nil {
		return errors.New("could not parse minimum waiting pool size")
	}

	max, err, maxInPercent := r.GetMax()
	if err != nil {
		return errors.New("could not parse maximum waiting pool size")
	}

	if minInPercent != maxInPercent {
		return errors.New("minimum and maximum pool sizes must either be both in percent or both an absolute value")
	}

	if min < 0 || max < 0 {
		return errors.New("minimum and maximum waiting pool sizes must be greater or equal to 0")
	}

	if max < min {
		return errors.New("minimum waiting pool size must be less or equal to maximum pool size")
	}

	return nil
}

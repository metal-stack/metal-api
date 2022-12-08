package metal

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// A Partition represents a location.
type Partition struct {
	Base
	BootConfiguration          BootConfiguration `rethinkdb:"bootconfig" json:"bootconfig"`
	MgmtServiceAddress         string            `rethinkdb:"mgmtserviceaddr" json:"mgmtserviceaddr"`
	PrivateNetworkPrefixLength uint8             `rethinkdb:"privatenetworkprefixlength" json:"privatenetworkprefixlength"`
	WaitingPoolMinSize         string            `rethinkdb:"waitingpoolminsize" json:"waitingpoolminsize"`
	WaitingPoolMaxSize         string            `rethinkdb:"waitingpoolmaxsize" json:"waitingpoolmaxsize"`
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

func NewScalerRange(min, max string) (*ScalerRange, error) {
	r := &ScalerRange{WaitingPoolMinSize: min, WaitingPoolMaxSize: max}

	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *ScalerRange) Min(of int) (int64, error) {
	if strings.HasSuffix(r.WaitingPoolMinSize, "%") {
		s := strings.Split(r.WaitingPoolMinSize, "%")[0]
		min, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, err
		}

		return int64(math.Round(float64(of) * float64(min) / 100)), nil
	}

	return strconv.ParseInt(r.WaitingPoolMinSize, 10, 64)
}

func (r *ScalerRange) Max(of int) (int64, error) {
	if strings.HasSuffix(r.WaitingPoolMaxSize, "%") {
		s := strings.Split(r.WaitingPoolMaxSize, "%")[0]
		min, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, err
		}

		return int64(math.Round(float64(of) * float64(min) / 100)), nil
	}

	return strconv.ParseInt(r.WaitingPoolMaxSize, 10, 64)
}

func (r *ScalerRange) IsDisabled() bool {
	return r.WaitingPoolMinSize == "" && r.WaitingPoolMaxSize == ""
}

func (r *ScalerRange) Validate() error {
	if r.IsDisabled() {
		return nil
	}

	min, err := r.Min(100)
	if err != nil {
		return errors.New("could not parse minimum waiting pool size")
	}

	max, err := r.Max(100)
	if err != nil {
		return errors.New("could not parse maximum waiting pool size")
	}

	if strings.HasSuffix(r.WaitingPoolMinSize, "%") != strings.HasSuffix(r.WaitingPoolMaxSize, "%") {
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

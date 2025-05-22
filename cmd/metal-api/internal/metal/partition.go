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
	Labels                     map[string]string `rethinkdb:"labels" json:"labels"`
	DNSServers                 DNSServers        `rethinkdb:"dns_servers" json:"dns_servers"`
	NTPServers                 NTPServers        `rethinkdb:"ntp_servers" json:"ntp_servers"`
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

// ByID creates an indexed map of partitions where the id is the index.
func (sz Partitions) ByID() PartitionMap {
	res := make(PartitionMap)
	for i, s := range sz {
		res[s.ID] = sz[i]
	}
	return res
}

type ScalerRange struct {
	isDisabled bool
	inPercent  bool
	minSize    int
	maxSize    int
}

func NewScalerRange(min, max string) (*ScalerRange, error) {
	r := &ScalerRange{}
	if r.setDisabled(min, max); r.IsDisabled() {
		return r, nil
	}

	if err := r.setRange(min, max); err != nil {
		return nil, err
	}

	if err := r.Validate(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *ScalerRange) setDisabled(min, max string) {
	if min == "" && max == "" {
		r.isDisabled = true
		return
	}
	r.isDisabled = false
}

func (r *ScalerRange) IsDisabled() bool {
	return r.isDisabled
}

func (r *ScalerRange) setRange(min, max string) error {
	minSize, err, minInPercent := stringToSize(min)
	if err != nil {
		return errors.New("could not parse minimum waiting pool size")
	}

	maxSize, err, maxInPercent := stringToSize(max)
	if err != nil {
		return errors.New("could not parse maximum waiting pool size")
	}

	if minInPercent != maxInPercent {
		return errors.New("minimum and maximum pool sizes must either be both in percent or both an absolute value")
	}

	if minInPercent {
		r.inPercent = true
	} else {
		r.inPercent = false
	}

	r.minSize = int(minSize)
	r.maxSize = int(maxSize)
	return nil
}

func stringToSize(input string) (size int64, err error, inPercent bool) {
	s := input
	inPercent = false

	if strings.HasSuffix(input, "%") {
		s = strings.Split(input, "%")[0]
		inPercent = true
	}

	size, err = strconv.ParseInt(s, 10, 64)
	return size, err, inPercent
}

func (r *ScalerRange) Min(of int) int {
	if r.inPercent {
		return percentOf(r.minSize, of)
	}
	return r.minSize
}

func (r *ScalerRange) Max(of int) int {
	if r.inPercent {
		return percentOf(r.maxSize, of)
	}
	return r.maxSize
}

func percentOf(percent, of int) int {
	return int(math.Round(float64(of) * float64(percent) / 100))
}

func (r *ScalerRange) Validate() error {
	if r.IsDisabled() {
		return nil
	}

	if r.minSize <= 0 || r.maxSize <= 0 {
		return errors.New("minimum and maximum waiting pool sizes must be greater than 0")
	}

	if r.maxSize < r.minSize {
		return errors.New("minimum waiting pool size must be less or equal to maximum pool size")
	}

	return nil
}

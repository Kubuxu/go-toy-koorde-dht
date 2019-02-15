package koorde

import (
	"math/bits"

	"github.com/pkg/errors"
)

const KEY_SPACE = 256

type koordeConfig struct {
	degree           uint
	degreeShift      uint
	backupSuccessors int
}

func Config(degree uint, backupSuccessors int) (koordeConfig, error) {
	cfg := koordeConfig{}
	if degree < 2 {
		return cfg, errors.New("degree can't be smaller than 2")
	}
	if backupSuccessors < 0 {
		return cfg, errors.New("backupSuccessors can't be smaller than 0")
	}
	if (degree & (degree - 1)) != 0 {
		return cfg, errors.New("degree has to be power of two")
	}

	cfg.degree = degree
	cfg.backupSuccessors = backupSuccessors
	cfg.degreeShift = uint(bits.TrailingZeros(cfg.degree))

	return cfg, nil
}
